package plugin_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/mqtt-datasource/pkg/mqtt"
	"github.com/grafana/mqtt-datasource/pkg/plugin"
	"github.com/stretchr/testify/require"
)

func TestSimpleValueMessage(t *testing.T) {
	frame := plugin.ToFrame("test/data", []mqtt.Message{
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
	values := []float64{
		-0.5182926829268293,
		-0.3582317073170732,
		0.1753048780487805,
		0.20599365234375,
		-0.050048828125,
		1.03582763671875,
	}
	msg := fmt.Sprintf(`{"ax": %v, "ay": %v, "az": %v, "gx": %v, "gy": %v, "gz": %v}`,
		values[0], values[1], values[2], values[3], values[4], values[5])
	frame := plugin.ToFrame("test/data", []mqtt.Message{
		{
			Timestamp: timestamp,
			Value:     msg,
		},
	})
	numFields := len(values) + 1
	require.NotNil(t, frame)
	require.Equal(t, numFields, len(frame.Fields))
	v, ok := frame.Fields[0].ConcreteAt(0)
	require.Equal(t, true, ok)
	require.Equal(t, v, timestamp)
	for idx, val := range values {
		v, err := frame.Fields[idx+1].FloatAt(0)
		require.NoError(t, err)
		require.Equal(t, val, v)
	}
}
