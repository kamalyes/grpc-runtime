package runtime

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kamalyes/grpc-runtime/testpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestJSONPbMarshal(t *testing.T) {
	msg := testpb.ABitOfEverything{
		SingleNested:        &testpb.ABitOfEverything_Nested{},
		RepeatedStringValue: []string{},
		MappedStringValue:   map[string]string{},
		MappedNestedValue:   map[string]*testpb.ABitOfEverything_Nested{},
		RepeatedEnumValue:   []testpb.NumericEnum{},
		TimestampValue:      &timestamppb.Timestamp{},
		Uuid:                "6EC2446F-7E89-4127-B3E6-5C05E6BECBA7",
		Nested: []*testpb.ABitOfEverything_Nested{
			{Name: "foo", Amount: 12345},
		},
		Uint64Value: 0xFFFFFFFFFFFFFFFF,
		EnumValue:   testpb.NumericEnum_ONE,
		OneofValue: &testpb.ABitOfEverything_OneofString{
			OneofString: "bar",
		},
		MapValue: map[string]testpb.NumericEnum{
			"a": testpb.NumericEnum_ONE,
			"b": testpb.NumericEnum_ZERO,
		},
		RepeatedEnumAnnotation:   []testpb.NumericEnum{},
		EnumValueAnnotation:      testpb.NumericEnum_ONE,
		RepeatedStringAnnotation: []string{},
		RepeatedNestedAnnotation: []*testpb.ABitOfEverything_Nested{},
		NestedAnnotation:         &testpb.ABitOfEverything_Nested{},
	}

	for i, spec := range []struct {
		useEnumNumbers, emitUnpopulated bool
		indent                          string
		useProtoNames                   bool
		verifier                        func(t *testing.T, json string)
	}{
		{verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, "ONE"), `strings.Contains(%q, "ONE") = false; want true`, json)
			assert.True(t, strings.Contains(json, "uint64Value"), `strings.Contains(%q, "uint64Value") = false; want true`, json)
		}},
		{useEnumNumbers: true, verifier: func(t *testing.T, json string) {
			assert.False(t, strings.Contains(json, "ONE"), `strings.Contains(%q, "ONE") = true; want false`, json)
		}},
		{emitUnpopulated: true, verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, `"sfixed32Value"`), `strings.Contains(%q, "sfixed32Value") = false; want true`, json)
		}},
		{indent: "\t\t", verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, "\t\t\"amount\":"), `strings.Contains(%q, "\t\t\"amount\":") = false; want true`, json)
		}},
		{useProtoNames: true, verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, "uint64_value"), `strings.Contains(%q, "uint64_value") = false; want true`, json)
		}},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: spec.emitUnpopulated,
					Indent:          spec.indent,
					UseProtoNames:   spec.useProtoNames,
					UseEnumNumbers:  spec.useEnumNumbers,
				},
			}
			buf, err := m.Marshal(&msg)
			assert.NoError(t, err, "m.Marshal(%v); spec=%v", &msg, spec)

			var got testpb.ABitOfEverything
			unmarshaler := &protojson.UnmarshalOptions{}
			assert.NoError(t, unmarshaler.Unmarshal(buf, &got), "jsonpb.UnmarshalString(%q, &got); spec=%v", string(buf), spec)
			assert.Empty(t, cmp.Diff(&got, &msg, protocmp.Transform()), "case %d: spec=%v", i, spec)
			if spec.verifier != nil {
				spec.verifier(t, string(buf))
			}
		})
	}
}

