/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:17:22
 * @FilePath: \grpc-runtime\metadata\stream.go
 * @Description: ServerMetadata 和 ServerTransportStream 定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package metadata

import (
	"context"

	grpcmeta "google.golang.org/grpc/metadata"
)

// ServerMetadata 存储服务端处理过程中的 metadata
type ServerMetadata struct {
	HeaderMD  grpcmeta.MD
	TrailerMD grpcmeta.MD
}

// ServerTransportStream 实现 grpc.TransportStream 接口的最小子集
// 用于在 HTTP handler 中访问 gRPC 的 SendHeader/SetTrailer
type ServerTransportStream struct {
	header  grpcmeta.MD
	trailer grpcmeta.MD
}

// NewServerTransportStream 创建空的 ServerTransportStream
func NewServerTransportStream() *ServerTransportStream {
	return &ServerTransportStream{
		header:  grpcmeta.MD{},
		trailer: grpcmeta.MD{},
	}
}

// Method 返回空字符串（HTTP handler 不需要 gRPC 方法名）
func (*ServerTransportStream) Method() string { return "" }

// SetHeader 设置 header metadata
func (s *ServerTransportStream) SetHeader(md grpcmeta.MD) error {
	s.header = grpcmeta.Join(s.header, md)
	return nil
}

// SendHeader 发送 header（在此实现中仅保存）
func (s *ServerTransportStream) SendHeader(md grpcmeta.MD) error {
	s.header = grpcmeta.Join(s.header, md)
	return nil
}

// SetTrailer 设置 trailer metadata
func (s *ServerTransportStream) SetTrailer(md grpcmeta.MD) error {
	s.trailer = grpcmeta.Join(s.trailer, md)
	return nil
}

// Header 返回已设置的 header
func (s *ServerTransportStream) Header() grpcmeta.MD { return s.header }

// Trailer 返回已设置的 trailer
func (s *ServerTransportStream) Trailer() grpcmeta.MD { return s.trailer }

// ServerTransportStreamFromContext 从 context 中提取 ServerTransportStream
func ServerTransportStreamFromContext(ctx context.Context) *ServerTransportStream {
	if stream, ok := ctx.Value(streamKey{}).(*ServerTransportStream); ok {
		return stream
	}
	return nil
}

type streamKey struct{}

// ContextWithServerTransportStream 将 ServerTransportStream 注入 context
func ContextWithServerTransportStream(ctx context.Context, stream *ServerTransportStream) context.Context {
	return context.WithValue(ctx, streamKey{}, stream)
}
