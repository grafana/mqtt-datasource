package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
)

type framer struct {
	path     []string
	iterator *jsoniter.Iterator
	fields   []*data.Field
	fieldMap map[string]int
}

func (df *framer) next(logger log.Logger) error {
	switch df.iterator.WhatIsNext() {
	case jsoniter.StringValue:
		v := df.iterator.ReadString()
		df.addValue(data.FieldTypeNullableString, &v)
	case jsoniter.NumberValue:
		v := df.iterator.ReadFloat64()
		df.addValue(data.FieldTypeNullableFloat64, &v)
	case jsoniter.BoolValue:
		v := df.iterator.ReadBool()
		df.addValue(data.FieldTypeNullableBool, &v)
	case jsoniter.NilValue:
		df.addNil(logger)
		df.iterator.ReadNil()
	case jsoniter.ArrayValue:
		df.addValue(data.FieldTypeJSON, json.RawMessage(df.iterator.SkipAndReturnBytes()))
	case jsoniter.ObjectValue:
		size := len(df.path)
		if size > 0 {
			df.addValue(data.FieldTypeJSON, json.RawMessage(df.iterator.SkipAndReturnBytes()))
			break
		}
		for fname := df.iterator.ReadObject(); fname != ""; fname = df.iterator.ReadObject() {
			if size == 0 {
				df.path = append(df.path, fname)
				if err := df.next(logger); err != nil {
					return err
				}
			}
		}
	case jsoniter.InvalidValue:
		return fmt.Errorf("invalid value")
	}
	df.path = []string{}
	return nil
}

func (df *framer) key() string {
	if len(df.path) == 0 {
		return "Value"
	}
	return strings.Join(df.path, "")
}

func (df *framer) addNil(logger log.Logger) {
	if idx, ok := df.fieldMap[df.key()]; ok {
		df.fields[idx].Set(0, nil)
		return
	}
	logger.Debug("nil value for unknown field", "key", df.key())
}

func (df *framer) addValue(fieldType data.FieldType, v interface{}) {
	if idx, ok := df.fieldMap[df.key()]; ok {
		if df.fields[idx].Type() != fieldType {
			log.DefaultLogger.Debug("field type mismatch", "key", df.key(), "existing", df.fields[idx], "new", fieldType)
			return
		}
		df.fields[idx].Append(v)
		return
	}
	field := data.NewFieldFromFieldType(fieldType, df.fields[0].Len())
	field.Name = df.key()
	field.Append(v)
	df.fields = append(df.fields, field)
	df.fieldMap[df.key()] = len(df.fields) - 1
}

func newFramer() *framer {
	df := &framer{
		fieldMap: make(map[string]int),
	}
	timeField := data.NewFieldFromFieldType(data.FieldTypeTime, 0)
	timeField.Name = "Time"
	df.fields = append(df.fields, timeField)
	df.fieldMap["Time"] = 0
	return df
}

func (df *framer) toFrame(messages []Message, logger log.Logger) (*data.Frame, error) {
	// clear the data in the fields
	for _, field := range df.fields {
		for i := field.Len() - 1; i >= 0; i-- {
			field.Delete(i)
		}
	}

	for _, message := range messages {
		df.iterator = jsoniter.ParseBytes(jsoniter.ConfigDefault, message.Value)
		err := df.next(logger)
		if err != nil {
			// If JSON parsing fails, treat the raw bytes as a string value
			logger.Debug("JSON parsing failed, treating as raw string", "error", err, "value", string(message.Value))
			rawValue := string(message.Value)
			df.addValue(data.FieldTypeNullableString, &rawValue)
		}
		df.fields[0].Append(message.Timestamp)
		df.extendFields(df.fields[0].Len() - 1)
	}

	return data.NewFrame("mqtt", df.fields...), nil
}

func (df *framer) extendFields(idx int) {
	for _, f := range df.fields {
		if idx+1 > f.Len() {
			f.Extend(idx + 1 - f.Len())
		}
	}
}
