/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:16:07
 * @FilePath: \grpc-runtime\metadata\annotate.go
 * @Description: 上下文注解，将 gRPC metadata 注入 context
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package metadata

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type mdKey struct{}
type headerKey struct{}

// AnnotateIncoming 将 gRPC metadata 对注入 context
// 如果 ctx 已有入站 metadata 则合并，否则新建
func AnnotateIncoming(ctx context.Context, md metadata.MD) context.Context {
	if existing, ok := metadata.FromIncomingContext(ctx); ok {
		md = metadata.Join(existing, md)
	}
	return metadata.NewIncomingContext(ctx, md)
}

// WithServerMetadata 将 ServerMetadata 存入 context
func WithServerMetadata(ctx context.Context, md ServerMetadata) context.Context {
	return context.WithValue(ctx, mdKey{}, md)
}

// FromServerMetadata 从 context 取出 ServerMetadata
func FromServerMetadata(ctx context.Context) (ServerMetadata, bool) {
	md, ok := ctx.Value(mdKey{}).(ServerMetadata)
	return md, ok
}

// WithHeader 将 HTTP header 存入 context
func WithHeader(ctx context.Context, header map[string][]string) context.Context {
	return context.WithValue(ctx, headerKey{}, header)
}

// FromHeader 从 context 取出 HTTP header
func FromHeader(ctx context.Context) (map[string][]string, bool) {
	h, ok := ctx.Value(headerKey{}).(map[string][]string)
	return h, ok
}
