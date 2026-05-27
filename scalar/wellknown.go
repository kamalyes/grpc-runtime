/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:16:00
 * @FilePath: \grpc-runtime\scalar\wellknown.go
 * @Description: protobuf well-known 类型转换（Timestamp/Duration/Enum/Wrapper）
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package scalar

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Timestamp 将 RFC3339 字符串解析为 timestamppb.Timestamp
func Timestamp(val string) (*timestamppb.Timestamp, error) {
	var r timestamppb.Timestamp
	quoted := strconv.Quote(strings.Trim(val, `"`))
	unmarshaler := protojson.UnmarshalOptions{}
	if err := unmarshaler.Unmarshal([]byte(quoted), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Duration 将字符串解析为 durationpb.Duration
func Duration(val string) (*durationpb.Duration, error) {
	var r durationpb.Duration
	quoted := strconv.Quote(strings.Trim(val, `"`))
	unmarshaler := protojson.UnmarshalOptions{}
	if err := unmarshaler.Unmarshal([]byte(quoted), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Enum 将字符串解析为枚举 int32 值
// 先按名称查找 enumValMap，找不到则按数值解析并校验是否在枚举范围内
func Enum(val string, enumValMap map[string]int32) (int32, error) {
	if e, ok := enumValMap[val]; ok {
		return e, nil
	}
	i, err := Int32(val)
	if err != nil {
		return 0, fmt.Errorf("%s is not valid", val)
	}
	for _, v := range enumValMap {
		if v == i {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%s is not valid", val)
}

// EnumSlice 将 sep 分隔的字符串拆分后逐个解析为枚举 int32 切片
func EnumSlice(val, sep string, enumValMap map[string]int32) ([]int32, error) {
	return ParseSlice(val, sep, func(s string) (int32, error) {
		return Enum(s, enumValMap)
	})
}

// --- google.protobuf.wrappers ---

// StringValue 将字符串包装为 wrapperspb.StringValue
func StringValue(val string) (*wrapperspb.StringValue, error) {
	return wrapperspb.String(val), nil
}

// FloatValue 将字符串解析后包装为 wrapperspb.FloatValue
func FloatValue(val string) (*wrapperspb.FloatValue, error) {
	parsed, err := Float32(val)
	return wrapperspb.Float(parsed), err
}

// DoubleValue 将字符串解析后包装为 wrapperspb.DoubleValue
func DoubleValue(val string) (*wrapperspb.DoubleValue, error) {
	parsed, err := Float64(val)
	return wrapperspb.Double(parsed), err
}

// BoolValue 将字符串解析后包装为 wrapperspb.BoolValue
func BoolValue(val string) (*wrapperspb.BoolValue, error) {
	parsed, err := Bool(val)
	return wrapperspb.Bool(parsed), err
}

// Int32Value 将字符串解析后包装为 wrapperspb.Int32Value
func Int32Value(val string) (*wrapperspb.Int32Value, error) {
	parsed, err := Int32(val)
	return wrapperspb.Int32(parsed), err
}

// UInt32Value 将字符串解析后包装为 wrapperspb.UInt32Value
func UInt32Value(val string) (*wrapperspb.UInt32Value, error) {
	parsed, err := Uint32(val)
	return wrapperspb.UInt32(parsed), err
}

// Int64Value 将字符串解析后包装为 wrapperspb.Int64Value
func Int64Value(val string) (*wrapperspb.Int64Value, error) {
	parsed, err := Int64(val)
	return wrapperspb.Int64(parsed), err
}

// UInt64Value 将字符串解析后包装为 wrapperspb.UInt64Value
func UInt64Value(val string) (*wrapperspb.UInt64Value, error) {
	parsed, err := Uint64(val)
	return wrapperspb.UInt64(parsed), err
}

// BytesValue 将字符串解码后包装为 wrapperspb.BytesValue
func BytesValue(val string) (*wrapperspb.BytesValue, error) {
	parsed, err := Bytes(val)
	return wrapperspb.Bytes(parsed), err
}

// --- proto2 指针转换 ---

// StringP 返回字符串指针
func StringP(val string) (*string, error) {
	return proto.String(val), nil
}

// BoolP 将字符串解析为 bool 指针
func BoolP(val string) (*bool, error) {
	b, err := Bool(val)
	if err != nil {
		return nil, err
	}
	return proto.Bool(b), nil
}

// Float64P 将字符串解析为 float64 指针
func Float64P(val string) (*float64, error) {
	f, err := Float64(val)
	if err != nil {
		return nil, err
	}
	return proto.Float64(f), nil
}

// Float32P 将字符串解析为 float32 指针
func Float32P(val string) (*float32, error) {
	f, err := Float32(val)
	if err != nil {
		return nil, err
	}
	return proto.Float32(f), nil
}

// Int64P 将字符串解析为 int64 指针
func Int64P(val string) (*int64, error) {
	i, err := Int64(val)
	if err != nil {
		return nil, err
	}
	return proto.Int64(i), nil
}

// Int32P 将字符串解析为 int32 指针
func Int32P(val string) (*int32, error) {
	i, err := Int32(val)
	if err != nil {
		return nil, err
	}
	return proto.Int32(i), err
}

// Uint64P 将字符串解析为 uint64 指针
func Uint64P(val string) (*uint64, error) {
	i, err := Uint64(val)
	if err != nil {
		return nil, err
	}
	return proto.Uint64(i), err
}

// Uint32P 将字符串解析为 uint32 指针
func Uint32P(val string) (*uint32, error) {
	i, err := Uint32(val)
	if err != nil {
		return nil, err
	}
	return proto.Uint32(i), err
}
