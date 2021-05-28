package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/mqtt-datasource/pkg/mqtt"
)

func ToFrame(topic string, messages []mqtt.Message) *data.Frame {
	count := len(messages)
	if count > 0 {
		first := messages[0].Value
		if strings.HasPrefix(first, "{") {
			return jsonMessagesToFrame(topic, messages)
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

func jsonMessagesToFrame(topic string, messages []mqtt.Message) *data.Frame {
	count := len(messages)
	if count == 0 {
		return nil
	}

	var body map[string]float64
	err := json.Unmarshal([]byte(messages[0].Value), &body)
	if err != nil {
		frame := data.NewFrame(topic)
		frame.AppendNotices(data.Notice{Severity: data.NoticeSeverityError,
			Text: fmt.Sprintf("error unmarshalling json message: %s", err.Error()),
		})
		return frame
	}

	timeField := data.NewFieldFromFieldType(data.FieldTypeTime, count)
	timeField.Name = "Time"
	timeField.SetConcrete(0, messages[0].Timestamp)

	// Create a field for each key and set the first value
	keys := make([]string, 0, len(body))
	fields := make(map[string]*data.Field, len(body))
	for key, val := range body {
		field := data.NewFieldFromFieldType(data.FieldTypeNullableFloat64, count)
		field.Name = key
		field.SetConcrete(0, val)
		fields[key] = field
		keys = append(keys, key)
	}
	sort.Strings(keys) // keys stable field order.

	// Add rows 1...n
	for row, m := range messages {
		if row == 0 {
			continue
		}

		err := json.Unmarshal([]byte(m.Value), &body)
		if err != nil {
			continue // bad row?
		}

		timeField.SetConcrete(row, m.Timestamp)
		for key, val := range body {
			field, ok := fields[key]
			if ok {
				field.SetConcrete(row, val)
			}
		}
	}

	frame := data.NewFrame(topic, timeField)
	for _, key := range keys {
		frame.Fields = append(frame.Fields, fields[key])
	}
	return frame
}
