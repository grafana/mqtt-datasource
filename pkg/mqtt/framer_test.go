package mqtt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental"
	"github.com/stretchr/testify/require"
)

var update = true

func Test_framer(t *testing.T) {

	t.Run("string", func(t *testing.T) {
		runTest(t, "string", "hello")
	})

	t.Run("int", func(t *testing.T) {
		runTest(t, "int", 123)
	})

	t.Run("float", func(t *testing.T) {
		runTest(t, "float", 123.456)
	})

	t.Run("bool", func(t *testing.T) {
		runTest(t, "bool", true)
	})

	t.Run("array", func(t *testing.T) {
		runTest(t, "array", []interface{}{1, 2, 3})
	})

	t.Run("object", func(t *testing.T) {
		runTest(t, "object", map[string]interface{}{"a": 1, "b": 2})
	})

	t.Run("nested-object", func(t *testing.T) {
		runTest(t, "nested-object", map[string]interface{}{"a": 1, "b": map[string]any{"c": []any{1, 2, 3}}})
	})

	t.Run("null", func(t *testing.T) {
		runTest(t, "null", nil)
	})
}

func runTest(t *testing.T, name string, v any) {
	t.Helper()
	f := newFramer()
	frame, err := f.toFrame([]Message{{Timestamp: time.Unix(100, 0), Value: toJSON(v)}})
	require.NoError(t, err)
	require.NotNil(t, frame)
	experimental.CheckGoldenJSONFrame(t, "testdata", name, frame, update)
}

func toJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
