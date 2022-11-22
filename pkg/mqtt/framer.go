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

func (df *framer) next() error {
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
		df.addNil()
		df.iterator.ReadNil()
	case jsoniter.ArrayValue:
		df.addValue(data.FieldTypeJSON, json.RawMessage(df.iterator.SkipAndReturnBytes()))
	case jsoniter.ObjectValue:
		size := len(df.path)
		for fname := df.iterator.ReadObject(); fname != ""; fname = df.iterator.ReadObject() {
			if size == 0 {
				df.path = []string{fname}
				df.next()
				continue
			}
			df.addValue(data.FieldTypeJSON, json.RawMessage(df.iterator.SkipAndReturnBytes()))
			df.path = []string{}
		}

	case jsoniter.InvalidValue:
		return fmt.Errorf("invalid value")
	}
	return nil
}

func (df *framer) key() string {
	if len(df.path) == 0 {
		return "Value"
	}
	return strings.Join(df.path, "")
}

func (df *framer) addNil() {
	if idx, ok := df.fieldMap[df.key()]; ok {
		df.fields[idx].Set(0, nil)
	} else {
		log.DefaultLogger.Warn("Skip nil field", "key", df.key())
	}
}

func (df *framer) addValue(fieldType data.FieldType, v interface{}) {
	if idx, ok := df.fieldMap[df.key()]; ok {
		df.fields[idx].Append(v)
		return
	}

	field := data.NewFieldFromFieldType(fieldType, 1)
	field.Name = df.key()
	field.Set(0, v)
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

func (df *framer) toFrame(messages []Message) (*data.Frame, error) {
	// clear the data in the fields
	for _, field := range df.fields {
		for i := 0; i < field.Len(); i++ {
			field.Delete(i)
		}
	}

	for _, message := range messages {
		df.iterator = jsoniter.ParseBytes(jsoniter.ConfigDefault, message.Value)
		err := df.next()
		if err != nil {
			return nil, err
		}
		df.fields[0].Append(message.Timestamp)
	}

	return data.NewFrame("mqtt", df.fields...), nil
}
