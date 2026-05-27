/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\scalar\convert_test.go
 * @Description: 标量转换单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package scalar

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	v, err := String("hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", v)
}

func TestBool(t *testing.T) {
	v, err := Bool("true")
	assert.NoError(t, err)
	assert.True(t, v)

	_, err = Bool("invalid")
	assert.Error(t, err)
}

func TestInt64(t *testing.T) {
	v, err := Int64("42")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), v)

	_, err = Int64("abc")
	assert.Error(t, err)
}

func TestInt32(t *testing.T) {
	v, err := Int32("42")
	assert.NoError(t, err)
	assert.Equal(t, int32(42), v)

	_, err = Int32("abc")
	assert.Error(t, err)
}

func TestUint64(t *testing.T) {
	v, err := Uint64("42")
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), v)

	_, err = Uint64("-1")
	assert.Error(t, err)
}

func TestUint32(t *testing.T) {
	v, err := Uint32("42")
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), v)

	_, err = Uint32("-1")
	assert.Error(t, err)
}

func TestFloat64(t *testing.T) {
	v, err := Float64("3.14")
	assert.NoError(t, err)
	assert.InDelta(t, 3.14, v, 0.001)

	_, err = Float64("abc")
	assert.Error(t, err)
}

func TestFloat32(t *testing.T) {
	v, err := Float32("3.14")
	assert.NoError(t, err)
	assert.InDelta(t, float32(3.14), v, 0.001)

	_, err = Float32("abc")
	assert.Error(t, err)
}

func TestBytes(t *testing.T) {
	// StdEncoding
	v, err := Bytes("aGVsbG8=")
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), v)

	// URLEncoding
	urlEncoded := base64.URLEncoding.EncodeToString([]byte("world"))
	v, err = Bytes(urlEncoded)
	assert.NoError(t, err)
	assert.Equal(t, []byte("world"), v)

	// Invalid
	_, err = Bytes("!!!invalid!!!")
	assert.Error(t, err)
}

func TestParseSlice(t *testing.T) {
	v, err := ParseSlice("1,2,3", ",", Int64)
	assert.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3}, v)

	// 错误的元素
	_, err = ParseSlice("1,abc,3", ",", Int64)
	assert.Error(t, err)
}

func TestStringSlice(t *testing.T) {
	v, err := StringSlice("a,b,c", ",")
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, v)
}

func TestBoolSlice(t *testing.T) {
	v, err := BoolSlice("true,false,true", ",")
	assert.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, v)
}

func TestFloat64Slice(t *testing.T) {
	v, err := Float64Slice("1.1,2.2,3.3", ",")
	assert.NoError(t, err)
	assert.InDelta(t, 1.1, v[0], 0.001)
	assert.InDelta(t, 2.2, v[1], 0.001)
}

func TestFloat32Slice(t *testing.T) {
	v, err := Float32Slice("1.1,2.2", ",")
	assert.NoError(t, err)
	assert.InDelta(t, float32(1.1), v[0], 0.001)
}

func TestInt64Slice(t *testing.T) {
	v, err := Int64Slice("1,2,3", ",")
	assert.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3}, v)
}

func TestInt32Slice(t *testing.T) {
	v, err := Int32Slice("1,2,3", ",")
	assert.NoError(t, err)
	assert.Equal(t, []int32{1, 2, 3}, v)
}

func TestUint64Slice(t *testing.T) {
	v, err := Uint64Slice("1,2,3", ",")
	assert.NoError(t, err)
	assert.Equal(t, []uint64{1, 2, 3}, v)
}

func TestUint32Slice(t *testing.T) {
	v, err := Uint32Slice("1,2,3", ",")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{1, 2, 3}, v)
}

func TestBytesSlice(t *testing.T) {
	v, err := BytesSlice("aGVsbG8=,d29ybGQ=", ",")
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("hello"), []byte("world")}, v)
}