func TestJSONPbMarshalFields(t *testing.T) {
	var m JSONPb
	m.UseEnumNumbers = true

	for _, spec := range builtinFieldFixtures {
		m.Indent = spec.indent
		buf, err := m.Marshal(spec.data)
		assert.NoError(t, err, "m.Marshal(%#v)", spec.data)
		assert.Equal(t, spec.json, string(buf), "m.Marshal(%#v)", spec.data)
	}

	m.Indent = ""

	nums := []testpb.NumericEnum{testpb.NumericEnum_ZERO, testpb.NumericEnum_ONE}
	buf, err := m.Marshal(nums)
	assert.NoError(t, err, "m.Marshal(%#v)", nums)
	assert.Equal(t, `[0,1]`, string(buf))

	m.UseEnumNumbers = false
	buf, err = m.Marshal(testpb.NumericEnum_ONE)
	assert.NoError(t, err, "m.Marshal(%#v)", testpb.NumericEnum_ONE)
	assert.Equal(t, `"ONE"`, string(buf))

	buf, err = m.Marshal(nums)
	assert.NoError(t, err, "m.Marshal(%#v)", nums)
	assert.Equal(t, `["ZERO","ONE"]`, string(buf))
}

func TestJSONPbUnmarshal(t *testing.T) {
	var m JSONPb
	var got testpb.ABitOfEverything

	want := testpb.ABitOfEverything{
		Uuid: "6EC2446F-7E89-4127-B3E6-5C05E6BECBA7",
		Nested: []*testpb.ABitOfEverything_Nested{
			{Name: "foo", Amount: 12345},
		},
		Uint64Value: 0xFFFFFFFFFFFFFFFF,
		EnumValue:   testpb.NumericEnum_ONE,
		OneofValue: &testpb.ABitOfEverything_OneofString{
			OneofString: "bar",
		},
		MapValue: map[string]testpb.NumericEnum{
			"a": testpb.NumericEnum_ONE,
			"b": testpb.NumericEnum_ZERO,
		},
	}

	for i, data := range []string{
		`{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","nested":[{"name":"foo","amount":12345}],"uint64Value":18446744073709551615,"enumValue":"ONE","oneofString":"bar","mapValue":{"a":1,"b":0}}`,
		`{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","nested":[{"name":"foo","amount":12345}],"uint64Value":"18446744073709551615","enumValue":"ONE","oneofString":"bar","mapValue":{"a":1,"b":0}}`,
		`{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","nested":[{"name":"foo","amount":12345}],"uint64Value":18446744073709551615,"enumValue":1,"oneofString":"bar","mapValue":{"a":1,"b":0}}`,
	} {
		assert.NoError(t, m.Unmarshal([]byte(data), &got), "case %d: m.Unmarshal(%q, &got)", i, data)
		assert.Empty(t, cmp.Diff(&got, &want, protocmp.Transform()), "case %d", i)
	}
}

func TestJSONPbUnmarshalFields(t *testing.T) {
	var m JSONPb
	for _, fixt := range fieldFixtures {
		if fixt.skipUnmarshal {
			continue
		}
		dest := reflect.New(reflect.TypeOf(fixt.data))
		assert.NoError(t, m.Unmarshal([]byte(fixt.json), dest.Interface()), "m.Unmarshal(%q, %T)", fixt.json, dest.Interface())
		assert.Empty(t, cmp.Diff(dest.Elem().Interface(), fixt.data, protocmp.Transform()), "input = %v", fixt.json)
	}
}

