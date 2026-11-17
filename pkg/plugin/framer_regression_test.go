package plugin

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Regression tests for pkg/plugin/framer.go. They were written independently of PR #4 to reproduce the
// bugs it claims to fix (and that the E2E suite reproduces), so they must fail on the code before the fix.

func toFrame(t *testing.T, f *Framer, body string) (frame *data.Frame) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ToFrame(%s) panicked: %v", body, r)
		}
	}()
	frame, err := f.ToFrame(&TimestampedMessage{Timestamp: time.Now(), Value: []byte(body)})
	if err != nil {
		t.Fatalf("ToFrame(%s) returned an error: %v", body, err)
	}
	return frame
}

func fieldValue(t *testing.T, frame *data.Frame, name string) (interface{}, bool) {
	t.Helper()
	field, _ := frame.FieldByName(name)
	if field == nil {
		t.Fatalf("frame has no field %q (fields: %v)", name, fieldNames(frame))
	}
	if field.Len() != 1 {
		t.Fatalf("field %q has %d rows, want exactly 1", name, field.Len())
	}
	return field.ConcreteAt(0)
}

func fieldNames(frame *data.Frame) []string {
	names := make([]string, 0, len(frame.Fields))
	for _, f := range frame.Fields {
		names = append(names, f.Name)
	}
	return names
}

func TestFramerProducesOneRowPerMessage(t *testing.T) {
	f := NewFramer()
	frame := toFrame(t, f, `{"name":"a","value":1.5,"ok":true}`)

	if got := fmt.Sprint(fieldNames(frame)); got != "[RmqMsgConsumedTimestamp name value ok]" {
		t.Errorf("unexpected fields %s", got)
	}
	if v, _ := fieldValue(t, frame, "value"); v != 1.5 {
		t.Errorf("value = %v, want 1.5", v)
	}
	if v, _ := fieldValue(t, frame, "name"); v != "a" {
		t.Errorf("name = %v, want a", v)
	}
}

func TestFramerPadsFieldsMissingFromALaterMessage(t *testing.T) {
	f := NewFramer()
	toFrame(t, f, `{"a":1,"b":2}`)
	frame := toFrame(t, f, `{"a":3}`)

	if v, _ := fieldValue(t, frame, "a"); v != 3.0 {
		t.Errorf("a = %v, want 3", v)
	}
	if _, ok := fieldValue(t, frame, "b"); ok {
		t.Errorf("b should be null when it is missing from the message")
	}
}

func TestFramerNullAfterValueDoesNotPanic(t *testing.T) {
	f := NewFramer()
	toFrame(t, f, `{"reading":1}`)
	frame := toFrame(t, f, `{"reading":null,"name":"after-null"}`)

	if _, ok := fieldValue(t, frame, "reading"); ok {
		t.Errorf("reading should be null")
	}
	if v, _ := fieldValue(t, frame, "name"); v != "after-null" {
		t.Errorf("name = %v, want after-null", v)
	}
}

func TestFramerNullOnNestedObjectFieldDoesNotPanic(t *testing.T) {
	f := NewFramer()
	toFrame(t, f, `{"nested":{"k":1}}`)
	toFrame(t, f, `{"nested":null}`)
	frame := toFrame(t, f, `{"nested":{"k":2}}`)

	v, _ := fieldValue(t, frame, "nested")
	if raw, ok := v.(json.RawMessage); !ok || string(raw) != `{"k":2}` {
		t.Errorf("nested = %v (%T), want {\"k\":2}", v, v)
	}
}

func TestFramerKeepsValueWhenFieldTypeChanges(t *testing.T) {
	f := NewFramer()
	toFrame(t, f, `{"reading":1}`)
	frame := toFrame(t, f, `{"reading":"now a string"}`)

	if v, _ := fieldValue(t, frame, "reading"); v != "now a string" {
		t.Errorf("reading = %v, want the new string value", v)
	}
}

func TestFramerSurvivesNonJSONMessage(t *testing.T) {
	f := NewFramer()
	toFrame(t, f, `{"a":1}`)
	toFrame(t, f, `this is not json`)
	frame := toFrame(t, f, `{"a":2}`)

	if v, _ := fieldValue(t, frame, "a"); v != 2.0 {
		t.Errorf("a = %v, want 2", v)
	}
}
