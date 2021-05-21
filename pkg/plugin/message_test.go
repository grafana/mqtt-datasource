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
	frame := plugin.ToFrame("test/data", []mqtt.Message{
		{
			Timestamp: time.Unix(1, 0),
			Value:     `{"gx": -0.5182926829268293, "gy": -0.3582317073170732, "gz": 0.1753048780487805, "ax": 0.20599365234375, "ay": -0.050048828125, "az": 1.03582763671875}`,
		},
	})
	require.NotNil(t, frame)

	str, err := frame.StringTable(100, 5)
	require.NoError(t, err)
	fmt.Printf("FRAME: %s", str)
	require.Equal(t, 7, len(frame.Fields))
}
