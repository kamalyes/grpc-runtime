/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:21:22
 * @FilePath: \grpc-runtime\response\status_test.go
 * @Description: gRPC 状态码与 HTTP 状态码映射测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package response

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestHTTPStatusFromCode(t *testing.T) {
	tests := []struct {
		name     string
		code     codes.Code
		wantHTTP int
	}{
		{"OK", codes.OK, http.StatusInternalServerError},
		{"Canceled", codes.Canceled, http.StatusBadRequest},
		{"Unknown", codes.Unknown, http.StatusInternalServerError},
		{"InvalidArgument", codes.InvalidArgument, http.StatusBadRequest},
		{"DeadlineExceeded", codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{"NotFound", codes.NotFound, http.StatusNotFound},
		{"AlreadyExists", codes.AlreadyExists, http.StatusConflict},
		{"PermissionDenied", codes.PermissionDenied, http.StatusForbidden},
		{"ResourceExhausted", codes.ResourceExhausted, http.StatusInsufficientStorage},
		{"FailedPrecondition", codes.FailedPrecondition, http.StatusBadRequest},
		{"Aborted", codes.Aborted, http.StatusConflict},
		{"OutOfRange", codes.OutOfRange, http.StatusBadRequest},
		{"Unimplemented", codes.Unimplemented, http.StatusNotImplemented},
		{"Internal", codes.Internal, http.StatusInternalServerError},
		{"Unavailable", codes.Unavailable, http.StatusServiceUnavailable},
		{"DataLoss", codes.DataLoss, http.StatusInternalServerError},
		{"Unauthenticated", codes.Unauthenticated, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTTPStatusFromCode(tt.code)
			assert.Equal(t, tt.wantHTTP, got)
		})
	}
}

func TestHTTPStatusFromCode_OutOfRange(t *testing.T) {
	got := HTTPStatusFromCode(codes.Code(100))
	assert.Equal(t, http.StatusInternalServerError, got)
}

func TestCodeFromHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		httpCode int
		wantCode codes.Code
	}{
		// 2xx
		{"OK", http.StatusOK, codes.OK},
		// 4xx
		{"BadRequest", http.StatusBadRequest, codes.InvalidArgument},
		{"Unauthorized", http.StatusUnauthorized, codes.Unauthenticated},
		{"Forbidden", http.StatusForbidden, codes.PermissionDenied},
		{"NotFound", http.StatusNotFound, codes.NotFound},
		{"MethodNotAllowed", http.StatusMethodNotAllowed, codes.Unimplemented},
		{"RequestTimeout", http.StatusRequestTimeout, codes.Canceled},
		{"Conflict", http.StatusConflict, codes.AlreadyExists},
		{"PreconditionFailed", http.StatusPreconditionFailed, codes.FailedPrecondition},
		{"RequestEntityTooLarge", http.StatusRequestEntityTooLarge, codes.InvalidArgument},
		{"TooManyRequests", http.StatusTooManyRequests, codes.ResourceExhausted},
		// 5xx
		{"InternalServerError", http.StatusInternalServerError, codes.Internal},
		{"NotImplemented", http.StatusNotImplemented, codes.Unimplemented},
		{"BadGateway", http.StatusBadGateway, codes.Unavailable},
		{"ServiceUnavailable", http.StatusServiceUnavailable, codes.Unavailable},
		{"GatewayTimeout", http.StatusGatewayTimeout, codes.DeadlineExceeded},
		{"InsufficientStorage", http.StatusInsufficientStorage, codes.ResourceExhausted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CodeFromHTTPStatus(tt.httpCode)
			assert.Equal(t, tt.wantCode, got)
		})
	}
}

func TestCodeFromHTTPStatus_Unknown(t *testing.T) {
	// 未注册的 HTTP 状态码
	assert.Equal(t, codes.Unknown, CodeFromHTTPStatus(199))
	assert.Equal(t, codes.Unknown, CodeFromHTTPStatus(599))
	// 未注册的 2xx 应该返回 OK
	assert.Equal(t, codes.OK, CodeFromHTTPStatus(208))
}

func TestIsHTTPSuccess(t *testing.T) {
	assert.True(t, IsHTTPSuccess(200))
	assert.True(t, IsHTTPSuccess(204))
	assert.True(t, IsHTTPSuccess(299))
	assert.False(t, IsHTTPSuccess(199))
	assert.False(t, IsHTTPSuccess(300))
}

func TestIsHTTPClientError(t *testing.T) {
	assert.True(t, IsHTTPClientError(400))
	assert.True(t, IsHTTPClientError(404))
	assert.True(t, IsHTTPClientError(499))
	assert.False(t, IsHTTPClientError(399))
	assert.False(t, IsHTTPClientError(500))
}

func TestIsHTTPServerError(t *testing.T) {
	assert.True(t, IsHTTPServerError(500))
	assert.True(t, IsHTTPServerError(503))
	assert.False(t, IsHTTPServerError(499))
	assert.False(t, IsHTTPServerError(600))
}

func TestIsHTTPRedirect(t *testing.T) {
	assert.True(t, IsHTTPRedirect(301))
	assert.True(t, IsHTTPRedirect(302))
	assert.True(t, IsHTTPRedirect(399))
	assert.False(t, IsHTTPRedirect(299))
	assert.False(t, IsHTTPRedirect(400))
}
