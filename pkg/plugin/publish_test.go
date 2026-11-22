package plugin_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

type fakeTeamChecker struct {
	members   map[string]bool
	err       error
	verifyErr error
	calls     int
}

func (f *fakeTeamChecker) IsMember(_ context.Context, login string) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	return f.members[login], nil
}

func (f *fakeTeamChecker) Verify(_ context.Context) error { return f.verifyErr }

type capturedResponse struct {
	status int
	body   map[string]any
}

func callResource(t *testing.T, ds *plugin.MQTTDatasource, method, path, login, body string) capturedResponse {
	t.Helper()
	req := &backend.CallResourceRequest{Method: method, Path: path, Body: []byte(body)}
	if login != "" {
		req.PluginContext.User = &backend.User{Login: login}
	}
	var response capturedResponse
	err := ds.CallResource(context.Background(), req, backend.CallResourceResponseSenderFunc(func(res *backend.CallResourceResponse) error {
		response.status = res.Status
		require.NoError(t, json.Unmarshal(res.Body, &response.body))
		return nil
	}))
	require.NoError(t, err)
	return response
}

func publishSettings(t *testing.T, publish string) plugin.PublishSettings {
	t.Helper()
	settings, err := plugin.ParsePublishSettings(
		[]byte(`{"publish":`+publish+`}`),
		map[string]string{"publishServiceAccountToken": "token"},
	)
	require.NoError(t, err)
	return settings
}

const defaultTestPublish = `{"enabled":true,"allowedTeams":["Writers"],"allowedTopics":["pk/control/+/+"],"deniedTopics":["pk/control/raw/#"]}`

func newPublishingDatasource(t *testing.T, publish string, teams *fakeTeamChecker) (*plugin.MQTTDatasource, *fakeMQTTClient) {
	t.Helper()
	client := &fakeMQTTClient{connected: true}
	ds := plugin.NewMQTTDatasource(client, "uid")
	ds.Publisher = plugin.NewPublisher(publishSettings(t, publish), nil, teams)
	return ds, client
}

