/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\unescape.go
 * @Description: 路径段 percent-decoding，支持多种 unescaping 模式
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import "strings"

// unescapePath 对路径段执行 percent-decoding
func unescapePath(s string, mode int) (string, error) {
	if !strings.Contains(s, "%") {
		return s, nil
	}

	// 统计 % 并验证格式
	n := 0
	for i := 0; i < len(s); {
		if s[i] == '%' {
			n++
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				trimmed := s[i:]
				if len(trimmed) > 3 {
					trimmed = trimmed[:3]
				}
				return "", MalformedSequenceError(trimmed)
			}
			i += 3
		} else {
			i++
		}
	}

	if n == 0 {
		return s, nil
	}

	var t strings.Builder
	t.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			c := unhex(s[i+1])<<4 | unhex(s[i+2])
			if shouldUnescape(c, mode) {
				t.WriteByte(c)
				i += 2
				continue
			}
			fallthrough
		default:
			t.WriteByte(s[i])
		}
	}
	return t.String(), nil
}

// ishex 判断字节是否为十六进制字符
func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

// unhex 将十六进制字符转换为数值
func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// shouldUnescape 根据 unescaping 模式判断是否解码该字符
func shouldUnescape(c byte, mode int) bool {
	switch mode {
	case 1: // AllExceptReserved
		if isRFC6570Reserved(c) {
			return false
		}
	case 2: // AllExceptSlash
		if c == '/' {
			return false
		}
	case 3: // AllCharacters
		return true
	}
	return true
}

// isRFC6570Reserved 判断字符是否为 RFC 6570 保留字符
func isRFC6570Reserved(c byte) bool {
	switch c {
	case '!', '#', '$', '&', '\'', '(', ')', '*',
		'+', ',', '/', ':', ';', '=', '?', '@', '[', ']':
		return true
	default:
		return false
	}
}

// MalformedSequenceError 路径中畸形的 percent-encoding 错误
type MalformedSequenceError string

// Error 返回错误信息
func (e MalformedSequenceError) Error() string {
	return "malformed path escape " + string(e)
}