func TestJSONPbEncoder(t *testing.T) {
	msg := testpb.ABitOfEverything{
		SingleNested:        &testpb.ABitOfEverything_Nested{},
		RepeatedStringValue: []string{},
		MappedStringValue:   map[string]string{},
		MappedNestedValue:   map[string]*testpb.ABitOfEverything_Nested{},
		RepeatedEnumValue:   []testpb.NumericEnum{},
		TimestampValue:      &timestamppb.Timestamp{},
		Uuid:                "6EC2446F-7E89-4127-B3E6-5C05E6BECBA7",
		Nested: []*testpb.ABitOfEverything_Nested{
			{Name: "foo", Amount: 12345},
		},
		Uint64Value: 0xFFFFFFFFFFFFFFFF,
		OneofValue: &testpb.ABitOfEverything_OneofString{
			OneofString: "bar",
		},
		MapValue: map[string]testpb.NumericEnum{
			"a": testpb.NumericEnum_ONE,
			"b": testpb.NumericEnum_ZERO,
		},
		RepeatedEnumAnnotation:   []testpb.NumericEnum{},
		EnumValueAnnotation:      testpb.NumericEnum_ONE,
		RepeatedStringAnnotation: []string{},
		RepeatedNestedAnnotation: []*testpb.ABitOfEverything_Nested{},
		NestedAnnotation:         &testpb.ABitOfEverything_Nested{},
	}

	for i, spec := range []struct {
		useEnumNumbers, emitUnpopulated bool
		indent                          string
		useProtoNames                   bool
		verifier                        func(t *testing.T, json string)
	}{
		{verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, "ONE"), `strings.Contains(%q, "ONE") = false; want true`, json)
			assert.True(t, strings.Contains(json, "uint64Value"), `strings.Contains(%q, "uint64Value") = false; want true`, json)
		}},
		{useEnumNumbers: true, verifier: func(t *testing.T, json string) {
			assert.False(t, strings.Contains(json, "ONE"), `strings.Contains(%q, "ONE") = true; want false`, json)
		}},
		{emitUnpopulated: true, verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, `"sfixed32Value"`), `strings.Contains(%q, "sfixed32Value") = false; want true`, json)
		}},
		{indent: "\t\t", verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, "\t\t\"amount\":"), `strings.Contains(%q, "\t\t\"amount\":") = false; want true`, json)
		}},
		{useProtoNames: true, verifier: func(t *testing.T, json string) {
			assert.True(t, strings.Contains(json, "uint64_value"), `strings.Contains(%q, "uint64_value") = false; want true`, json)
		}},
	} {
		m := JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: spec.emitUnpopulated,
				Indent:          spec.indent,
				UseProtoNames:   spec.useProtoNames,
				UseEnumNumbers:  spec.useEnumNumbers,
			},
		}

		var buf bytes.Buffer
		enc := m.NewEncoder(&buf)
		assert.NoError(t, enc.Encode(&msg), "enc.Encode(%v); spec=%v", &msg, spec)

		var got testpb.ABitOfEverything
		unmarshaler := &protojson.UnmarshalOptions{}
		assert.NoError(t, unmarshaler.Unmarshal(buf.Bytes(), &got), "jsonpb.UnmarshalString(%q, &got); spec=%v", buf.String(), spec)
		assert.Empty(t, cmp.Diff(&got, &msg, protocmp.Transform()), "case %d", i)
		if spec.verifier != nil {
			spec.verifier(t, buf.String())
		}
	}
}

func TestJSONPbEncoderFields(t *testing.T) {
	var m JSONPb
	for _, fixt := range fieldFixtures {
		var buf bytes.Buffer
		enc := m.NewEncoder(&buf)
		assert.NoError(t, enc.Encode(fixt.data), "enc.Encode(%#v)", fixt.data)
		assert.Equal(t, fixt.json+string(m.Delimiter()), buf.String(), "enc.Encode(%#v)", fixt.data)
	}

	m.UseEnumNumbers = true
	buf, err := m.Marshal(testpb.NumericEnum_ONE)
	assert.NoError(t, err, "m.Marshal(%#v)", testpb.NumericEnum_ONE)
	assert.Equal(t, "1", string(buf))
}

func TestJSONPbDecoder(t *testing.T) {
	var m JSONPb
	var got testpb.ABitOfEverything

	want := testpb.ABitOfEverything{
		Uuid: "6EC2446F-7E89-4127-B3E6-5C05E6BECBA7",
		Nested: []*testpb.ABitOfEverything_Nested{
			{Name: "foo", Amount: 12345},
		},
		Uint64Value: 0xFFFFFFFFFFFFFFFF,
		EnumValue:   testpb.NumericEnum_ONE,
		OneofValue: &testpb.ABitOfEverything_OneofString{
			OneofString: "bar",
		},
		MapValue: map[string]testpb.NumericEnum{
			"a": testpb.NumericEnum_ONE,
			"b": testpb.NumericEnum_ZERO,
		},
	}

	for _, data := range []string{
		`{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","nested":[{"name":"foo","amount":12345}],"uint64Value":18446744073709551615,"enumValue":"ONE","oneofString":"bar","mapValue":{"a":1,"b":0}}`,
		`{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","nested":[{"name":"foo","amount":12345}],"uint64Value":"18446744073709551615","enumValue":"ONE","oneofString":"bar","mapValue":{"a":1,"b":0}}`,
		`{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","nested":[{"name":"foo","amount":12345}],"uint64Value":18446744073709551615,"enumValue":1,"oneofString":"bar","mapValue":{"a":1,"b":0}}`,
	} {
		r := strings.NewReader(data)
		dec := m.NewDecoder(r)
		assert.NoError(t, dec.Decode(&got), "m.Unmarshal(&got); data=%q", data)
		assert.Empty(t, cmp.Diff(&got, &want, protocmp.Transform()), "data %q", data)
	}
}

