package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	PayloadFormatRaw      = "raw"
	PayloadFormatEnvelope = "envelope"

	publishTokenSecureKey = "publishServiceAccountToken"

	defaultPublishQoS             = 1
	defaultPublishOrigin          = "grafana"
	defaultMaxPayloadBytes        = 256
	defaultTeamCacheSeconds       = 60
	defaultGrafanaURL             = "http://localhost:3000"
	defaultRateLimitPerMinute     = 10
	maxPublishTopicBytes          = 1024
	maxPublishRequestBodyBytes    = 4096
	maxConfiguredMaxPayloadBytes  = 65536
	maxConfiguredTeamCacheSeconds = 3600
)

var originPattern = regexp.MustCompile(`^[a-z0-9_.-]{1,32}$`)

// PublishSettings configures publishing through the datasource resource API.
// They live below jsonData.publish; the service account token is read from
// secureJsonData.publishServiceAccountToken.
type PublishSettings struct {
	Enabled       bool     `json:"enabled"`
	AllowedTeams  []string `json:"allowedTeams"`
	AllowedTopics []string `json:"allowedTopics"`
	DeniedTopics  []string `json:"deniedTopics"`
	QoS           *int     `json:"qos"`
	Retain        bool     `json:"retain"`
	PayloadFormat string   `json:"payloadFormat"`
	Origin        string   `json:"origin"`
	// MaxPayloadBytes limits the published MQTT payload, including the envelope.
	MaxPayloadBytes  int    `json:"maxPayloadBytes"`
	TeamCacheSeconds int    `json:"teamCacheSeconds"`
	GrafanaURL       string `json:"grafanaUrl"`
	// RateLimitPerMinute limits publish requests per user; 0 uses the default, a negative value disables the limit.
	RateLimitPerMinute int `json:"rateLimitPerMinute"`

	ServiceAccountToken string `json:"-"`
}

// ParsePublishSettings reads and validates jsonData.publish. A disabled
// configuration is returned without validation.
func ParsePublishSettings(jsonData []byte, secureJSONData map[string]string) (PublishSettings, error) {
	var wrapper struct {
		Publish *PublishSettings `json:"publish"`
	}
	if len(jsonData) > 0 {
		if err := json.Unmarshal(jsonData, &wrapper); err != nil {
			return PublishSettings{}, fmt.Errorf("invalid publish settings: %w", err)
		}
	}
	if wrapper.Publish == nil || !wrapper.Publish.Enabled {
		return PublishSettings{}, nil
	}

	settings := *wrapper.Publish
	settings.ServiceAccountToken = secureJSONData[publishTokenSecureKey]
	settings.applyDefaults()
	if err := settings.validate(); err != nil {
		return settings, err
	}
	return settings, nil
}

func (s *PublishSettings) applyDefaults() {
	if s.QoS == nil {
		qos := defaultPublishQoS
		s.QoS = &qos
	}
	if s.PayloadFormat == "" {
		s.PayloadFormat = PayloadFormatRaw
	}
	if s.Origin == "" {
		s.Origin = defaultPublishOrigin
	}
	if s.MaxPayloadBytes == 0 {
		s.MaxPayloadBytes = defaultMaxPayloadBytes
	}
	if s.TeamCacheSeconds == 0 {
		s.TeamCacheSeconds = defaultTeamCacheSeconds
	}
	if s.GrafanaURL == "" {
		s.GrafanaURL = defaultGrafanaURL
	}
	if s.RateLimitPerMinute == 0 {
		s.RateLimitPerMinute = defaultRateLimitPerMinute
	}
	s.AllowedTeams = trimNonEmpty(s.AllowedTeams)
	s.AllowedTopics = trimNonEmpty(s.AllowedTopics)
	s.DeniedTopics = trimNonEmpty(s.DeniedTopics)
	s.GrafanaURL = strings.TrimRight(s.GrafanaURL, "/")
}