func TestPublishResource(t *testing.T) {
	writers := func() *fakeTeamChecker { return &fakeTeamChecker{members: map[string]bool{"til": true}} }

	t.Run("publishes raw value for team member", func(t *testing.T) {
		ds, client := newPublishingDatasource(t, defaultTestPublish, writers())

		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"pk/control/dashboard/flow_target","value":"65"}`)

		require.Equal(t, http.StatusOK, res.status)
		require.Equal(t, true, res.body["published"])
		require.Equal(t, []fakePublish{{topic: "pk/control/dashboard/flow_target", qos: 1, retain: false, payload: "65"}}, client.published)
	})

	t.Run("publishes envelope with origin and signed-in user", func(t *testing.T) {
		ds, client := newPublishingDatasource(t, `{"enabled":true,"allowedTeams":["Writers"],"allowedTopics":["pk/control/+/+"],"payloadFormat":"envelope","origin":"grafana","qos":0,"retain":true}`, writers())

		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"pk/control/dashboard/flow_target","value":65.5}`)

		require.Equal(t, http.StatusOK, res.status)
		require.Len(t, client.published, 1)
		require.Equal(t, byte(0), client.published[0].qos)
		require.True(t, client.published[0].retain)
		require.JSONEq(t, `{"value":"65.5","origin":"grafana","user":"til"}`, client.published[0].payload)
	})

	t.Run("rejections", func(t *testing.T) {
		tests := []struct {
			name   string
			login  string
			body   string
			status int
		}{
			{"no user", "", `{"topic":"pk/control/a/b","value":"1"}`, http.StatusForbidden},
			{"user not in team", "guest", `{"topic":"pk/control/a/b","value":"1"}`, http.StatusForbidden},
			{"topic not allowed", "til", `{"topic":"pk/view/a/b","value":"1"}`, http.StatusForbidden},
			{"denied topic", "til", `{"topic":"pk/control/raw/62","value":"1"}`, http.StatusForbidden},
			{"wildcard topic", "til", `{"topic":"pk/control/+/b","value":"1"}`, http.StatusBadRequest},
			{"missing value", "til", `{"topic":"pk/control/a/b"}`, http.StatusBadRequest},
			{"null value", "til", `{"topic":"pk/control/a/b","value":null}`, http.StatusBadRequest},
			{"bool value", "til", `{"topic":"pk/control/a/b","value":true}`, http.StatusBadRequest},
			{"empty value", "til", `{"topic":"pk/control/a/b","value":""}`, http.StatusBadRequest},
			{"control character", "til", `{"topic":"pk/control/a/b","value":"1\n2"}`, http.StatusBadRequest},
			{"unknown field", "til", `{"topic":"pk/control/a/b","value":"1","retain":true}`, http.StatusBadRequest},
			{"invalid JSON", "til", `{"topic":`, http.StatusBadRequest},
			{"payload too large", "til", `{"topic":"pk/control/a/b","value":"` + strings.Repeat("x", 300) + `"}`, http.StatusBadRequest},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ds, client := newPublishingDatasource(t, defaultTestPublish, writers())

				res := callResource(t, ds, http.MethodPost, "publish", tt.login, tt.body)

				require.Equal(t, tt.status, res.status)
				require.Equal(t, false, res.body["published"])
				require.NotEmpty(t, res.body["error"])
				if tt.name == "payload too large" {
					require.Equal(t, "payload too large", res.body["error"])
				}
				require.Empty(t, client.published)
			})
		}
	})

	t.Run("payload size limit applies to the envelope", func(t *testing.T) {
		ds, client := newPublishingDatasource(t, `{"enabled":true,"allowedTeams":["W"],"allowedTopics":["a"],"payloadFormat":"envelope","maxPayloadBytes":40}`, writers())

		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"a","value":"1234567890"}`)

		require.Equal(t, http.StatusBadRequest, res.status)
		require.Empty(t, client.published)
	})

	t.Run("fails closed when team lookup fails", func(t *testing.T) {
		ds, client := newPublishingDatasource(t, defaultTestPublish, &fakeTeamChecker{err: errors.New("boom")})

		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"pk/control/a/b","value":"1"}`)

		require.Equal(t, http.StatusServiceUnavailable, res.status)
		require.Empty(t, client.published)
	})

	t.Run("reports broker problems as unavailable", func(t *testing.T) {
		ds, client := newPublishingDatasource(t, defaultTestPublish, writers())
		client.connected = false
		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"pk/control/a/b","value":"1"}`)
		require.Equal(t, http.StatusServiceUnavailable, res.status)

		client.connected = true
		client.publishErr = errors.New("broker said no")
		res = callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"pk/control/a/b","value":"1"}`)
		require.Equal(t, http.StatusServiceUnavailable, res.status)
	})

	t.Run("rate limits per user", func(t *testing.T) {
		ds, client := newPublishingDatasource(t, `{"enabled":true,"allowedTeams":["W"],"allowedTopics":["a"],"rateLimitPerMinute":2}`, writers())

		for i := 0; i < 2; i++ {
			require.Equal(t, http.StatusOK, callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"a","value":"1"}`).status)
		}
		require.Equal(t, http.StatusTooManyRequests, callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"a","value":"1"}`).status)
		require.Len(t, client.published, 2)
	})

	t.Run("disabled publishing is forbidden", func(t *testing.T) {
		client := &fakeMQTTClient{connected: true}
		ds := plugin.NewMQTTDatasource(client, "uid")

		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"a","value":"1"}`)

		require.Equal(t, http.StatusForbidden, res.status)
		require.Empty(t, client.published)
	})

	t.Run("misconfigured publishing is unavailable", func(t *testing.T) {
		client := &fakeMQTTClient{connected: true}
		ds := plugin.NewMQTTDatasource(client, "uid")
		ds.Publisher = plugin.NewPublisher(plugin.PublishSettings{Enabled: true}, errors.New("bad config"), nil)

		res := callResource(t, ds, http.MethodPost, "publish", "til", `{"topic":"a","value":"1"}`)

		require.Equal(t, http.StatusServiceUnavailable, res.status)
	})

	t.Run("only POST is accepted and unknown paths are not found", func(t *testing.T) {
		ds, _ := newPublishingDatasource(t, defaultTestPublish, writers())
		require.Equal(t, http.StatusMethodNotAllowed, callResource(t, ds, http.MethodGet, "publish", "til", "").status)
		require.Equal(t, http.StatusMethodNotAllowed, callResource(t, ds, http.MethodPost, "publish/permissions", "til", "").status)
		require.Equal(t, http.StatusNotFound, callResource(t, ds, http.MethodGet, "other", "til", "").status)
	})
}

func TestPublishPermissionsResource(t *testing.T) {
	t.Run("team member can publish", func(t *testing.T) {
		ds, _ := newPublishingDatasource(t, defaultTestPublish, &fakeTeamChecker{members: map[string]bool{"til": true}})
		res := callResource(t, ds, http.MethodGet, "publish/permissions", "til", "")
		require.Equal(t, http.StatusOK, res.status)
		require.Equal(t, true, res.body["canPublish"])
	})

	t.Run("others cannot publish", func(t *testing.T) {
		ds, _ := newPublishingDatasource(t, defaultTestPublish, &fakeTeamChecker{members: map[string]bool{"til": true}})
		res := callResource(t, ds, http.MethodGet, "publish/permissions", "guest", "")
		require.Equal(t, http.StatusOK, res.status)
		require.Equal(t, false, res.body["canPublish"])
	})

	t.Run("disabled publishing", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{connected: true}, "uid")
		res := callResource(t, ds, http.MethodGet, "publish/permissions", "til", "")
		require.Equal(t, http.StatusOK, res.status)
		require.Equal(t, false, res.body["canPublish"])
	})

	t.Run("team lookup failure", func(t *testing.T) {
		ds, _ := newPublishingDatasource(t, defaultTestPublish, &fakeTeamChecker{err: errors.New("boom")})
		res := callResource(t, ds, http.MethodGet, "publish/permissions", "til", "")
		require.Equal(t, http.StatusServiceUnavailable, res.status)
		require.Equal(t, false, res.body["canPublish"])
	})
}

func TestCheckHealthWithPublishing(t *testing.T) {
	t.Run("reports publishing enabled", func(t *testing.T) {
		ds, _ := newPublishingDatasource(t, defaultTestPublish, &fakeTeamChecker{})
		res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
		require.NoError(t, err)
		require.Equal(t, backend.HealthStatusOk, res.Status)
		require.Equal(t, "MQTT Connected, publishing enabled", res.Message)
	})

	t.Run("reports team API problems", func(t *testing.T) {
		ds, _ := newPublishingDatasource(t, defaultTestPublish, &fakeTeamChecker{verifyErr: errors.New("team \"Writers\" not found")})
		res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
		require.NoError(t, err)
		require.Equal(t, backend.HealthStatusError, res.Status)
		require.Contains(t, res.Message, "Writers")
	})

	t.Run("reports invalid configuration", func(t *testing.T) {
		ds := plugin.NewMQTTDatasource(&fakeMQTTClient{connected: true}, "uid")
		ds.Publisher = plugin.NewPublisher(plugin.PublishSettings{Enabled: true}, errors.New("invalid publish settings: qos"), nil)
		res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
		require.NoError(t, err)
		require.Equal(t, backend.HealthStatusError, res.Status)
		require.Contains(t, res.Message, "qos")
	})
}
