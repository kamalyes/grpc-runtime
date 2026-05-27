/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:00:00
 * @FilePath: \grpc-runtime\utilities\string_array_flag.go
 * @Description: 字符串数组命令行标志
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package utilities

import (
	"flag"
	"strings"
)

// flagInterface flag 的精简接口
type flagInterface interface {
	Var(value flag.Value, name string, usage string)
}

// StringArrayFlag 定义具有指定名称和用法的字符串数组标志
// 返回值是存储标志重复值的 StringArrayFlags 变量的地址
func StringArrayFlag(f flagInterface, name string, usage string) *StringArrayFlags {
	value := &StringArrayFlags{}
	f.Var(value, name, usage)
	return value
}

// StringArrayFlags []string 的包装，提供 flag.Var 接口
type StringArrayFlags []string

// String 返回 StringArrayFlags 的字符串表示
func (i *StringArrayFlags) String() string {
	return strings.Join(*i, ",")
}

// Set 向 StringArrayFlags 追加值
func (i *StringArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