func TestEnum(t *testing.T) {
	enumMap := map[string]int32{"FOO": 0, "BAR": 1}

	v, err := Enum("FOO", enumMap)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), v)

	v, err = Enum("1", enumMap)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), v)

	_, err = Enum("INVALID", enumMap)
	assert.Error(t, err)

	_, err = Enum("99", enumMap)
	assert.Error(t, err)
}

func TestEnumSlice(t *testing.T) {
	enumMap := map[string]int32{"FOO": 0, "BAR": 1}
	v, err := EnumSlice("FOO,BAR", ",", enumMap)
	assert.NoError(t, err)
	assert.Equal(t, []int32{0, 1}, v)
}

func TestTimestamp(t *testing.T) {
	v, err := Timestamp("2025-01-01T00:00:00Z")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	_, err = Timestamp("invalid")
	assert.Error(t, err)
}

func TestDuration(t *testing.T) {
	v, err := Duration("1.5s")
	assert.NoError(t, err)
	assert.NotNil(t, v)

	_, err = Duration("invalid")
	assert.Error(t, err)
}

func TestWrapperTypes(t *testing.T) {
	sv, err := StringValue("hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", sv.GetValue())

	fv, err := FloatValue("3.14")
	assert.NoError(t, err)
	assert.InDelta(t, float32(3.14), fv.GetValue(), 0.001)

	dv, err := DoubleValue("3.14")
	assert.NoError(t, err)
	assert.InDelta(t, 3.14, dv.GetValue(), 0.001)

	bv, err := BoolValue("true")
	assert.NoError(t, err)
	assert.True(t, bv.GetValue())

	iv, err := Int32Value("42")
	assert.NoError(t, err)
	assert.Equal(t, int32(42), iv.GetValue())

	u32v, err := UInt32Value("42")
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), u32v.GetValue())

	i64v, err := Int64Value("42")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), i64v.GetValue())

	u64v, err := UInt64Value("42")
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), u64v.GetValue())

	byV, err := BytesValue("aGVsbG8=")
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), byV.GetValue())
}

func TestWrapperErrors(t *testing.T) {
	_, err := FloatValue("abc")
	assert.Error(t, err)

	_, err = DoubleValue("abc")
	assert.Error(t, err)

	_, err = BoolValue("abc")
	assert.Error(t, err)

	_, err = Int32Value("abc")
	assert.Error(t, err)

	_, err = UInt32Value("-1")
	assert.Error(t, err)

	_, err = Int64Value("abc")
	assert.Error(t, err)

	_, err = UInt64Value("-1")
	assert.Error(t, err)

	_, err = BytesValue("!!!invalid!!!")
	assert.Error(t, err)
}

func TestPointerTypes(t *testing.T) {
	sp, err := StringP("hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", *sp)

	ip, err := Int64P("42")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), *ip)

	bp, err := BoolP("true")
	assert.NoError(t, err)
	assert.True(t, *bp)

	fp, err := Float64P("3.14")
	assert.NoError(t, err)
	assert.InDelta(t, 3.14, *fp, 0.001)

	f32p, err := Float32P("3.14")
	assert.NoError(t, err)
	assert.InDelta(t, float32(3.14), *f32p, 0.001)

	i32p, err := Int32P("42")
	assert.NoError(t, err)
	assert.Equal(t, int32(42), *i32p)

	u64p, err := Uint64P("42")
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), *u64p)

	u32p, err := Uint32P("42")
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), *u32p)
}

func TestPointerErrors(t *testing.T) {
	_, err := BoolP("abc")
	assert.Error(t, err)

	_, err = Float64P("abc")
	assert.Error(t, err)

	_, err = Float32P("abc")
	assert.Error(t, err)

	_, err = Int64P("abc")
	assert.Error(t, err)

	_, err = Int32P("abc")
	assert.Error(t, err)

	_, err = Uint64P("-1")
	assert.Error(t, err)

	_, err = Uint32P("-1")
	assert.Error(t, err)
}