func TestJSONPbDecoderFields(t *testing.T) {
	var m JSONPb
	for _, fixt := range fieldFixtures {
		if fixt.skipUnmarshal {
			continue
		}
		dest := reflect.New(reflect.TypeOf(fixt.data))
		dec := m.NewDecoder(strings.NewReader(fixt.json))
		assert.NoError(t, dec.Decode(dest.Interface()), "dec.Decode(%T); input = %q", dest.Interface(), fixt.json)
		assert.Equal(t, fixt.data, dest.Elem().Interface(), "input = %v", fixt.json)
	}
}

func TestJSONPbDecoderUnknownField(t *testing.T) {
	m := JSONPb{UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: false}}
	var got testpb.ABitOfEverything
	data := `{"uuid":"6EC2446F-7E89-4127-B3E6-5C05E6BECBA7","unknownField":"111"}`

	r := strings.NewReader(data)
	dec := m.NewDecoder(r)
	assert.Error(t, dec.Decode(&got), "m.Unmarshal(&got) not failed; want `unknown field` error; data=%q", data)
}

var (
	fieldFixtures = []struct {
		data          interface{}
		json          string
		skipUnmarshal bool
	}{
		{data: int32(1), json: "1"},
		{data: proto.Int32(1), json: "1"},
		{data: int64(1), json: "1"},
		{data: proto.Int64(1), json: "1"},
		{data: uint32(1), json: "1"},
		{data: proto.Uint32(1), json: "1"},
		{data: uint64(1), json: "1"},
		{data: proto.Uint64(1), json: "1"},
		{data: "abc", json: `"abc"`},
		{data: []byte("abc"), json: `"YWJj"`},
		{data: []byte{}, json: `""`},
		{data: proto.String("abc"), json: `"abc"`},
		{data: float32(1.5), json: "1.5"},
		{data: proto.Float32(1.5), json: "1.5"},
		{data: float64(1.5), json: "1.5"},
		{data: proto.Float64(1.5), json: "1.5"},
		{data: true, json: "true"},
		{data: false, json: "false"},
		{data: (*string)(nil), json: "null"},
		{data: testpb.NumericEnum_ONE, json: `"ONE"`, skipUnmarshal: true},
		{data: (*testpb.NumericEnum)(proto.Int32(int32(testpb.NumericEnum_ONE))), json: `"ONE"`, skipUnmarshal: true},
		{data: map[string]int32{"foo": 1}, json: `{"foo":1}`},
		{data: map[string]*testpb.SimpleMessage{"foo": {Id: "bar"}}, json: `{"foo":{"id":"bar"}}`},
		{data: map[int32]*testpb.SimpleMessage{1: {Id: "foo"}}, json: `{"1":{"id":"foo"}}`},
		{data: map[bool]*testpb.SimpleMessage{true: {Id: "foo"}}, json: `{"true":{"id":"foo"}}`},
		{data: &durationpb.Duration{Seconds: 123, Nanos: 456000000}, json: `"123.456s"`},
		{data: &timestamppb.Timestamp{Seconds: 1462875553, Nanos: 123000000}, json: `"2016-05-10T10:19:13.123Z"`},
		{data: new(emptypb.Empty), json: "{}"},
		{data: &structpb.Value{Kind: new(structpb.Value_NullValue)}, json: "null", skipUnmarshal: true},
		{data: &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 123.4}}, json: "123.4", skipUnmarshal: true},
		{data: &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "abc"}}, json: `"abc"`, skipUnmarshal: true},
		{data: &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}}, json: "true", skipUnmarshal: true},
		{data: &structpb.Struct{Fields: map[string]*structpb.Value{"foo_bar": {Kind: &structpb.Value_BoolValue{BoolValue: true}}}}, json: `{"foo_bar":true}`, skipUnmarshal: true},
		{data: wrapperspb.Bool(true), json: "true"},
		{data: wrapperspb.Double(123.456), json: "123.456"},
		{data: wrapperspb.Float(123.456), json: "123.456"},
		{data: wrapperspb.Int32(-123), json: "-123"},
		{data: wrapperspb.Int64(-123), json: `"-123"`},
		{data: wrapperspb.UInt32(123), json: "123"},
		{data: wrapperspb.UInt64(123), json: `"123"`},
	}
)

