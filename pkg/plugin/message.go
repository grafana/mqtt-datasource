package plugin

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

func ToFrame(fields map[string]*data.Field, topic string, messages []mqtt.Message) *data.Frame {
	count := len(messages)
	if count > 0 {
		first := messages[0].Value
		if strings.HasPrefix(first, "{") {
			return jsonMessagesToFrame(fields, topic, messages)
		}
	}

	// Fall through to expecting values
	timeField := data.NewFieldFromFieldType(data.FieldTypeTime, count)
	timeField.Name = "Time"
	valueField := data.NewFieldFromFieldType(data.FieldTypeFloat64, count)
	valueField.Name = "Value"

	for idx, m := range messages {
		if value, err := strconv.ParseFloat(m.Value, 64); err == nil {
			timeField.Set(idx, m.Timestamp)
			valueField.Set(idx, value)
		}
	}

	return data.NewFrame(topic, timeField, valueField)
}

// unnestMap recursively unwraps a given arbitrary map, thereby appending its concrete values under the resulting path.
func unnestMap(v interface{}, newmap map[string]interface{}, key string) map[string]interface{} {
	switch v := v.(type) {
	case map[string]interface{}: // Object
		for k, val := range v {
			unnestMap(val, newmap, key+"."+k)
		}
	case []interface{}: // Array
		for newkey := 0; newkey < len(v); newkey++ {
			newmap[key+"["+strconv.Itoa(newkey)+"]"] = v[newkey]
		}
	default: // Number (float64), String (string), Boolean (bool), Null (nil)
		newmap[key] = v
	}
	return newmap

}

// changeMapStructure transforms a nested arbitrary map to a simple key value layout.
func changeMapStructure(nestedMap map[string]interface{}) map[string]interface{} {
	newmap := make(map[string]interface{})
	for k, v := range nestedMap {
		newmap = unnestMap(v, newmap, k)
	}
	return newmap
}

func jsonMessagesToFrame(fields map[string]*data.Field, topic string, messages []mqtt.Message) *data.Frame {
	count := len(messages)
	if count == 0 {
		return nil
	}
	timeField := data.NewFieldFromFieldType(data.FieldTypeTime, count)
	timeField.Name = "Time"
	for row, m := range messages {
		timeField.SetConcrete(row, m.Timestamp)
		var body map[string]interface{}
		if err := json.Unmarshal([]byte(m.Value), &body); err != nil {
			return nil
		}
		for key, val := range changeMapStructure(body) {
			var t data.FieldType
			field, exists := fields[key]
			switch val.(type) {
			case float64:
				t = data.FieldTypeNullableFloat64
			case string:
				t = data.FieldTypeNullableString
			case bool:
				t = data.FieldTypeNullableBool
			default:
				delete(fields, key)
				continue
			}
			if !exists || field.Type() != t {
				field = data.NewFieldFromFieldType(t, count)
				field.Name = key
				fields[key] = field
			}
			field.SetConcrete(row, val)
		}
	}
	frame := data.NewFrame(topic, timeField)
	for _, f := range fields {
		frame.Fields = append(frame.Fields, f)
	}
	sort.Slice(frame.Fields, func(i, j int) bool {
		return frame.Fields[i].Name < frame.Fields[j].Name
	})
	return frame
}
