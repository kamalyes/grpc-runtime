package runtime

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kamalyes/grpc-runtime/testpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestJSONBuiltinMarshal(t *testing.T) {
	var m JSONBuiltin
	msg := &testpb.SimpleMessage{Id: "foo"}

	buf, err := m.Marshal(msg)
	assert.NoError(t, err, "m.Marshal(%v)", msg)

	got := new(testpb.SimpleMessage)
	assert.NoError(t, json.Unmarshal(buf, got), "json.Unmarshal(%q, got)", buf)
	assert.Empty(t, cmp.Diff(got, msg, protocmp.Transform()))
}

func TestJSONBuiltinMarshalField(t *testing.T) {
	var m JSONBuiltin

	for _, fixt := range builtinFieldFixtures {
		var buf []byte
		var err error
		if len(fixt.indent) == 0 {
			buf, err = m.Marshal(fixt.data)
		} else {
			buf, err = m.MarshalIndent(fixt.data, "", fixt.indent)
		}
		assert.NoError(t, err, "m.Marshal(%v)", fixt.data)
		assert.Equal(t, fixt.json, string(buf), "data = %#v", fixt.data)
	}
}

func TestJSONBuiltinMarshalFieldKnownErrors(t *testing.T) {
	var m JSONBuiltin
	for _, fixt := range builtinKnownErrors {
		buf, err := m.Marshal(fixt.data)
		assert.NoError(t, err, "m.Marshal(%v)", fixt.data)
		assert.NotEqual(t, fixt.json, string(buf), "surprisingly got expected output; data = %#v", fixt.data)
	}
}

func TestJSONBuiltinUnmarshal(t *testing.T) {
	var m JSONBuiltin
	got := new(testpb.SimpleMessage)
	data := []byte(`{"id": "foo"}`)

	assert.NoError(t, m.Unmarshal(data, got), "m.Unmarshal(%q, got)", data)

	want := &testpb.SimpleMessage{Id: "foo"}
	assert.Empty(t, cmp.Diff(got, want, protocmp.Transform()))
}

func TestJSONBuiltinUnmarshalField(t *testing.T) {
	var m JSONBuiltin
	for _, fixt := range builtinFieldFixtures {
		dest := alloc(reflect.TypeOf(fixt.data))
		assert.NoError(t, m.Unmarshal([]byte(fixt.json), dest.Interface()), "m.Unmarshal(%q, dest)", fixt.json)
		assert.Empty(t, cmp.Diff(dest.Elem().Interface(), fixt.data, protocmp.Transform()))
	}
}

func alloc(t reflect.Type) reflect.Value {
	if t == nil {
		return reflect.ValueOf(new(interface{}))
	}
	return reflect.New(t)
}

func TestJSONBuiltinUnmarshalFieldKnownErrors(t *testing.T) {
	var m JSONBuiltin
	for _, fixt := range builtinKnownErrors {
		dest := reflect.New(reflect.TypeOf(fixt.data))
		assert.Error(t, m.Unmarshal([]byte(fixt.json), dest.Interface()), "m.Unmarshal(%q, dest) succeeded; want an error", fixt.json)
	}
}

func TestJSONBuiltinEncoder(t *testing.T) {
	var m JSONBuiltin
	msg := &testpb.SimpleMessage{Id: "foo"}

	var buf bytes.Buffer
	enc := m.NewEncoder(&buf)
	assert.NoError(t, enc.Encode(msg), "enc.Encode(%v)", msg)

	got := new(testpb.SimpleMessage)
	assert.NoError(t, json.Unmarshal(buf.Bytes(), got), "json.Unmarshal(%q, got)", buf.String())
	assert.Empty(t, cmp.Diff(got, msg, protocmp.Transform()))
}

func TestJSONBuiltinEncoderFields(t *testing.T) {
	var m JSONBuiltin
	for _, fixt := range builtinFieldFixtures {
		var buf bytes.Buffer
		enc := m.NewEncoder(&buf)

		if fixt.indent != "" {
			if e, ok := enc.(*json.Encoder); ok {
				e.SetIndent("", fixt.indent)
			} else {
				assert.IsTypef(t, &json.Encoder{}, enc, "enc is not *json.Encoder, unable to set indentation settings")
			}
		}

		assert.NoError(t, enc.Encode(fixt.data), "enc.Encode(%#v)", fixt.data)
		assert.Equal(t, fixt.json+"\n", buf.String(), "data = %#v", fixt.data)
	}
}

func TestJSONBuiltinDecoder(t *testing.T) {
	var m JSONBuiltin
	got := new(testpb.SimpleMessage)
	data := `{"id": "foo"}`

	r := strings.NewReader(data)
	dec := m.NewDecoder(r)
	assert.NoError(t, dec.Decode(got), "m.Unmarshal(got)")

	want := &testpb.SimpleMessage{Id: "foo"}
	assert.Empty(t, cmp.Diff(got, want, protocmp.Transform()))
}

func TestJSONBuiltinDecoderFields(t *testing.T) {
	var m JSONBuiltin
	for _, fixt := range builtinFieldFixtures {
		r := strings.NewReader(fixt.json)
		dec := m.NewDecoder(r)
		dest := alloc(reflect.TypeOf(fixt.data))
		assert.NoError(t, dec.Decode(dest.Interface()), "dec.Decode(dest); data = %q", fixt.json)
		assert.Empty(t, cmp.Diff(dest.Elem().Interface(), fixt.data, protocmp.Transform()))
	}
}

