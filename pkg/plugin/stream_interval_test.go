package plugin

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestStreamInterval(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want time.Duration
	}{
		{0, time.Second},
		{-time.Second, time.Second},
		{100 * time.Millisecond, 100 * time.Millisecond},
		{time.Second, time.Second},
		{20 * time.Second, time.Second},
		{10 * time.Minute, time.Second},
	}
	for _, tt := range tests {
		if got := streamInterval(tt.in); got != tt.want {
			t.Errorf("streamInterval(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestQueryChannelStableForLargeIntervals(t *testing.T) {
	ds := &MQTTDatasource{channelPrefix: "ds/test-uid"}
	queryJSON, _ := json.Marshal(map[string]interface{}{
		"topic":        "sensor/temperature",
		"streamingKey": "ds-uid/hash/org",
	})

	channel := func(interval time.Duration) string {
		resp := ds.query(backend.DataQuery{JSON: queryJSON, Interval: interval, RefID: "A"})
		if resp.Error != nil {
			t.Fatalf("query failed: %v", resp.Error)
		}
		return resp.Frames[0].Meta.Channel
	}

	base := channel(time.Second)
	if !strings.HasPrefix(base, "ds/test-uid/1s/") {
		t.Fatalf("unexpected channel %q", base)
	}
	for _, interval := range []time.Duration{2 * time.Second, time.Minute, 30 * time.Minute} {
		if got := channel(interval); got != base {
			t.Errorf("channel for interval %v = %q, want %q", interval, got, base)
		}
	}
}
