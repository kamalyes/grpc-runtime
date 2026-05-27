/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\response\error.go
 * @Description: gRPC 错误到 HTTP 错误的映射
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package response

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/status"
)

// HTTPError 表示一个 HTTP 错误响应
type HTTPError struct {
	HTTPStatus int    `json:"-"`
	Message    string `json:"message"`
}

// Error 实现 error 接口
func (e *HTTPError) Error() string {
	return e.Message
}

// ErrHTTP 实现 HTTPStatus 接口
func (e *HTTPError) ErrHTTP() int {
	return e.HTTPStatus
}

// HTTPStatus 接口用于从 error 中提取 HTTP 状态码
type HTTPStatus interface {
	ErrHTTP() int
}

// HTTPStatusFromError 从 error 中提取 HTTP 状态码
// 如果 error 实现了 HTTPStatus 接口则使用其值
// 如果是 gRPC status 则按 code 映射
// 否则返回 500
func HTTPStatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if hs, ok := err.(HTTPStatus); ok {
		return hs.ErrHTTP()
	}
	if s, ok := status.FromError(err); ok {
		return HTTPStatusFromCode(s.Code())
	}
	return http.StatusInternalServerError
}

// FromGRPCStatus 将 gRPC status 转为 HTTPError
func FromGRPCStatus(s *status.Status) *HTTPError {
	return &HTTPError{
		HTTPStatus: HTTPStatusFromCode(s.Code()),
		Message:    s.Message(),
	}
}

// NewHTTPError 创建 HTTPError
func NewHTTPError(httpStatus int, format string, args ...interface{}) *HTTPError {
	return &HTTPError{
		HTTPStatus: httpStatus,
		Message:    fmt.Sprintf(format, args...),
	}
}
