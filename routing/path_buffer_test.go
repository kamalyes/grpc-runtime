/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:52:22
 * @FilePath: \grpc-runtime\routing\path_buffer_test.go
 * @Description: PathBuffer 单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathBufferSplit(t *testing.T) {
	var buf PathBuffer

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"Empty", "", []string(nil)},
		{"Root", "/", []string{""}},
		{"Simple", "/v1/users", []string{"v1", "users"}},
		{"NoLeadingSlash", "v1/users", []string{"v1", "users"}},
		{"TrailingSlash", "/v1/users/", []string{"v1", "users", ""}},
		{"MultipleSlashes", "/a/b/c/d", []string{"a", "b", "c", "d"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buf.Split(tt.input, false)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestPathBufferSplitEncodedSlash(t *testing.T) {
	var buf PathBuffer

	// encodedSlashSplitter 按 / 或 %2F 分割，%2F 被当作分隔符
	result := buf.Split("/v1/users%2F123", true)
	assert.Equal(t, []string{"v1", "users", "123"}, result)
}

func TestPathBufferReuse(t *testing.T) {
	var buf PathBuffer

	// 第一次分割
	result := buf.Split("/v1/users", false)
	assert.Equal(t, []string{"v1", "users"}, result)

	// 第二次分割复用底层 slice
	result = buf.Split("/v2/tenants", false)
	assert.Equal(t, []string{"v2", "tenants"}, result)
}
