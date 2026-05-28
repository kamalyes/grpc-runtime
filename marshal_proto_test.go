package runtime

import (
	"bytes"
	"testing"

	"github.com/kamalyes/grpc-runtime/testpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var message = &testpb.ABitOfEverything{
	SingleNested:        &testpb.ABitOfEverything_Nested{},
	RepeatedStringValue: nil,
	MappedStringValue:   nil,
	MappedNestedValue:   nil,
	RepeatedEnumValue:   nil,
	TimestampValue:      &timestamppb.Timestamp{},
	Uuid:                "6EC2446F-7E89-4127-B3E6-5C05E6BECBA7",
	Nested: []*testpb.ABitOfEverything_Nested{
		{
			Name:   "foo",
			Amount: 12345,
		},
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

func TestProtoMarshalUnmarshal(t *testing.T) {
	marshaller := ProtoMarshaller{}

	buffer, err := marshaller.Marshal(message)
	assert.NoError(t, err, "Marshalling returned error")

	unmarshalled := &testpb.ABitOfEverything{}
	err = marshaller.Unmarshal(buffer, unmarshalled)
	assert.NoError(t, err, "Unmarshalling returned error")

	assert.True(t, proto.Equal(unmarshalled, message), "Unmarshalled didn't match original message: (original = %v) != (unmarshalled = %v)", message, unmarshalled)
}

func TestProtoEncoderDecodert(t *testing.T) {
	marshaller := ProtoMarshaller{}

	var buf bytes.Buffer

	encoder := marshaller.NewEncoder(&buf)
	decoder := marshaller.NewDecoder(&buf)

	err := encoder.Encode(message)
	assert.NoError(t, err, "Encoding returned error")

	unencoded := &testpb.ABitOfEverything{}
	err = decoder.Decode(unencoded)
	assert.NoError(t, err, "Unmarshalling returned error")

	assert.True(t, proto.Equal(unencoded, message), "Unencoded didn't match original message: (original = %v) != (unencoded = %v)", message, unencoded)
}
