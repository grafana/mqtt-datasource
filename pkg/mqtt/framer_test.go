package mqtt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental"
	"github.com/stretchr/testify/require"
)

var update = false

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

	t.Run("object with changing field type", func(t *testing.T) {
		runTest(t, "object-changing-type",
			map[string]interface{}{"a": 1, "b": 2},
			map[string]interface{}{"a": "test", "b": false},
			map[string]interface{}{"a": 3, "b": 4},
		)
	})

	t.Run("sparse fields", func(t *testing.T) {
		runTest(t, "sparse-values",
			map[string]interface{}{"a": 1, "b": 2},
			map[string]interface{}{"b": 3},
			map[string]interface{}{"a": 4},
			map[string]interface{}{"c": 5},
		)
	})

	t.Run("nested-object", func(t *testing.T) {
		runTest(t, "nested-object", map[string]interface{}{"a": 1, "b": map[string]any{"c": []any{1, 2, 3}}})
	})

	t.Run("null", func(t *testing.T) {
		runTest(t, "null", nil)
	})

	// Test raw string values (without JSON encoding) - this reproduces the user issue
	t.Run("raw string values", func(t *testing.T) {
		runRawTest(t, "raw-string", []byte("on"), []byte("off"), []byte("admin_off"))
	})

	// Test raw numeric values
	t.Run("raw numeric values", func(t *testing.T) {
		runRawTest(t, "raw-number", []byte("123"), []byte("456.789"))
	})

	// Test mixed raw values
	t.Run("mixed raw values", func(t *testing.T) {
		runRawTest(t, "raw-mixed", []byte("25"), []byte("on"), []byte("123.45"))
	})
}

func runTest(t *testing.T, name string, values ...any) {
	t.Helper()
	f := newFramer()
	timestamp := time.Unix(0, 0)
	messages := []Message{}
	for i, v := range values {
		messages = append(messages, Message{Timestamp: timestamp.Add(time.Duration(i) * time.Minute), Value: toJSON(v)})
	}
	frame, err := f.toFrame(messages, log.DefaultLogger)
	require.NoError(t, err)
	require.NotNil(t, frame)
	experimental.CheckGoldenJSONFrame(t, "testdata", name, frame, update)
}

func runRawTest(t *testing.T, name string, rawValues ...[]byte) {
	t.Helper()
	f := newFramer()
	timestamp := time.Unix(0, 0)
	messages := []Message{}
	for i, v := range rawValues {
		messages = append(messages, Message{Timestamp: timestamp.Add(time.Duration(i) * time.Minute), Value: v})
	}
	frame, err := f.toFrame(messages, log.DefaultLogger)
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
