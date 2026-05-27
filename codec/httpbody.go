/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 10:58:07
 * @FilePath: \grpc-runtime\codec\httpbody.go
 * @Description: google.api.HttpBody 的 Marshaler 实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package codec

import "google.golang.org/genproto/googleapis/api/httpbody"

// HTTPBodyMarshaler 支持 google.api.HttpBody 的 Marshaler
// 如果响应是 HttpBody 消息则直接使用其 data 和 content_type
// 否则回退到内嵌的默认 Marshaler
type HTTPBodyMarshaler struct {
	Marshaler
}

// ContentType 优先返回 HttpBody 的 content_type，否则回退到默认
func (h *HTTPBodyMarshaler) ContentType(v interface{}) string {
	if httpBody, ok := v.(*httpbody.HttpBody); ok {
		return httpBody.GetContentType()
	}
	return h.Marshaler.ContentType(v)
}

// Marshal 优先返回 HttpBody 的 data，否则回退到默认
func (h *HTTPBodyMarshaler) Marshal(v interface{}) ([]byte, error) {
	if httpBody, ok := v.(*httpbody.HttpBody); ok {
		return httpBody.GetData(), nil
	}
	return h.Marshaler.Marshal(v)
}