var (
	defaultIndent = "  "

	builtinFieldFixtures = []struct {
		data   interface{}
		indent string
		json   string
	}{
		{data: "", json: `""`},
		{data: "", indent: defaultIndent, json: `""`},
		{data: proto.String(""), json: `""`},
		{data: proto.String(""), indent: defaultIndent, json: `""`},
		{data: "foo", json: `"foo"`},
		{data: "foo", indent: defaultIndent, json: `"foo"`},
		{data: []byte("foo"), json: `"Zm9v"`},
		{data: []byte("foo"), indent: defaultIndent, json: `"Zm9v"`},
		{data: []byte{}, json: `""`},
		{data: []byte{}, indent: defaultIndent, json: `""`},
		{data: proto.String("foo"), json: `"foo"`},
		{data: proto.String("foo"), indent: defaultIndent, json: `"foo"`},
		{data: int32(-1), json: "-1"},
		{data: int32(-1), indent: defaultIndent, json: "-1"},
		{data: proto.Int32(-1), json: "-1"},
		{data: proto.Int32(-1), indent: defaultIndent, json: "-1"},
		{data: int64(-1), json: "-1"},
		{data: int64(-1), indent: defaultIndent, json: "-1"},
		{data: proto.Int64(-1), json: "-1"},
		{data: proto.Int64(-1), indent: defaultIndent, json: "-1"},
		{data: uint32(123), json: "123"},
		{data: uint32(123), indent: defaultIndent, json: "123"},
		{data: proto.Uint32(123), json: "123"},
		{data: proto.Uint32(123), indent: defaultIndent, json: "123"},
		{data: uint64(123), json: "123"},
		{data: uint64(123), indent: defaultIndent, json: "123"},
		{data: proto.Uint64(123), json: "123"},
		{data: proto.Uint64(123), indent: defaultIndent, json: "123"},
		{data: float32(-1.5), json: "-1.5"},
		{data: float32(-1.5), indent: defaultIndent, json: "-1.5"},
		{data: proto.Float32(-1.5), json: "-1.5"},
		{data: proto.Float32(-1.5), indent: defaultIndent, json: "-1.5"},
		{data: float64(-1.5), json: "-1.5"},
		{data: float64(-1.5), indent: defaultIndent, json: "-1.5"},
		{data: proto.Float64(-1.5), json: "-1.5"},
		{data: proto.Float64(-1.5), indent: defaultIndent, json: "-1.5"},
		{data: true, json: "true"},
		{data: true, indent: defaultIndent, json: "true"},
		{data: proto.Bool(true), json: "true"},
		{data: proto.Bool(true), indent: defaultIndent, json: "true"},
		{data: (*string)(nil), json: "null"},
		{data: (*string)(nil), indent: defaultIndent, json: "null"},
		{data: new(emptypb.Empty), json: "{}"},
		{data: new(emptypb.Empty), indent: defaultIndent, json: "{}"},
		{data: testpb.NumericEnum_ONE, json: "1"},
		{data: testpb.NumericEnum_ONE, indent: defaultIndent, json: "1"},
		{data: nil, json: "null"},
		{data: nil, indent: defaultIndent, json: "null"},
		{data: (*string)(nil), json: "null"},
		{data: (*string)(nil), indent: defaultIndent, json: "null"},
		{data: []interface{}{nil, "foo", -1.0, 1.234, true}, json: `[null,"foo",-1,1.234,true]`},
		{data: []interface{}{nil, "foo", -1.0, 1.234, true}, indent: defaultIndent, json: "[\n  null,\n  \"foo\",\n  -1,\n  1.234,\n  true\n]"},
		{data: map[string]interface{}{"bar": nil, "baz": -1.0, "fiz": 1.234, "foo": true}, json: `{"bar":null,"baz":-1,"fiz":1.234,"foo":true}`},
		{data: map[string]interface{}{"bar": nil, "baz": -1.0, "fiz": 1.234, "foo": true}, indent: defaultIndent, json: "{\n  \"bar\": null,\n  \"baz\": -1,\n  \"fiz\": 1.234,\n  \"foo\": true\n}"},
		{data: (*testpb.NumericEnum)(proto.Int32(int32(testpb.NumericEnum_ONE))), json: "1"},
		{data: (*testpb.NumericEnum)(proto.Int32(int32(testpb.NumericEnum_ONE))), indent: defaultIndent, json: "1"},
		{data: map[string]int{"FOO": 0, "BAR": -1}, json: "{\"BAR\":-1,\"FOO\":0}"},
		{data: map[string]int{"FOO": 0, "BAR": -1}, indent: defaultIndent, json: "{\n  \"BAR\": -1,\n  \"FOO\": 0\n}"},
		{data: struct {
			A string
			B int
			C map[string]int
		}{A: "Go", B: 3, C: map[string]int{"FOO": 0, "BAR": -1}}, json: "{\"A\":\"Go\",\"B\":3,\"C\":{\"BAR\":-1,\"FOO\":0}}"},
		{data: struct {
			A string
			B int
			C map[string]int
		}{A: "Go", B: 3, C: map[string]int{"FOO": 0, "BAR": -1}}, indent: defaultIndent, json: "{\n  \"A\": \"Go\",\n  \"B\": 3,\n  \"C\": {\n    \"BAR\": -1,\n    \"FOO\": 0\n  }\n}"},
	}

	builtinKnownErrors = []struct {
		data interface{}
		json string
	}{
		{data: testpb.NumericEnum_ONE, json: "ONE"},
		{data: (*testpb.NumericEnum)(proto.Int32(int32(testpb.NumericEnum_ONE))), json: "ONE"},
		{data: &testpb.ABitOfEverything_OneofString{OneofString: "abc"}, json: `"abc"`},
		{data: &timestamppb.Timestamp{Seconds: 1462875553, Nanos: 123000000}, json: `"2016-05-10T10:19:13.123Z"`},
		{data: wrapperspb.Int32(123), json: "123"},
	}
)
