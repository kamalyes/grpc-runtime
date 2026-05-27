/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\scalar.go
 * @Description: 标量转换函数，委托给 scalar 子包实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"github.com/kamalyes/grpc-runtime/scalar"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// --- 基本标量转换 ---

func String(val string) (string, error)                          { return scalar.String(val) }
func Bool(val string) (bool, error)                              { return scalar.Bool(val) }
func Float64(val string) (float64, error)                       { return scalar.Float64(val) }
func Float32(val string) (float32, error)                       { return scalar.Float32(val) }
func Int64(val string) (int64, error)                            { return scalar.Int64(val) }
func Int32(val string) (int32, error)                            { return scalar.Int32(val) }
func Uint64(val string) (uint64, error)                          { return scalar.Uint64(val) }
func Uint32(val string) (uint32, error)                          { return scalar.Uint32(val) }
func Bytes(val string) ([]byte, error)                           { return scalar.Bytes(val) }

// --- 切片转换 ---

func StringSlice(val, sep string) ([]string, error)              { return scalar.StringSlice(val, sep) }
func BoolSlice(val, sep string) ([]bool, error)                  { return scalar.BoolSlice(val, sep) }
func Float64Slice(val, sep string) ([]float64, error)           { return scalar.Float64Slice(val, sep) }
func Float32Slice(val, sep string) ([]float32, error)            { return scalar.Float32Slice(val, sep) }
func Int64Slice(val, sep string) ([]int64, error)               { return scalar.Int64Slice(val, sep) }
func Int32Slice(val, sep string) ([]int32, error)               { return scalar.Int32Slice(val, sep) }
func Uint64Slice(val, sep string) ([]uint64, error)             { return scalar.Uint64Slice(val, sep) }
func Uint32Slice(val, sep string) ([]uint32, error)             { return scalar.Uint32Slice(val, sep) }
func BytesSlice(val, sep string) ([][]byte, error)              { return scalar.BytesSlice(val, sep) }

// --- well-known 类型 ---

func Timestamp(val string) (*timestamppb.Timestamp, error)      { return scalar.Timestamp(val) }
func Duration(val string) (*durationpb.Duration, error)          { return scalar.Duration(val) }
func Enum(val string, enumValMap map[string]int32) (int32, error) {
	return scalar.Enum(val, enumValMap)
}
func EnumSlice(val, sep string, enumValMap map[string]int32) ([]int32, error) {
	return scalar.EnumSlice(val, sep, enumValMap)
}

// --- wrapper 类型 ---

func StringValue(val string) (*wrapperspb.StringValue, error)    { return scalar.StringValue(val) }
func FloatValue(val string) (*wrapperspb.FloatValue, error)     { return scalar.FloatValue(val) }
func DoubleValue(val string) (*wrapperspb.DoubleValue, error)   { return scalar.DoubleValue(val) }
func BoolValue(val string) (*wrapperspb.BoolValue, error)      { return scalar.BoolValue(val) }
func Int32Value(val string) (*wrapperspb.Int32Value, error)    { return scalar.Int32Value(val) }
func UInt32Value(val string) (*wrapperspb.UInt32Value, error)  { return scalar.UInt32Value(val) }
func Int64Value(val string) (*wrapperspb.Int64Value, error)    { return scalar.Int64Value(val) }
func UInt64Value(val string) (*wrapperspb.UInt64Value, error)  { return scalar.UInt64Value(val) }
func BytesValue(val string) (*wrapperspb.BytesValue, error)    { return scalar.BytesValue(val) }

// --- proto2 指针转换 ---

func StringP(val string) (*string, error)                       { return scalar.StringP(val) }
func BoolP(val string) (*bool, error)                           { return scalar.BoolP(val) }
func Float64P(val string) (*float64, error)                     { return scalar.Float64P(val) }
func Float32P(val string) (*float32, error)                     { return scalar.Float32P(val) }
func Int64P(val string) (*int64, error)                         { return scalar.Int64P(val) }
func Int32P(val string) (*int32, error)                         { return scalar.Int32P(val) }
func Uint64P(val string) (*uint64, error)                       { return scalar.Uint64P(val) }
func Uint32P(val string) (*uint32, error)                       { return scalar.Uint32P(val) }
