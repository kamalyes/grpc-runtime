/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 10:59:26
 * @FilePath: \grpc-runtime\routing\path_buffer.go
 * @Description: 可复用的路径分割缓冲区，减少请求热路径上的临时分配
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"regexp"
	"strings"
)

var encodedSlashSplitter = regexp.MustCompile("(/|%2F)")

// PathBuffer 可复用的路径分割缓冲区
// 调用方可通过 Reset 复用底层 slice，避免每次请求分配临时存储
type PathBuffer struct {
	components []string
}

// Reset 清空缓冲区
func (b *PathBuffer) Reset() {
	b.components = b.components[:0]
}

// Split 将路径按 / 分割为组件，去掉前导 /
func (b *PathBuffer) Split(path string, splitEncodedSlash bool) []string {
	if path == "" || path == "/" {
		b.components = b.components[:0]
		if path == "/" {
			b.components = append(b.components, "")
		}
		return b.components
	}
	trimmed := strings.TrimPrefix(path, "/")
	if splitEncodedSlash {
		return encodedSlashSplitter.Split(trimmed, -1)
	}
	b.components = append(b.components[:0], strings.Split(trimmed, "/")...)
	return b.components
}
