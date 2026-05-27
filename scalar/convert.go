/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\scalar\convert.go
 * @Description: 基本标量类型转换，将字符串解析为 int/float/bool/string/bytes
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package scalar

import (
	"encoding/base64"
	"strconv"
	"strings"
)

// String 直接返回输入字符串，保持与其他转换函数签名一致
func String(val string) (string, error) {
	return val, nil
}

// Bool 将字符串解析为 bool
func Bool(val string) (bool, error) {
	return strconv.ParseBool(val)
}

// Float64 将字符串解析为 float64
func Float64(val string) (float64, error) {
	return strconv.ParseFloat(val, 64)
}

// Float32 将字符串解析为 float32
func Float32(val string) (float32, error) {
	f, err := strconv.ParseFloat(val, 32)
	if err != nil {
		return 0, err
	}
	return float32(f), nil
}

// Int64 将字符串解析为 int64
func Int64(val string) (int64, error) {
	return strconv.ParseInt(val, 0, 64)
}

// Int32 将字符串解析为 int32
func Int32(val string) (int32, error) {
	i, err := strconv.ParseInt(val, 0, 32)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}

// Uint64 将字符串解析为 uint64
func Uint64(val string) (uint64, error) {
	return strconv.ParseUint(val, 0, 64)
}

// Uint32 将字符串解析为 uint32
func Uint32(val string) (uint32, error) {
	i, err := strconv.ParseUint(val, 0, 32)
	if err != nil {
		return 0, err
	}
	return uint32(i), nil
}

// Bytes 将 URL-safe base64 字符串解码为字节切片
func Bytes(val string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(val)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

// ParseSlice 泛型切片转换，将 sep 分隔的字符串拆分后逐个调用 parse 函数
// 消除 StringSlice/BoolSlice/Int64Slice 等重复代码
func ParseSlice[T any](val, sep string, parse func(string) (T, error)) ([]T, error) {
	parts := strings.Split(val, sep)
	result := make([]T, len(parts))
	for i, part := range parts {
		v, err := parse(part)
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

// StringSlice 将 sep 分隔的字符串拆分为字符串切片
func StringSlice(val, sep string) ([]string, error) {
	return strings.Split(val, sep), nil
}

// BoolSlice 将 sep 分隔的字符串拆分为 bool 切片
func BoolSlice(val, sep string) ([]bool, error) {
	return ParseSlice(val, sep, Bool)
}

// Float64Slice 将 sep 分隔的字符串拆分为 float64 切片
func Float64Slice(val, sep string) ([]float64, error) {
	return ParseSlice(val, sep, Float64)
}

// Float32Slice 将 sep 分隔的字符串拆分为 float32 切片
func Float32Slice(val, sep string) ([]float32, error) {
	return ParseSlice(val, sep, Float32)
}

// Int64Slice 将 sep 分隔的字符串拆分为 int64 切片
func Int64Slice(val, sep string) ([]int64, error) {
	return ParseSlice(val, sep, Int64)
}

// Int32Slice 将 sep 分隔的字符串拆分为 int32 切片
func Int32Slice(val, sep string) ([]int32, error) {
	return ParseSlice(val, sep, Int32)
}

// Uint64Slice 将 sep 分隔的字符串拆分为 uint64 切片
func Uint64Slice(val, sep string) ([]uint64, error) {
	return ParseSlice(val, sep, Uint64)
}

// Uint32Slice 将 sep 分隔的字符串拆分为 uint32 切片
func Uint32Slice(val, sep string) ([]uint32, error) {
	return ParseSlice(val, sep, Uint32)
}

// BytesSlice 将 sep 分隔的字符串拆分为字节切片的切片
func BytesSlice(val, sep string) ([][]byte, error) {
	return ParseSlice(val, sep, Bytes)
}
