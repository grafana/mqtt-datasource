package plugin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func newTeamAPI(t *testing.T, requests *atomic.Int32, healthy *atomic.Bool) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !healthy.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/teams/search":
			switch r.URL.Query().Get("name") {
			case "Writers":
				_, _ = w.Write([]byte(`{"teams":[{"id":7,"name":"Writers"}]}`))
			case "Ops":
				_, _ = w.Write([]byte(`{"teams":[{"id":8,"name":"Ops"}]}`))
			default:
				_, _ = w.Write([]byte(`{"teams":[]}`))
			}
		case "/api/teams/7/members":
			_, _ = w.Write([]byte(`[{"login":"til","email":"til@example.org"}]`))
		case "/api/teams/8/members":
			_, _ = w.Write([]byte(`[{"login":"ops"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGrafanaTeamChecker(t *testing.T) {
	t.Run("checks membership across teams by login", func(t *testing.T) {
		var requests atomic.Int32
		var healthy atomic.Bool
		healthy.Store(true)
		server := newTeamAPI(t, &requests, &healthy)
		checker := plugin.NewGrafanaTeamChecker(server.URL, "secret", []string{"Writers", "Ops"}, time.Minute)

		for login, want := range map[string]bool{"til": true, "ops": true, "til@example.org": false, "guest": false, "": false} {
			got, err := checker.IsMember(context.Background(), login)
			require.NoError(t, err)
			require.Equal(t, want, got, login)
		}
		require.Equal(t, int32(4), requests.Load(), "membership is cached")
	})

	t.Run("fails closed and does not cache errors", func(t *testing.T) {
		var requests atomic.Int32
		var healthy atomic.Bool
		server := newTeamAPI(t, &requests, &healthy)
		checker := plugin.NewGrafanaTeamChecker(server.URL, "secret", []string{"Writers"}, time.Minute)

		member, err := checker.IsMember(context.Background(), "til")
		require.Error(t, err)
		require.False(t, member)

		healthy.Store(true)
		member, err = checker.IsMember(context.Background(), "til")
		require.NoError(t, err)
		require.True(t, member)
	})

	t.Run("unknown team and wrong token are errors", func(t *testing.T) {
		var requests atomic.Int32
		var healthy atomic.Bool
		healthy.Store(true)
		server := newTeamAPI(t, &requests, &healthy)

		err := plugin.NewGrafanaTeamChecker(server.URL, "secret", []string{"Nobody"}, time.Minute).Verify(context.Background())
		require.ErrorContains(t, err, `team "Nobody" not found`)

		err = plugin.NewGrafanaTeamChecker(server.URL, "wrong", []string{"Writers"}, time.Minute).Verify(context.Background())
		require.ErrorContains(t, err, "401")
	})
}
