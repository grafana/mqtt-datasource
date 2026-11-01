package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const (
	resourcePublish            = "publish"
	resourcePublishPermissions = "publish/permissions"
)

// Publisher handles authorised MQTT publishing for one datasource instance.
type Publisher struct {
	settings  PublishSettings
	configErr error
	teams     TeamChecker
	limiter   *rateLimiter
}

// NewPublisher creates a publisher. A non-nil configErr keeps publishing
// disabled and is reported by the health check and resource API.
func NewPublisher(settings PublishSettings, configErr error, teams TeamChecker) *Publisher {
	p := &Publisher{settings: settings, configErr: configErr, teams: teams}
	if settings.RateLimitPerMinute > 0 {
		p.limiter = newRateLimiter(settings.RateLimitPerMinute, time.Minute, time.Now)
	}
	return p
}

func newPublisherFromInstanceSettings(s backend.DataSourceInstanceSettings) *Publisher {
	settings, err := ParsePublishSettings(s.JSONData, s.DecryptedSecureJSONData)
	if err != nil {
		log.DefaultLogger.Error("MQTT publishing disabled", "error", err)
		return NewPublisher(settings, err, nil)
	}
	if !settings.Enabled {
		return NewPublisher(settings, nil, nil)
	}
	teams := NewGrafanaTeamChecker(
		settings.GrafanaURL,
		settings.ServiceAccountToken,
		settings.AllowedTeams,
		time.Duration(settings.TeamCacheSeconds)*time.Second,
	)
	return NewPublisher(settings, nil, teams)
}

func (p *Publisher) enabled() bool {
	return p != nil && (p.settings.Enabled || p.configErr != nil)
}

type publishRequest struct {
	Topic string          `json:"topic"`
	Value json.RawMessage `json:"value"`
}

type resourceError struct {
	status  int
	message string
}

func (e *resourceError) Error() string { return e.message }

func newResourceError(status int, message string) *resourceError {
	return &resourceError{status: status, message: message}
}

