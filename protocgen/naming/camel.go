/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:26:32
 * @FilePath: \grpc-runtime\protocgen\naming\camel.go
 * @Description: 命名转换工具，提供 snake_case 到 CamelCase/camelCase 的转换
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package naming

import (
	"path/filepath"
	"strings"
)

// Camel 将 snake_case 名称转换为 CamelCase 名称
// 内部下划线后跟小写字母时，移除下划线并将字母转为大写
// 例如：_my_field_name_2 → XMyFieldName_2
func Camel(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		t = append(t, 'X')
		i++
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		if isASCIILower(c) {
			c ^= ' '
		}
		t = append(t, c)
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// CamelIdentifier 将带包路径的标识符转换为 CamelCase
// 不影响包名/路径部分，仅转换最后的标识符
func CamelIdentifier(s string) string {
	const dot = "."
	if !strings.Contains(s, dot) {
		return Camel(s)
	}
	identifier := filepath.Ext(s)
	path := strings.TrimSuffix(s, identifier)
	identifier = strings.TrimPrefix(identifier, dot)
	return path + dot + Camel(identifier)
}

// JSONCamelCase 将 snake_case 标识符转换为 camelCase（首字母小写）
// 遵循 protobuf JSON 规范
func JSONCamelCase(s string) string {
	var b []byte
	var wasUnderscore bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '_' {
			if wasUnderscore && isASCIILower(c) {
				c -= 'a' - 'A'
			}
			b = append(b, c)
		}
		wasUnderscore = c == '_'
	}
	return string(b)
}

// isASCIILower 判断是否为 ASCII 小写字母
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// isASCIIDigit 判断是否为 ASCII 数字
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