func TestJSONPbUnmarshalNullField(t *testing.T) {
	var out map[string]interface{}
	const json = `{"foo": null}`
	marshaler := &JSONPb{}
	assert.NoError(t, marshaler.Unmarshal([]byte(json), &out))

	value, hasKey := out["foo"]
	assert.True(t, hasKey, "unmarshaled map did not have key 'foo'")
	assert.Nil(t, value)
}

func TestJSONPbMarshalResponseBodies(t *testing.T) {
	marshaler := &JSONPb{}
	for i, spec := range []struct {
		input           interface{}
		emitUnpopulated bool
		verifier        func(*testing.T, interface{}, []byte)
	}{
		{input: &testpb.ResponseBodyOut{Response: &testpb.ResponseBodyOut_Response{Data: "abcdef"}}, verifier: func(t *testing.T, input interface{}, json []byte) {
			var out testpb.ResponseBodyOut
			assert.NoError(t, marshaler.Unmarshal(json, &out))
			assert.Empty(t, cmp.Diff(input, &out, protocmp.Transform()))
		}},
		{emitUnpopulated: true, input: &testpb.ResponseBodyOut{}, verifier: func(t *testing.T, input interface{}, json []byte) {
			var out testpb.ResponseBodyOut
			assert.NoError(t, marshaler.Unmarshal(json, &out))
			assert.Empty(t, cmp.Diff(input, &out, protocmp.Transform()))
		}},
		{input: &testpb.RepeatedResponseBodyOut_Response{}, verifier: func(t *testing.T, input interface{}, json []byte) {
			var out testpb.RepeatedResponseBodyOut_Response
			assert.NoError(t, marshaler.Unmarshal(json, &out))
			assert.Empty(t, cmp.Diff(input, &out, protocmp.Transform()))
		}},
		{emitUnpopulated: true, input: &testpb.RepeatedResponseBodyOut_Response{}, verifier: func(t *testing.T, input interface{}, json []byte) {
			var out testpb.RepeatedResponseBodyOut_Response
			assert.NoError(t, marshaler.Unmarshal(json, &out))
			assert.Empty(t, cmp.Diff(input, &out, protocmp.Transform()))
		}},
		{input: ([]*testpb.RepeatedResponseBodyOut_Response)(nil), verifier: func(t *testing.T, input interface{}, json []byte) {
			var out []*testpb.RepeatedResponseBodyOut_Response
			assert.NoError(t, marshaler.Unmarshal(json, &out))
			assert.Empty(t, cmp.Diff(input, out, protocmp.Transform()))
		}},
		{emitUnpopulated: true, input: []*testpb.RepeatedResponseBodyOut_Response{{}, {Data: "abc", Type: testpb.RepeatedResponseBodyOut_Response_B}}, verifier: func(t *testing.T, input interface{}, json []byte) {
			var out []*testpb.RepeatedResponseBodyOut_Response
			assert.NoError(t, marshaler.Unmarshal(json, &out))
			assert.Empty(t, cmp.Diff(input, out, protocmp.Transform()))
		}},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := JSONPb{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: spec.emitUnpopulated}}
			buf, err := m.Marshal(spec.input)
			assert.NoError(t, err, "m.Marshal(%v); spec=%v", spec.input, spec)
			if spec.verifier != nil {
				spec.verifier(t, spec.input, buf)
			}
		})
	}
}
