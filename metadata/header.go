/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:16:38
 * @FilePath: \grpc-runtime\metadata\header.go
 * @Description: HTTP header 与 gRPC metadata 匹配
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package metadata

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// immutableHeaderPrefix 不可修改的 header 前缀
var immutableHeaderPrefix = map[string]struct{}{
	"grpc-": {},
}

// AnnotateIncomingContext 将 HTTP header 转为 gRPC metadata 并注入 context
func AnnotateIncomingContext(ctx context.Context, req *http.Request, propagateHeader []string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	for _, h := range propagateHeader {
		if v := req.Header.Get(h); v != "" {
			md.Append(h, v)
		}
	}

	return metadata.NewIncomingContext(ctx, md)
}

// OutgoingHeaderMatcher 返回匹配出站 header 的函数
// 如果 immutable 为 true，则跳过 grpc- 前缀的 header
type OutgoingHeaderMatcher func(key string) (string, bool)

// NewOutgoingHeaderMatcher 创建出站 header 匹配器
func NewOutgoingHeaderMatcher(immutable bool) OutgoingHeaderMatcher {
	return func(key string) (string, bool) {
		if immutable {
			for prefix := range immutableHeaderPrefix {
				if strings.HasPrefix(key, prefix) {
					return "", false
				}
			}
		}
		return key, true
	}
}
