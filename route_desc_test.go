/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\route_desc_test.go
 * @Description: 路由描述 facade 测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kamalyes/grpc-runtime/testpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestBodyBinding(t *testing.T) {
	t.Run("NoBody", func(t *testing.T) {
		b := NoBody()
		assert.False(t, b.HasBody)
		assert.Empty(t, b.FieldPath)
	})

	t.Run("Body", func(t *testing.T) {
		b := Body("user")
		assert.True(t, b.HasBody)
		assert.Equal(t, "user", b.FieldPath)
	})
}

func TestNewQueryFilter(t *testing.T) {
	qf := NewQueryFilter("user_id", "name.nested")
	assert.NotNil(t, qf)
}

func TestRouteDescWithInvoker(t *testing.T) {
	mux := NewServeMux()

	route := RouteDesc{
		Method:    http.MethodGet,
		Template:  "/v1/test",
		Operation: "/test.Service/Method",
		Request:   func() proto.Message { return new(testpb.Proto3Message) },
		Body:      NoBody(),
		Invoker: func(ctx context.Context, req proto.Message, target any) (proto.Message, ServerMetadata, error) {
			return req, ServerMetadata{}, nil
		},
	}

	err := RegisterRoutes(context.Background(), mux, []RouteDesc{route})
	assert.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	w := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		mux.ServeHTTP(w, r)
	})
}

func TestRouteDescWithHandler(t *testing.T) {
	mux := NewServeMux()

	var handlerCalled bool
	route := RouteDesc{
		Method:   http.MethodGet,
		Template: "/v1/legacy",
		Handler: func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			handlerCalled = true
		},
	}

	err := RegisterRoutes(context.Background(), mux, []RouteDesc{route})
	assert.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/v1/legacy", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	assert.True(t, handlerCalled)
}

func TestRouteDescMinimal(t *testing.T) {
	mux := NewServeMux()

	route := RouteDesc{
		Method:   http.MethodGet,
		Template: "/v1/minimal",
	}

	err := RegisterRoutes(context.Background(), mux, []RouteDesc{route})
	assert.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/v1/minimal", nil)
	w := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		mux.ServeHTTP(w, r)
	})
}

func TestRegisterRoutesMultiple(t *testing.T) {
	mux := NewServeMux()

	routes := []RouteDesc{
		{Method: http.MethodGet, Template: "/v1/a", Handler: func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {}},
		{Method: http.MethodGet, Template: "/v1/b", Handler: func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {}},
		{Method: http.MethodPost, Template: "/v1/c", Handler: func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {}},
	}

	err := RegisterRoutes(context.Background(), mux, routes)
	assert.NoError(t, err)

	for _, path := range []string{"/v1/a", "/v1/b"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code, "path=%s", path)
	}

	// GET /v1/c 命中旧 runtime 的 method mismatch 语义：gRPC Unimplemented -> HTTP 501。
	r := httptest.NewRequest(http.MethodGet, "/v1/c", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestBuildRequestPathParams(t *testing.T) {
	mux := NewServeMux()
	msg := new(testpb.Proto3Message)

	pathParams := map[string]string{
		"string_value": "hello",
	}

	err := BuildRequest(context.Background(), mux, httptest.NewRequest(http.MethodGet, "/", nil), msg, pathParams, NoBody(), nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello", msg.GetStringValue())
}

func TestBuildRequestQueryParams(t *testing.T) {
	mux := NewServeMux()
	msg := new(testpb.Proto3Message)

	qf := QueryFilter()
	r := httptest.NewRequest(http.MethodGet, "/?string_value=world", nil)

	err := BuildRequest(context.Background(), mux, r, msg, nil, NoBody(), qf)
	assert.NoError(t, err)
	assert.Equal(t, "world", msg.GetStringValue())
}

func TestBuildRequestNoParams(t *testing.T) {
	mux := NewServeMux()
	msg := new(testpb.Proto3Message)

	err := BuildRequest(context.Background(), mux, httptest.NewRequest(http.MethodGet, "/", nil), msg, nil, NoBody(), nil)
	assert.NoError(t, err)
}
