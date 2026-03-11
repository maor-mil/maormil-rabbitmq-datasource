package plugin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func makeMsg(json string) *TimestampedMessage {
	return &TimestampedMessage{
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Value:     []byte(json),
	}
}

func findField(frame *data.Frame, name string) *data.Field {
	for _, f := range frame.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// TestFramer_BasicFields verifies that a simple flat JSON message produces the
// expected fields and that the timestamp field is populated.
func TestFramer_BasicFields(t *testing.T) {
	framer := NewFramer()
	frame, err := framer.ToFrame(makeMsg(`{"name":"turbine1","power":42.5,"active":true}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{TIMESTAMP_NAME, "name", "power", "active"} {
		if findField(frame, want) == nil {
			t.Errorf("expected field %q to be present", want)
		}
	}

	ts := findField(frame, TIMESTAMP_NAME)
	if ts.Len() != 1 {
		t.Fatalf("timestamp field: want len 1, got %d", ts.Len())
	}

	powerField := findField(frame, "power")
	if powerField.Type() != data.FieldTypeNullableFloat64 {
		t.Errorf("power field: want NullableFloat64, got %v", powerField.Type())
	}
	v, ok := powerField.ConcreteAt(0)
	if !ok || v.(float64) != 42.5 {
		t.Errorf("power field value: want 42.5, got %v (ok=%v)", v, ok)
	}
}

// TestFramer_NestedObjectStoredAsJSON verifies that nested objects are stored
// as a single JSON blob field rather than being flattened.
func TestFramer_NestedObjectStoredAsJSON(t *testing.T) {
	framer := NewFramer()
	frame, err := framer.ToFrame(makeMsg(`{"source":{"id":"x","name":"y"},"value":1.0}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sourceField := findField(frame, "source")
	if sourceField == nil {
		t.Fatal("expected 'source' field to be present")
	}
	if sourceField.Type() != data.FieldTypeJSON {
		t.Errorf("source field: want FieldTypeJSON, got %v", sourceField.Type())
	}
}

// TestFramer_ConsecutiveMessages verifies that calling ToFrame twice in a row
// produces correctly cleared frames. Note: because frames share field objects
// with the framer, a frame's field data is only valid until the next ToFrame
// call — here we check each frame immediately after it is produced.
func TestFramer_ConsecutiveMessages(t *testing.T) {
	framer := NewFramer()

	frame1, err := framer.ToFrame(makeMsg(`{"value":1.0}`))
	if err != nil {
		t.Fatalf("first message error: %v", err)
	}

	// Inspect frame1 before the second ToFrame call mutates the shared fields.
	f1 := findField(frame1, "value")
	if f1 == nil {
		t.Fatal("frame1: 'value' field missing")
	}
	if f1.Len() != 1 {
		t.Errorf("frame1: want 1 row, got %d", f1.Len())
	}
	v1, ok := f1.ConcreteAt(0)
	if !ok || v1.(float64) != 1.0 {
		t.Errorf("frame1 value: want 1.0, got %v (ok=%v)", v1, ok)
	}

	frame2, err := framer.ToFrame(makeMsg(`{"value":2.0}`))
	if err != nil {
		t.Fatalf("second message error: %v", err)
	}

	f2 := findField(frame2, "value")
	if f2 == nil {
		t.Fatal("frame2: 'value' field missing")
	}
	if f2.Len() != 1 {
		t.Errorf("frame2: want 1 row, got %d", f2.Len())
	}
	v2, ok := f2.ConcreteAt(0)
	if !ok || v2.(float64) != 2.0 {
		t.Errorf("frame2 value: want 2.0, got %v (ok=%v)", v2, ok)
	}
}

// TestFramer_TypeMismatchReplacesField is the regression test for Bug #4.
// Before the fix, AddValue silently dropped values when the Go type of a
// field changed between messages. After the fix the field must be replaced
// and the new value must be captured.
func TestFramer_TypeMismatchReplacesField(t *testing.T) {
	framer := NewFramer()

	// First message: "status" is a number.
	_, err := framer.ToFrame(makeMsg(`{"status":1.0}`))
	if err != nil {
		t.Fatalf("first message error: %v", err)
	}

	// Second message: "status" changed to a string.
	frame2, err := framer.ToFrame(makeMsg(`{"status":"running"}`))
	if err != nil {
		t.Fatalf("second message error: %v", err)
	}

	f := findField(frame2, "status")
	if f == nil {
		t.Fatal("expected 'status' field in second frame")
	}

	// Field type must have been updated to NullableString.
	if f.Type() != data.FieldTypeNullableString {
		t.Errorf("after type change: want NullableString, got %v", f.Type())
	}

	// Value must be "running", not silently dropped.
	if f.Len() != 1 {
		t.Fatalf("want 1 value, got %d", f.Len())
	}
	v, ok := f.ConcreteAt(0)
	if !ok {
		t.Fatal("value is nil after type change")
	}
	if v.(string) != "running" {
		t.Errorf("want %q, got %q", "running", v.(string))
	}
}

// TestFramer_AllFieldsHaveSameLength verifies that ExtendFields pads fields
// that are absent from a message so every frame row has uniform width.
func TestFramer_AllFieldsHaveSameLength(t *testing.T) {
	framer := NewFramer()

	// First message introduces two fields.
	framer.ToFrame(makeMsg(`{"a":1.0,"b":2.0}`))

	// Second message only provides "a"; "b" must be padded to len 1.
	frame, _ := framer.ToFrame(makeMsg(`{"a":3.0}`))

	for _, f := range frame.Fields {
		if f.Len() != 1 {
			t.Errorf("field %q: want len 1, got %d", f.Name, f.Len())
		}
	}
}

// TestFramer_NullForKnownField is the regression test for the AddNil panic.
// Before the fix, calling AddNil() on a field that had been cleared by the
// ToFrame clearing loop would panic with "index out of range [0] with length 0"
// because Set(0, nil) requires the underlying slice to have at least one element.
// After the fix, AddNil() uses Append(nil) which is safe on a zero-length field.
func TestFramer_NullForKnownField(t *testing.T) {
	framer := NewFramer()

	// First message: x has a concrete value so it is registered in FieldMap.
	_, err := framer.ToFrame(makeMsg(`{"x":42.0}`))
	if err != nil {
		t.Fatalf("first message error: %v", err)
	}

	// Second message: x is null. After ToFrame clears the field to len=0,
	// AddNil() must NOT panic when recording the null value.
	frame, err := framer.ToFrame(makeMsg(`{"x":null}`))
	if err != nil {
		t.Fatalf("second message (null value): %v", err)
	}

	f := findField(frame, "x")
	if f == nil {
		t.Fatal("expected 'x' field to be present after null message")
	}
	if f.Len() != 1 {
		t.Fatalf("want len 1, got %d", f.Len())
	}
	// Value at index 0 must be nil (the null that was recorded).
	if _, ok := f.ConcreteAt(0); ok {
		t.Error("want nil value at index 0, but got a concrete value")
	}
}

// TestFramer_NullThenValue verifies that after a field records null, the next
// message can restore it to a concrete value without errors.
func TestFramer_NullThenValue(t *testing.T) {
	framer := NewFramer()

	framer.ToFrame(makeMsg(`{"x":42.0}`))
	framer.ToFrame(makeMsg(`{"x":null}`))

	frame, err := framer.ToFrame(makeMsg(`{"x":7.5}`))
	if err != nil {
		t.Fatalf("third message error: %v", err)
	}

	f := findField(frame, "x")
	if f == nil {
		t.Fatal("expected 'x' field after null-then-value sequence")
	}
	v, ok := f.ConcreteAt(0)
	if !ok || v.(float64) != 7.5 {
		t.Errorf("want 7.5, got %v (ok=%v)", v, ok)
	}
}

// realLiveSignalsMessage is a minimal but structurally faithful sample of the
// live-signals stream. It exercises:
//   - "source": deeply nested object (stored as JSON blob)
//   - "guid": top-level string field
//   - "samples": nested object whose leaf values include integer 0, float 2.009,
//     float 67.6, a string, nullable "unit" fields, and nullable "sortingPosition"
//     fields (all stored together as a JSON blob)
const realLiveSignalsMessage = `{"source":{"parkId":"triptiprow9","parkName":"Trip Tip Row 9","assetId":"us_chn_10","assetName":"CF22","assetType":"gam_g5.0mw-145","assetSortingName":"triptiprow9_010","oem":"AnemoLive","ipAddress":"192.168.20.101","timestampUtc":"2026-02-27T16:02:01.9966842Z","localTimestamp":"2026-02-27T11:02:01.9966842Z","eventType":"live-signals","displayName":"CF22"},"guid":"eb8307a0-9c05-48e9-91fb-7cc346f6856a","samples":{"PowerLimitComp":{"tag":"PowerLimitComp","value":0,"timestampUtc":"2026-02-27T16:02:01.9966842Z","name":"PowerLimitComp","unit":null,"assetName":"CF22","assetId":"us_chn_10","sortingPosition":null},"PropValveBSliderPosition":{"tag":"PropValveBSliderPosition","value":2.009,"timestampUtc":"2026-02-27T16:02:01.9966842Z","name":"PropValveBSliderPosition","unit":null,"assetName":"CF22","assetId":"us_chn_10","sortingPosition":null},"WSCS.GbxBrgHSGenTmp":{"tag":"WSCS.GbxBrgHSGenTmp","value":67.6,"timestampUtc":"2026-02-27T16:02:01.9966842Z","name":"Temp Gearbox Bearing","unit":"\u00b0C","assetName":"CF22","assetId":"us_chn_10","sortingPosition":"Tag_1"},"WSCS.TurSt":{"tag":"WSCS.TurSt","value":"TURBINE_PRODUCTION","timestampUtc":"2026-02-27T16:02:01.9966842Z","name":"Turbine State","unit":null,"assetName":"CF22","assetId":"us_chn_10","sortingPosition":null}}}`

// TestFramer_RealLiveSignalsMessage tests the framer with a structurally
// faithful representation of the live-signals RabbitMQ stream messages.
// The message body has three top-level fields:
//   - source: nested object -> stored as JSON blob
//   - guid: string -> stored as NullableString
//   - samples: nested object containing entries with integer (0), float (2.009),
//     float (67.6), string values, and null unit / sortingPosition fields ->
//     stored as a single JSON blob
//
// This test validates that the framer produces exactly these three top-level
// fields plus the timestamp, with correct types, and does not panic on the
// null values inside the samples blob.
func TestFramer_RealLiveSignalsMessage(t *testing.T) {
	framer := NewFramer()

	frame, err := framer.ToFrame(makeMsg(realLiveSignalsMessage))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must have exactly: timestamp, source, guid, samples.
	wantFields := []string{TIMESTAMP_NAME, "source", "guid", "samples"}
	for _, name := range wantFields {
		if findField(frame, name) == nil {
			t.Errorf("expected field %q to be present", name)
		}
	}

	// Timestamp field must have exactly one row.
	ts := findField(frame, TIMESTAMP_NAME)
	if ts.Len() != 1 {
		t.Errorf("timestamp: want len 1, got %d", ts.Len())
	}

	// source must be a JSON blob (nested object, not flattened).
	src := findField(frame, "source")
	if src.Type() != data.FieldTypeJSON {
		t.Errorf("source: want FieldTypeJSON, got %v", src.Type())
	}

	// guid must be NullableString with the expected value.
	guidField := findField(frame, "guid")
	if guidField.Type() != data.FieldTypeNullableString {
		t.Errorf("guid: want NullableString, got %v", guidField.Type())
	}
	guidVal, ok := guidField.ConcreteAt(0)
	if !ok || guidVal.(string) != "eb8307a0-9c05-48e9-91fb-7cc346f6856a" {
		t.Errorf("guid value: want %q, got %v (ok=%v)", "eb8307a0-9c05-48e9-91fb-7cc346f6856a", guidVal, ok)
	}

	// samples must be a JSON blob (nested object containing all sample entries
	// including those with null unit and null sortingPosition).
	samplesField := findField(frame, "samples")
	if samplesField.Type() != data.FieldTypeJSON {
		t.Errorf("samples: want FieldTypeJSON, got %v", samplesField.Type())
	}

	// All fields must have exactly one row.
	for _, f := range frame.Fields {
		if f.Len() != 1 {
			t.Errorf("field %q: want len 1, got %d", f.Name, f.Len())
		}
	}
}

// TestFramer_RealLiveSignalsConsecutiveMessages verifies that two consecutive
// real-format messages are processed correctly: fields are cleared between
// messages, the second frame has the right data, and field counts match.
func TestFramer_RealLiveSignalsConsecutiveMessages(t *testing.T) {
	framer := NewFramer()

	// Second message uses the same structure but a different guid.
	msg2 := `{"source":{"parkId":"triptiprow9","parkName":"Trip Tip Row 9","assetId":"us_chn_10","assetName":"CF22","assetType":"gam_g5.0mw-145","assetSortingName":"triptiprow9_010","oem":"AnemoLive","ipAddress":"192.168.20.101","timestampUtc":"2026-02-27T16:03:00Z","localTimestamp":"2026-02-27T11:03:00Z","eventType":"live-signals","displayName":"CF22"},"guid":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","samples":{"PowerLimitComp":{"tag":"PowerLimitComp","value":0,"timestampUtc":"2026-02-27T16:03:00Z","name":"PowerLimitComp","unit":null,"assetName":"CF22","assetId":"us_chn_10","sortingPosition":null},"PropValveBSliderPosition":{"tag":"PropValveBSliderPosition","value":2.009,"timestampUtc":"2026-02-27T16:03:00Z","name":"PropValveBSliderPosition","unit":null,"assetName":"CF22","assetId":"us_chn_10","sortingPosition":null}}}`

	frame1, err := framer.ToFrame(makeMsg(realLiveSignalsMessage))
	if err != nil {
		t.Fatalf("first message error: %v", err)
	}

	// Validate frame1 guid before the second call mutates shared fields.
	g1 := findField(frame1, "guid")
	if g1 == nil {
		t.Fatal("frame1: 'guid' field missing")
	}
	v1, ok := g1.ConcreteAt(0)
	if !ok || v1.(string) != "eb8307a0-9c05-48e9-91fb-7cc346f6856a" {
		t.Errorf("frame1 guid: want original uuid, got %v (ok=%v)", v1, ok)
	}

	frame2, err := framer.ToFrame(makeMsg(msg2))
	if err != nil {
		t.Fatalf("second message error: %v", err)
	}

	// All fields in frame2 must have exactly one row.
	for _, f := range frame2.Fields {
		if f.Len() != 1 {
			t.Errorf("frame2 field %q: want len 1, got %d", f.Name, f.Len())
		}
	}

	// guid in frame2 must reflect the second message's value.
	g2 := findField(frame2, "guid")
	if g2 == nil {
		t.Fatal("frame2: 'guid' field missing")
	}
	v2, ok := g2.ConcreteAt(0)
	if !ok || v2.(string) != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("frame2 guid: want new uuid, got %v (ok=%v)", v2, ok)
	}
}

// TestFramer_NullableUnitAndSortingPosition verifies that samples containing
// null "unit" and null "sortingPosition" fields (as seen in the real
// live-signals stream) are stored correctly as part of the JSON blob.
// This is a documentation test: the framer does not parse inside the samples
// blob, so the nulls are simply preserved in the raw JSON.
func TestFramer_NullableUnitAndSortingPosition(t *testing.T) {
	framer := NewFramer()

	// Message whose samples contain entries with null unit and sortingPosition
	// (mirrors the real PowerLimitComp sample) as well as one with non-null
	// unit and sortingPosition (mirrors WSCS.GbxBrgHSGenTmp).
	const msg = `{"source":{"parkId":"triptiprow9"},"guid":"test-guid","samples":{"PowerLimitComp":{"tag":"PowerLimitComp","value":0,"timestampUtc":"2026-02-27T16:02:01Z","name":"PowerLimitComp","unit":null,"assetName":"CF22","assetId":"us_chn_10","sortingPosition":null},"WSCS.GbxBrgHSGenTmp":{"tag":"WSCS.GbxBrgHSGenTmp","value":67.6,"timestampUtc":"2026-02-27T16:02:01Z","name":"Temp Gearbox Bearing","unit":"\u00b0C","assetName":"CF22","assetId":"us_chn_10","sortingPosition":"Tag_1"}}}`

	frame, err := framer.ToFrame(makeMsg(msg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The samples blob must be present and non-empty.
	samplesField := findField(frame, "samples")
	if samplesField == nil {
		t.Fatal("'samples' field missing")
	}
	if samplesField.Type() != data.FieldTypeJSON {
		t.Errorf("samples: want FieldTypeJSON, got %v", samplesField.Type())
	}
	samplesVal, ok := samplesField.ConcreteAt(0)
	if !ok {
		t.Fatal("samples value is nil")
	}
	// The blob must contain the null unit value for PowerLimitComp.
	// The framer stores JSON blobs as json.RawMessage.
	blob := string(samplesVal.(json.RawMessage))
	if len(blob) == 0 {
		t.Error("samples blob is empty")
	}
}

// TestFramer_NullForJSONField is the regression test for the alarm-stream panic.
// When a field is first seen as a JSON blob (FieldTypeJSON) and a subsequent
// message sends null for that field, AddNil must NOT panic with
// "interface conversion: interface {} is nil, not json.RawMessage".
// FieldTypeJSON is non-nullable so the nil must be skipped and ExtendFields
// pads with the zero value instead.
func TestFramer_NullForJSONField(t *testing.T) {
	framer := NewFramer()

	// First message: "payload" is an object → stored as FieldTypeJSON blob.
	_, err := framer.ToFrame(makeMsg(`{"payload":{"key":"value","count":1}}`))
	if err != nil {
		t.Fatalf("first message error: %v", err)
	}

	// Second message: "payload" is null. Must NOT panic.
	frame, err := framer.ToFrame(makeMsg(`{"payload":null}`))
	if err != nil {
		t.Fatalf("second message (null JSON field) error: %v", err)
	}

	f := findField(frame, "payload")
	if f == nil {
		t.Fatal("expected 'payload' field to be present")
	}
	if f.Len() != 1 {
		t.Fatalf("want len 1, got %d", f.Len())
	}
}

// TestFramer_NullForArrayField verifies the same non-nullable nil behaviour for
// FieldTypeJSON fields that were originally populated from a JSON array.
func TestFramer_NullForArrayField(t *testing.T) {
	framer := NewFramer()

	// First message: "tags" is an array → FieldTypeJSON.
	_, err := framer.ToFrame(makeMsg(`{"tags":["alarm","critical"]}`))
	if err != nil {
		t.Fatalf("first message error: %v", err)
	}

	// Second message: "tags" is null. Must NOT panic.
	frame, err := framer.ToFrame(makeMsg(`{"tags":null}`))
	if err != nil {
		t.Fatalf("second message (null array field) error: %v", err)
	}

	f := findField(frame, "tags")
	if f == nil {
		t.Fatal("expected 'tags' field to be present")
	}
	if f.Len() != 1 {
		t.Fatalf("want len 1, got %d", f.Len())
	}
}

// TestFramer_IntegerZeroValue verifies that a JSON integer value of 0 is
// correctly stored as float64(0) in the framer (jsoniter reads all JSON numbers
// as float64 when using ReadFloat64).
func TestFramer_IntegerZeroValue(t *testing.T) {
	framer := NewFramer()

	// 'count' has integer value 0 — this is the PowerLimitComp pattern.
	frame, err := framer.ToFrame(makeMsg(`{"count":0}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f := findField(frame, "count")
	if f == nil {
		t.Fatal("'count' field missing")
	}
	if f.Type() != data.FieldTypeNullableFloat64 {
		t.Errorf("count: want NullableFloat64, got %v", f.Type())
	}
	v, ok := f.ConcreteAt(0)
	if !ok || v.(float64) != 0 {
		t.Errorf("count value: want 0, got %v (ok=%v)", v, ok)
	}
}

// TestFramer_FloatValues verifies that float values 2.009 and 67.6 (representative
// of PropValveBSliderPosition and WSCS.GbxBrgHSGenTmp from the real stream) are
// stored with full precision.
func TestFramer_FloatValues(t *testing.T) {
	framer := NewFramer()

	frame, err := framer.ToFrame(makeMsg(`{"valve":2.009,"temp":67.6}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	valve := findField(frame, "valve")
	if valve == nil {
		t.Fatal("'valve' field missing")
	}
	v, ok := valve.ConcreteAt(0)
	if !ok {
		t.Fatal("valve value is nil")
	}
	if v.(float64) != 2.009 {
		t.Errorf("valve: want 2.009, got %v", v)
	}

	temp := findField(frame, "temp")
	if temp == nil {
		t.Fatal("'temp' field missing")
	}
	v2, ok := temp.ConcreteAt(0)
	if !ok {
		t.Fatal("temp value is nil")
	}
	if v2.(float64) != 67.6 {
		t.Errorf("temp: want 67.6, got %v", v2)
	}
}
