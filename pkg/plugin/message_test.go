package plugin_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
	"github.com/grafana/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func TestSimpleValueMessage(t *testing.T) {
	frame := plugin.ToFrame(make(map[string]*data.Field), "test/data", []mqtt.Message{
		{
			Timestamp: time.Unix(1, 0),
			Value:     "1",
		},
	})
	v, err := frame.Fields[1].FloatAt(0)
	require.NoError(t, err)
	require.Equal(t, v, float64(1))
}

func TestJSONValuesMessage(t *testing.T) {
	timestamp := time.Unix(1, 0)
	values := []interface{}{
		"-0.5182926829268293",
		"Hi",
		0.1753048780487805,
		false,
		-0.050048828125,
		true,
	}
	cases := []struct {
		msg  string
		keys []string
	}{
		{
			msg: fmt.Sprintf(`{"ax": %#v, "ay": %#v, "az": %#v, "gx": %#v, "gy": %#v, "gz": %#v}`,
				values[0], values[1], values[2], values[3], values[4], values[5]),
			keys: []string{"ax", "ay", "az", "gx", "gy", "gz"}},

		{
			msg: fmt.Sprintf(`{"ax": %#v, "ay":{"az": { "gx": %#v}, "gy": %#v}, "gz":{"b":{"bx": %#v,"by": %#v, "bz": %#v}}}`,
				values[0], values[1], values[2], values[3], values[4], values[5]),
			keys: []string{"ax", "ay.az.gx", "ay.gy", "gz.b.bx", "gz.b.by", "gz.b.bz"}},
	}
	for _, c := range cases {
		frame := plugin.ToFrame(make(map[string]*data.Field), "test/data", []mqtt.Message{
			{
				Timestamp: timestamp,
				Value:     c.msg,
			},
		})
		numFields := len(values) + 1
		require.NotNil(t, frame)
		require.Equal(t, numFields, len(frame.Fields))
		v, ok := frame.Fields[0].ConcreteAt(0)
		require.Equal(t, true, ok)
		require.Equal(t, v, timestamp)
		for idx, val := range values {
			require.Equal(t, frame.Fields[idx+1].Name, c.keys[idx])
			v, _ := frame.Fields[idx+1].ConcreteAt(0)
			require.Equal(t, val, v)
		}
	}
}
