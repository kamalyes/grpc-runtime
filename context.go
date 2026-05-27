/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 10:15:23
 * @FilePath: \grpc-runtime\context.go
 * @Description: 上下文注解兼容层，委托给 metadata 子包
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"context"
	"net/http"
	"time"

	runtimemeta "github.com/kamalyes/grpc-runtime/metadata"
	grpcmeta "google.golang.org/grpc/metadata"
)

// --- 常量，委托给 metadata 子包 ---

const MetadataHeaderPrefix = runtimemeta.MetadataHeaderPrefix
const MetadataPrefix = runtimemeta.MetadataPrefix
const MetadataTrailerPrefix = runtimemeta.MetadataTrailerPrefix

// DefaultContextTimeout gRPC 调用默认超时，0 表示无超时
var DefaultContextTimeout = 0 * time.Second

// --- context key 类型 ---

type (
	rpcMethodKey       struct{}
	httpPathPatternKey struct{}
	httpPatternKey     struct{}

	AnnotateContextOption func(ctx context.Context) context.Context
)

// WithHTTPPathPattern 返回设置 HTTP path pattern 的 AnnotateContextOption
func WithHTTPPathPattern(pattern string) AnnotateContextOption {
	return func(ctx context.Context) context.Context {
		return withHTTPPathPattern(ctx, pattern)
	}
}

// --- ServerMetadata，委托给 metadata 子包 ---

type ServerMetadata = runtimemeta.ServerMetadata

// NewServerMetadataContext 创建包含 ServerMetadata 的 context
func NewServerMetadataContext(ctx context.Context, md ServerMetadata) context.Context {
	return runtimemeta.WithServerMetadata(ctx, md)
}

// ServerMetadataFromContext 从 context 取出 ServerMetadata
func ServerMetadataFromContext(ctx context.Context) (md ServerMetadata, ok bool) {
	return runtimemeta.FromServerMetadata(ctx)
}

// --- ServerTransportStream，委托给 metadata 子包 ---

type ServerTransportStream = runtimemeta.ServerTransportStream

// --- AnnotateContext，委托给 metadata 子包 ---

func AnnotateContext(ctx context.Context, mux *ServeMux, req *http.Request, rpcMethodName string, options ...AnnotateContextOption) (context.Context, error) {
	ctx, md, err := annotateContext(ctx, mux, req, rpcMethodName, options...)
	if err != nil {
		return nil, err
	}
	if md == nil {
		return ctx, nil
	}
	return grpcmeta.NewOutgoingContext(ctx, md), nil
}

func AnnotateIncomingContext(ctx context.Context, mux *ServeMux, req *http.Request, rpcMethodName string, options ...AnnotateContextOption) (context.Context, error) {
	ctx, md, err := annotateContext(ctx, mux, req, rpcMethodName, options...)
	if err != nil {
		return nil, err
	}
	if md == nil {
		return ctx, nil
	}
	return grpcmeta.NewIncomingContext(ctx, md), nil
}

func annotateContext(ctx context.Context, mux *ServeMux, req *http.Request, rpcMethodName string, options ...AnnotateContextOption) (context.Context, grpcmeta.MD, error) {
	ctx = withRPCMethod(ctx, rpcMethodName)
	for _, o := range options {
		ctx = o(ctx)
	}

	incomingMatcher := func(key string) (string, bool) {
		return mux.incomingHeaderMatcher(key)
	}

	var annotators []func(context.Context, *http.Request) grpcmeta.MD
	if len(mux.metadataAnnotators) > 0 {
		annotators = mux.metadataAnnotators
	}

	return runtimemeta.AnnotateContext(ctx, req, incomingMatcher, annotators, int(DefaultContextTimeout/time.Second))
}

// --- context 辅助函数 ---

func withRPCMethod(ctx context.Context, rpcMethodName string) context.Context {
	return context.WithValue(ctx, rpcMethodKey{}, rpcMethodName)
}

func withHTTPPathPattern(ctx context.Context, httpPathPattern string) context.Context {
	return context.WithValue(ctx, httpPathPatternKey{}, httpPathPattern)
}

func withHTTPPattern(ctx context.Context, httpPattern Pattern) context.Context {
	return context.WithValue(ctx, httpPatternKey{}, httpPattern)
}

// RPCMethod 返回 context 中的 RPC 方法名
func RPCMethod(ctx context.Context) (string, bool) {
	m := ctx.Value(rpcMethodKey{})
	if m == nil {
		return "", false
	}
	ms, ok := m.(string)
	if !ok {
		return "", false
	}
	return ms, true
}

// HTTPPathPattern 返回 context 中的 HTTP path pattern
func HTTPPathPattern(ctx context.Context) (string, bool) {
	m := ctx.Value(httpPathPatternKey{})
	if m == nil {
		return "", false
	}
	ms, ok := m.(string)
	if !ok {
		return "", false
	}
	return ms, true
}

// HTTPPattern 返回 context 中的 HTTP Pattern
func HTTPPattern(ctx context.Context) (Pattern, bool) {
	v, ok := ctx.Value(httpPatternKey{}).(Pattern)
	return v, ok
}

// --- 常量/函数兼容，委托给 metadata 子包 ---

func isPermanentHTTPHeader(hdr string) bool    { return runtimemeta.IsPermanentHTTPHeader(hdr) }
func isMalformedHTTPHeader(header string) bool { return runtimemeta.IsMalformedHTTPHeader(header) }
