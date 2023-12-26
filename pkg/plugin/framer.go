package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
)

type Framer struct {
	Path     []string
	Iterator *jsoniter.Iterator
	Fields   []*data.Field
	FieldMap map[string]int
}

func NewFramer() *Framer {
	df := &Framer{
		FieldMap: make(map[string]int),
	}
	timeField := data.NewFieldFromFieldType(data.FieldTypeTime, 0)
	timeField.Name = TIMESTAMP_NAME
	df.Fields = append(df.Fields, timeField)
	df.FieldMap[TIMESTAMP_NAME] = 0
	return df
}

func (df *Framer) Next() error {
	switch df.Iterator.WhatIsNext() {
	case jsoniter.StringValue:
		v := df.Iterator.ReadString()
		df.AddValue(data.FieldTypeNullableString, &v)
	case jsoniter.NumberValue:
		v := df.Iterator.ReadFloat64()
		df.AddValue(data.FieldTypeNullableFloat64, &v)
	case jsoniter.BoolValue:
		v := df.Iterator.ReadBool()
		df.AddValue(data.FieldTypeNullableBool, &v)
	case jsoniter.NilValue:
		df.AddNil()
		df.Iterator.ReadNil()
	case jsoniter.ArrayValue:
		df.AddValue(data.FieldTypeJSON, json.RawMessage(df.Iterator.SkipAndReturnBytes()))
	case jsoniter.ObjectValue:
		size := len(df.Path)
		if size > 0 {
			df.AddValue(data.FieldTypeJSON, json.RawMessage(df.Iterator.SkipAndReturnBytes()))
			break
		}
		for fname := df.Iterator.ReadObject(); fname != ""; fname = df.Iterator.ReadObject() {
			if size == 0 {
				df.Path = append(df.Path, fname)
				if err := df.Next(); err != nil {
					return err
				}
			}
		}
	case jsoniter.InvalidValue:
		return fmt.Errorf("invalid value")
	}
	df.Path = []string{}
	return nil
}

func (df *Framer) Key() string {
	if len(df.Path) == 0 {
		return "Value"
	}
	return strings.Join(df.Path, "")
}

func (df *Framer) AddNil() {
	if idx, ok := df.FieldMap[df.Key()]; ok {
		df.Fields[idx].Set(0, nil)
		return
	}
	log.DefaultLogger.Info("nil value for unknown field", "key", df.Key())
}

func (df *Framer) AddValue(fieldType data.FieldType, v interface{}) {
	if idx, ok := df.FieldMap[df.Key()]; ok {
		if df.Fields[idx].Type() != fieldType {
			log.DefaultLogger.Info("field type mismatch", "key", df.Key(), "existing", df.Fields[idx], "new", fieldType)
			return
		}
		df.Fields[idx].Append(v)
		return
	}
	field := data.NewFieldFromFieldType(fieldType, df.Fields[0].Len())
	field.Name = df.Key()
	field.Append(v)
	df.Fields = append(df.Fields, field)
	df.FieldMap[df.Key()] = len(df.Fields) - 1
}

func (df *Framer) ToFrame(message *TimestampedMessage) (*data.Frame, error) {
	// clear the data in the fields
	for _, field := range df.Fields {
		for i := 0; i < field.Len(); i++ {
			field.Delete(i)
		}
	}

	df.Iterator = jsoniter.ParseBytes(jsoniter.ConfigDefault, message.Value)
	err := df.Next()
	if err != nil {
		log.DefaultLogger.Error("error parsing message", "error", err)
	}
	df.Fields[0].Append(message.Timestamp)
	df.ExtendFields(df.Fields[0].Len() - 1)

	return data.NewFrame("rabbitmq", df.Fields...), nil
}

func (df *Framer) ExtendFields(idx int) {
	for _, f := range df.Fields {
		if idx+1 > f.Len() {
			f.Extend(idx + 1 - f.Len())
		}
	}
}