func (s *PublishSettings) validate() error {
	var errs []error
	if len(s.AllowedTeams) == 0 {
		errs = append(errs, errors.New("allowedTeams must name at least one team"))
	}
	if len(s.AllowedTopics) == 0 {
		errs = append(errs, errors.New("allowedTopics must contain at least one pattern"))
	}
	for _, pattern := range append(append([]string{}, s.AllowedTopics...), s.DeniedTopics...) {
		if err := validateTopicFilter(pattern); err != nil {
			errs = append(errs, fmt.Errorf("invalid topic pattern %q: %w", pattern, err))
		}
	}
	if *s.QoS < 0 || *s.QoS > 2 {
		errs = append(errs, fmt.Errorf("qos must be 0, 1 or 2, got %d", *s.QoS))
	}
	if s.PayloadFormat != PayloadFormatRaw && s.PayloadFormat != PayloadFormatEnvelope {
		errs = append(errs, fmt.Errorf("payloadFormat must be %q or %q, got %q", PayloadFormatRaw, PayloadFormatEnvelope, s.PayloadFormat))
	}
	if !originPattern.MatchString(s.Origin) {
		errs = append(errs, fmt.Errorf("origin must match %s", originPattern.String()))
	}
	if s.MaxPayloadBytes < 1 || s.MaxPayloadBytes > maxConfiguredMaxPayloadBytes {
		errs = append(errs, fmt.Errorf("maxPayloadBytes must be between 1 and %d", maxConfiguredMaxPayloadBytes))
	}
	if s.TeamCacheSeconds < 1 || s.TeamCacheSeconds > maxConfiguredTeamCacheSeconds {
		errs = append(errs, fmt.Errorf("teamCacheSeconds must be between 1 and %d", maxConfiguredTeamCacheSeconds))
	}
	if u, err := url.Parse(s.GrafanaURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, fmt.Errorf("grafanaUrl must be an absolute http(s) URL, got %q", s.GrafanaURL))
	}
	if s.ServiceAccountToken == "" {
		errs = append(errs, fmt.Errorf("secureJsonData.%s is required to look up team membership", publishTokenSecureKey))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid publish settings: %w", errors.Join(errs...))
	}
	return nil
}

func trimNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// validateTopicFilter checks an MQTT topic filter: "+" must occupy a whole
// level and "#" must be the whole last level.
func validateTopicFilter(filter string) error {
	if filter == "" {
		return errors.New("empty pattern")
	}
	levels := strings.Split(filter, "/")
	for i, level := range levels {
		if strings.Contains(level, "#") && (level != "#" || i != len(levels)-1) {
			return errors.New(`"#" must be the last level on its own`)
		}
		if strings.Contains(level, "+") && level != "+" {
			return errors.New(`"+" must be a level on its own`)
		}
	}
	return nil
}

// validatePublishTopic checks a concrete topic name for publishing.
func validatePublishTopic(topic string) error {
	switch {
	case topic == "":
		return errors.New("topic is required")
	case len(topic) > maxPublishTopicBytes:
		return fmt.Errorf("topic is longer than %d bytes", maxPublishTopicBytes)
	case !utf8.ValidString(topic):
		return errors.New("topic is not valid UTF-8")
	case strings.ContainsAny(topic, "+#\x00"):
		return errors.New("topic must not contain wildcards or NUL")
	case strings.HasPrefix(topic, "$"):
		return errors.New("topics starting with $ are reserved")
	}
	return nil
}

// topicMatches implements MQTT topic filter matching for a concrete topic.
func topicMatches(filter, topic string) bool {
	filterLevels := strings.Split(filter, "/")
	topicLevels := strings.Split(topic, "/")
	for i, level := range filterLevels {
		if level == "#" {
			return true
		}
		if i >= len(topicLevels) {
			return false
		}
		if level != "+" && level != topicLevels[i] {
			return false
		}
	}
	return len(filterLevels) == len(topicLevels)
}

// topicAllowed returns true when the topic matches an allowed pattern and no
// denied pattern. Denied patterns take precedence.
func (s *PublishSettings) topicAllowed(topic string) bool {
	for _, filter := range s.DeniedTopics {
		if topicMatches(filter, topic) {
			return false
		}
	}
	for _, filter := range s.AllowedTopics {
		if topicMatches(filter, topic) {
			return true
		}
	}
	return false
}
