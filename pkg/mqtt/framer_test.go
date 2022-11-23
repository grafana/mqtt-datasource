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
}

func runTest(t *testing.T, name string, values ...any) {
	t.Helper()
	f := newFramer()
	timestamp := time.Unix(0, 0)
	messages := []Message{}
	for i, v := range values {
		messages = append(messages, Message{Timestamp: timestamp.Add(time.Duration(i) * time.Minute), Value: toJSON(v)})
	}
	frame, err := f.toFrame(messages)
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
