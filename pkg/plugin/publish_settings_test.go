package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopicMatches(t *testing.T) {
	tests := []struct {
		filter, topic string
		want          bool
	}{
		{"pk/control/+/+", "pk/control/dashboard/flow_target", true},
		{"pk/control/+/+", "pk/control/pid/cc_target/target", false},
		{"pk/control/+/+", "pk/control/dashboard", false},
		{"pk/control/#", "pk/control/pid/cc_target/target", true},
		{"pk/control/#", "pk/control", true},
		{"pk/control/raw/#", "pk/control/raw/62", true},
		{"pk/control/raw/#", "pk/control/rawx/62", false},
		{"#", "anything/at/all", true},
		{"a/b", "a/b", true},
		{"a/b", "a/b/c", false},
		{"+/b", "/b", true},
	}
	for _, tt := range tests {
		t.Run(tt.filter+" "+tt.topic, func(t *testing.T) {
			require.Equal(t, tt.want, topicMatches(tt.filter, tt.topic))
		})
	}
}

func TestTopicAllowedDeniedTakesPrecedence(t *testing.T) {
	settings := PublishSettings{
		AllowedTopics: []string{"pk/control/#"},
		DeniedTopics:  []string{"pk/control/raw/#"},
	}
	require.True(t, settings.topicAllowed("pk/control/dashboard/flow_target"))
	require.False(t, settings.topicAllowed("pk/control/raw/62"))
	require.False(t, settings.topicAllowed("pk/view/dashboard"))
}

func TestValidateTopicFilter(t *testing.T) {
	for _, valid := range []string{"#", "+", "a/+/c", "a/#", "+/+/#"} {
		require.NoError(t, validateTopicFilter(valid), valid)
	}
	for _, invalid := range []string{"", "a/#/c", "a#", "a/b+", "a/+b/c"} {
		require.Error(t, validateTopicFilter(invalid), invalid)
	}
}

func TestValidatePublishTopic(t *testing.T) {
	require.NoError(t, validatePublishTopic("pk/control/dashboard/flow_target"))
	for _, invalid := range []string{"", "pk/+/x", "pk/#", "$SYS/x", "a\x00b", "\xff"} {
		require.Error(t, validatePublishTopic(invalid), invalid)
	}
}

func TestParsePublishSettings(t *testing.T) {
	secure := map[string]string{"publishServiceAccountToken": "token"}

	t.Run("missing or disabled publish block disables publishing", func(t *testing.T) {
		settings, err := ParsePublishSettings([]byte(`{"uri":"tcp://broker"}`), nil)
		require.NoError(t, err)
		require.False(t, settings.Enabled)

		settings, err = ParsePublishSettings([]byte(`{"publish":{"enabled":false,"qos":7}}`), nil)
		require.NoError(t, err)
		require.False(t, settings.Enabled)
	})

	t.Run("defaults are applied", func(t *testing.T) {
		settings, err := ParsePublishSettings([]byte(`{"publish":{"enabled":true,"allowedTeams":["Writers"],"allowedTopics":["a/+"]}}`), secure)
		require.NoError(t, err)
		require.Equal(t, 1, *settings.QoS)
		require.False(t, settings.Retain)
		require.Equal(t, PayloadFormatRaw, settings.PayloadFormat)
		require.Equal(t, "grafana", settings.Origin)
		require.Equal(t, 256, settings.MaxPayloadBytes)
		require.Equal(t, 60, settings.TeamCacheSeconds)
		require.Equal(t, "http://localhost:3000", settings.GrafanaURL)
		require.Equal(t, 10, settings.RateLimitPerMinute)
		require.Equal(t, "token", settings.ServiceAccountToken)
	})

	t.Run("qos 0 is kept", func(t *testing.T) {
		settings, err := ParsePublishSettings([]byte(`{"publish":{"enabled":true,"qos":0,"allowedTeams":["W"],"allowedTopics":["a"]}}`), secure)
		require.NoError(t, err)
		require.Equal(t, 0, *settings.QoS)
	})

	t.Run("invalid settings are reported", func(t *testing.T) {
		_, err := ParsePublishSettings([]byte(`{"publish":{"enabled":true,"qos":3,"payloadFormat":"xml","origin":"Grafana UI","allowedTopics":["a/#/b"],"grafanaUrl":"localhost"}}`), nil)
		require.Error(t, err)
		for _, fragment := range []string{"allowedTeams", "a/#/b", "qos", "payloadFormat", "origin", "grafanaUrl", "publishServiceAccountToken"} {
			require.Contains(t, err.Error(), fragment)
		}
	})

	t.Run("invalid JSON is reported", func(t *testing.T) {
		_, err := ParsePublishSettings([]byte(`{"publish":{"enabled":"yes"}}`), secure)
		require.Error(t, err)
	})
}