func (ds *MQTTDatasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	switch req.Path {
	case resourcePublish:
		if req.Method != http.MethodPost {
			return sendJSON(sender, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		}
		return ds.handlePublish(ctx, req, sender)
	case resourcePublishPermissions:
		if req.Method != http.MethodGet {
			return sendJSON(sender, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		}
		return ds.handlePublishPermissions(ctx, req, sender)
	default:
		return sendJSON(sender, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (ds *MQTTDatasource) handlePublishPermissions(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	p := ds.Publisher
	if !p.enabled() {
		return sendJSON(sender, http.StatusOK, map[string]any{"canPublish": false, "reason": "publishing is disabled"})
	}
	if p.configErr != nil {
		return sendJSON(sender, http.StatusServiceUnavailable, map[string]any{"canPublish": false, "error": "publishing is misconfigured"})
	}
	login := userLogin(req.PluginContext)
	allowed, err := p.teams.IsMember(ctx, login)
	if err != nil {
		log.DefaultLogger.FromContext(ctx).Error("MQTT publish permission check failed", "user", login, "error", err)
		return sendJSON(sender, http.StatusServiceUnavailable, map[string]any{"canPublish": false, "error": "team membership could not be checked"})
	}
	if !allowed {
		return sendJSON(sender, http.StatusOK, map[string]any{"canPublish": false, "reason": "user is not in an allowed team"})
	}
	return sendJSON(sender, http.StatusOK, map[string]any{"canPublish": true})
}

func (ds *MQTTDatasource) handlePublish(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	logger := log.DefaultLogger.FromContext(ctx)
	login := userLogin(req.PluginContext)

	topic, value, err := ds.publish(ctx, login, req.Body)
	if err != nil {
		var resErr *resourceError
		if !errors.As(err, &resErr) {
			resErr = newResourceError(http.StatusInternalServerError, "internal error")
		}
		logger.Warn("MQTT publish rejected", "user", login, "topic", topic, "value", value, "status", resErr.status, "reason", err.Error())
		return sendJSON(sender, resErr.status, map[string]any{"published": false, "error": resErr.message})
	}
	logger.Info("MQTT publish accepted", "user", login, "topic", topic, "value", value)
	return sendJSON(sender, http.StatusOK, map[string]any{"published": true})
}

// publish validates and authorises one request and publishes it. It returns
// the topic and value as far as they could be parsed, for logging.
func (ds *MQTTDatasource) publish(ctx context.Context, login string, body []byte) (string, string, error) {
	p := ds.Publisher
	if !p.enabled() {
		return "", "", newResourceError(http.StatusForbidden, "publishing is disabled")
	}
	if p.configErr != nil {
		return "", "", newResourceError(http.StatusServiceUnavailable, "publishing is misconfigured")
	}
	if login == "" {
		return "", "", newResourceError(http.StatusForbidden, "no signed-in user")
	}
	if len(body) > maxPublishRequestBodyBytes {
		return "", "", newResourceError(http.StatusBadRequest, "request body too large")
	}

	var request publishRequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return "", "", newResourceError(http.StatusBadRequest, "request body must be a JSON object with topic and value")
	}
	topic := request.Topic
	if err := validatePublishTopic(topic); err != nil {
		return topic, "", newResourceError(http.StatusBadRequest, err.Error())
	}
	value, err := parsePublishValue(request.Value)
	if err != nil {
		return topic, "", newResourceError(http.StatusBadRequest, err.Error())
	}
	if !p.settings.topicAllowed(topic) {
		return topic, value, newResourceError(http.StatusForbidden, "topic is not allowed")
	}

	allowed, err := p.teams.IsMember(ctx, login)
	if err != nil {
		log.DefaultLogger.FromContext(ctx).Error("MQTT publish team lookup failed", "error", err)
		return topic, value, newResourceError(http.StatusServiceUnavailable, "team membership could not be checked")
	}
	if !allowed {
		return topic, value, newResourceError(http.StatusForbidden, "user is not in an allowed team")
	}
	if p.limiter != nil && !p.limiter.allow(login) {
		return topic, value, newResourceError(http.StatusTooManyRequests, "too many publish requests")
	}

	payload, err := p.payload(value, login)
	if err != nil {
		return topic, value, err
	}
	if !ds.Client.IsConnected() {
		return topic, value, newResourceError(http.StatusServiceUnavailable, "MQTT broker is not connected")
	}
	if err := ds.Client.Publish(topic, byte(*p.settings.QoS), p.settings.Retain, payload); err != nil {
		log.DefaultLogger.FromContext(ctx).Error("MQTT publish failed", "topic", topic, "error", err)
		return topic, value, newResourceError(http.StatusServiceUnavailable, "publishing to the MQTT broker failed")
	}
	return topic, value, nil
}

// parsePublishValue accepts a JSON string or number and returns its text.
func parsePublishValue(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", errors.New("value is required")
	}
	var value string
	switch trimmed[0] {
	case '"':
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return "", errors.New("value must be a string or number")
		}
	default:
		var number json.Number
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&number); err != nil {
			return "", errors.New("value must be a string or number")
		}
		value = number.String()
	}
	if value == "" {
		return "", errors.New("value must not be empty")
	}
	if !utf8.ValidString(value) {
		return "", errors.New("value is not valid UTF-8")
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", errors.New("value must not contain control characters")
	}
	return value, nil
}

func (p *Publisher) payload(value, login string) ([]byte, error) {
	var payload []byte
	switch p.settings.PayloadFormat {
	case PayloadFormatEnvelope:
		encoded, err := json.Marshal(struct {
			Value  string `json:"value"`
			Origin string `json:"origin"`
			User   string `json:"user"`
		}{Value: value, Origin: p.settings.Origin, User: login})
		if err != nil {
			return nil, newResourceError(http.StatusInternalServerError, "encoding payload failed")
		}
		payload = encoded
	default:
		payload = []byte(value)
	}
	if len(payload) > p.settings.MaxPayloadBytes {
		return nil, newResourceError(http.StatusBadRequest, "payload too large")
	}
	return payload, nil
}

func (p *Publisher) checkHealth(ctx context.Context) error {
	if p.configErr != nil {
		return p.configErr
	}
	return p.teams.Verify(ctx)
}

func userLogin(pluginContext backend.PluginContext) string {
	if pluginContext.User == nil {
		return ""
	}
	return pluginContext.User.Login
}

func sendJSON(sender backend.CallResourceResponseSender, status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return sender.Send(&backend.CallResourceResponse{
		Status:  status,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    encoded,
	})
}

type rateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu     sync.Mutex
	events map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration, now func() time.Time) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, now: now, events: make(map[string][]time.Time)}
}

func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	cutoff := now.Add(-r.window)
	recent := r.events[key][:0]
	for _, event := range r.events[key] {
		if event.After(cutoff) {
			recent = append(recent, event)
		}
	}
	if len(recent) >= r.limit {
		r.events[key] = recent
		return false
	}
	r.events[key] = append(recent, now)
	return true
}
