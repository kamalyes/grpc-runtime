/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\response\error_test.go
 * @Description: HTTP 错误处理测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHTTPError_Error(t *testing.T) {
	err := NewHTTPError(http.StatusNotFound, "resource not found")
	assert.Equal(t, "resource not found", err.Error())
	assert.Equal(t, http.StatusNotFound, err.ErrHTTP())
}

func TestHTTPStatusFromError_Nil(t *testing.T) {
	assert.Equal(t, http.StatusOK, HTTPStatusFromError(nil))
}

func TestHTTPStatusFromError_HTTPStatus(t *testing.T) {
	err := NewHTTPError(http.StatusForbidden, "forbidden")
	assert.Equal(t, http.StatusForbidden, HTTPStatusFromError(err))
}

func TestHTTPStatusFromError_GRPCStatus(t *testing.T) {
	err := status.Error(codes.NotFound, "not found")
	assert.Equal(t, http.StatusNotFound, HTTPStatusFromError(err))
}

func TestHTTPStatusFromError_UnknownError(t *testing.T) {
	// 非 gRPC status 的普通 error
	err := errors.New("plain error")
	assert.Equal(t, http.StatusInternalServerError, HTTPStatusFromError(err))
}

func TestHTTPStatusFromError_InvalidGRPCCode(t *testing.T) {
	err := status.Error(codes.Code(100), "unknown")
	assert.Equal(t, http.StatusInternalServerError, HTTPStatusFromError(err))
}

func TestFromGRPCStatus(t *testing.T) {
	s := status.New(codes.InvalidArgument, "bad request")
	httpErr := FromGRPCStatus(s)
	assert.Equal(t, http.StatusBadRequest, httpErr.HTTPStatus)
	assert.Equal(t, "bad request", httpErr.Message)
}

func TestNewHTTPError(t *testing.T) {
	err := NewHTTPError(http.StatusBadGateway, "upstream error: %s", "timeout")
	assert.Equal(t, http.StatusBadGateway, err.HTTPStatus)
	assert.Equal(t, "upstream error: timeout", err.Message)
}
