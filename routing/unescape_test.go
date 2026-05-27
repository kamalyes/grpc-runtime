/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\unescape_test.go
 * @Description: unescape 单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnescapePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		mode  int
		want  string
		err   bool
	}{
		{"NoPercent", "hello", 3, "hello", false},
		{"SimpleHex", "%41", 3, "A", false},
		{"MultipleHex", "%41%42%43", 3, "ABC", false},
		{"ReservedChar", "%2F", 1, "%2F", false}, // AllExceptReserved: / 不解码
		{"SlashMode2", "%2F", 2, "%2F", false},   // AllExceptSlash: / 不解码
		{"SlashMode3", "%2F", 3, "/", false},     // AllCharacters: 解码
		{"MalformedShort", "%4", 3, "", true},    // 不完整的 percent-encoding
		{"MalformedInvalid", "%GG", 3, "", true}, // 非十六进制
		{"Mixed", "hello%20world", 3, "hello world", false},
		{"LegacyMode", "%41", 0, "A", false}, // mode 0 走默认分支，仍做 unescape
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unescapePath(tt.input, tt.mode)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMalformedSequenceError(t *testing.T) {
	err := MalformedSequenceError("%GG")
	assert.Contains(t, err.Error(), "malformed path escape")
}

func TestIshex(t *testing.T) {
	assert.True(t, ishex('0'))
	assert.True(t, ishex('9'))
	assert.True(t, ishex('a'))
	assert.True(t, ishex('f'))
	assert.True(t, ishex('A'))
	assert.True(t, ishex('F'))
	assert.False(t, ishex('g'))
	assert.False(t, ishex('G'))
	assert.False(t, ishex('-'))
}

func TestUnhex(t *testing.T) {
	assert.Equal(t, byte(0), unhex('0'))
	assert.Equal(t, byte(9), unhex('9'))
	assert.Equal(t, byte(10), unhex('a'))
	assert.Equal(t, byte(15), unhex('f'))
	assert.Equal(t, byte(10), unhex('A'))
	assert.Equal(t, byte(15), unhex('F'))
	assert.Equal(t, byte(0), unhex('g'))
}

func TestIsRFC6570Reserved(t *testing.T) {
	reserved := []byte{'!', '#', '$', '&', '(', ')', '*', '+', ',', '/', ':', ';', '=', '?', '@', '[', ']'}
	for _, c := range reserved {
		assert.True(t, isRFC6570Reserved(c), "expected %c to be reserved", c)
	}
	assert.False(t, isRFC6570Reserved('a'))
	assert.False(t, isRFC6570Reserved('0'))
}
