/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\handler_test.go
 * @Description: RouteHandler 类型适配单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteHandlerFromFunc(t *testing.T) {
	var captured map[string]string

	oldFunc := func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		captured = pathParams
	}

	handler := RouteHandlerFromFunc(oldFunc)

	params := NewParams(2)
	params.Add("user_id", "123")
	params.Add("tenant_id", "acme")

	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/", nil), params)

	assert.Equal(t, "123", captured["user_id"])
	assert.Equal(t, "acme", captured["tenant_id"])
}

func TestRouteHandlerToHandlerFunc(t *testing.T) {
	var captured *Params

	handler := RouteHandler(func(w http.ResponseWriter, r *http.Request, params *Params) {
		captured = params
	})

	oldFunc := handler.ToHandlerFunc()

	w := httptest.NewRecorder()
	oldFunc(w, httptest.NewRequest("GET", "/", nil), map[string]string{
		"user_id":   "456",
		"tenant_id": "beta",
	})

	assert.NotNil(t, captured)
	val, ok := captured.Get("user_id")
	assert.True(t, ok)
	assert.Equal(t, "456", val)

	val, ok = captured.Get("tenant_id")
	assert.True(t, ok)
	assert.Equal(t, "beta", val)
}

func TestRouteHandlerRoundTrip(t *testing.T) {
	// RouteHandler → ToHandlerFunc → RouteHandlerFromFunc → 验证参数传递
	var captured *Params

	original := RouteHandler(func(w http.ResponseWriter, r *http.Request, params *Params) {
		captured = params
	})

	oldFunc := original.ToHandlerFunc()
	adapted := RouteHandlerFromFunc(oldFunc)

	params := NewParams(1)
	params.Add("key", "value")

	w := httptest.NewRecorder()
	adapted(w, httptest.NewRequest("GET", "/", nil), params)

	assert.NotNil(t, captured)
	val, ok := captured.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}
