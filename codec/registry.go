/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec\registry.go
 * @Description: Marshaler 注册表，MIME 类型到 Marshaler 的并发安全映射
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package codec

import (
	"errors"
	"mime"
	"net/http"
	"sync"
)

// MIMEWildcard 通配 MIME 类型，匹配任何未注册的 Content-Type
const MIMEWildcard = "*"

// defaultMarshaler 默认 marshaler 实例
var defaultMarshaler = &HTTPBodyMarshaler{
	Marshaler: &JSONPb{
		MarshalOptions:   defaultMarshalOptions,
		UnmarshalOptions:  defaultUnmarshalOptions,
	},
}

// Registry MIME 类型到 Marshaler 的并发安全注册表
type Registry struct {
	mu      sync.RWMutex
	mimeMap map[string]Marshaler
}

// NewRegistry 创建包含默认通配 marshaler 的注册表
func NewRegistry() *Registry {
	return &Registry{
		mimeMap: map[string]Marshaler{
			MIMEWildcard: defaultMarshaler,
		},
	}
}

// Add 注册 MIME 类型到 Marshaler 的映射
func (r *Registry) Add(mime string, m Marshaler) error {
	if len(mime) == 0 {
		return errors.New("empty MIME type")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mimeMap[mime] = m
	return nil
}

// Lookup 按 MIME 类型查找 Marshaler
func (r *Registry) Lookup(mime string) (Marshaler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.mimeMap[mime]
	return m, ok
}

// MarshalerForRequest 根据 HTTP 请求的 Accept 和 Content-Type 头查找 inbound/outbound marshaler
func MarshalerForRequest(reg *Registry, r *http.Request) (inbound Marshaler, outbound Marshaler) {
	acceptHeader := http.CanonicalHeaderKey("Accept")
	contentTypeHeader := http.CanonicalHeaderKey("Content-Type")

	for _, acceptVal := range r.Header[acceptHeader] {
		if m, ok := reg.Lookup(acceptVal); ok {
			outbound = m
			break
		}
	}

	for _, contentTypeVal := range r.Header[contentTypeHeader] {
		contentType, _, err := mime.ParseMediaType(contentTypeVal)
		if err != nil {
			continue
		}
		if m, ok := reg.Lookup(contentType); ok {
			inbound = m
			break
		}
	}

	if inbound == nil {
		inbound, _ = reg.Lookup(MIMEWildcard)
	}
	if outbound == nil {
		outbound = inbound
	}

	return inbound, outbound
}
