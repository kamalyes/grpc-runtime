/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:56:32
 * @FilePath: \grpc-runtime\metadata\context_test.go
 * @Description: metadata 上下文注解测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package metadata

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	grpcmeta "google.golang.org/grpc/metadata"
)

func TestAnnotateContext_BasicHeaders(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com/v1", nil)
	req.Header.Set("Grpc-Metadata-Foo", "bar")
	req.Header.Set("Authorization", "Token test123")

	_, md, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, md)

	// Authorization 应该被转发
	vals := md.Get("authorization")
	assert.Contains(t, vals, "Token test123")

	// Grpc-Metadata-Foo 应该被去掉前缀转发
	fooVals := md.Get("foo")
	assert.Contains(t, fooVals, "bar")
}

func TestAnnotateContext_Timeout(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Grpc-Timeout", "10S")

	annotated, _, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 0)
	assert.NoError(t, err)

	deadline, ok := annotated.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(10*time.Second), deadline, 2*time.Second)
}

func TestAnnotateContext_DefaultTimeout(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)

	annotated, _, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 5)
	assert.NoError(t, err)

	deadline, ok := annotated.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(5*time.Second), deadline, 2*time.Second)
}

func TestAnnotateContext_ZeroTimeout(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)

	annotated, _, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 0)
	assert.NoError(t, err)

	_, ok := annotated.Deadline()
	assert.False(t, ok)
}

func TestAnnotateContext_InvalidTimeout(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Grpc-Timeout", "invalid")

	_, _, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 0)
	assert.Error(t, err)
}

func TestAnnotateContext_MetadataAnnotators(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)

	annotator := func(ctx context.Context, req *http.Request) grpcmeta.MD {
		return grpcmeta.Pairs("custom-key", "custom-value")
	}

	_, md, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, []func(context.Context, *http.Request) grpcmeta.MD{annotator}, 0)
	assert.NoError(t, err)
	assert.NotNil(t, md)

	vals := md.Get("custom-key")
	assert.Contains(t, vals, "custom-value")
}

func TestAnnotateContext_ForwardedHeaders(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com/v1", nil)
	req.Header.Set("X-Forwarded-Host", "proxy.example.com")
	req.RemoteAddr = "192.168.1.1:12345"

	_, md, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, md)

	hostVals := md.Get("x-forwarded-host")
	assert.Contains(t, hostVals, "proxy.example.com")

	forVals := md.Get("x-forwarded-for")
	assert.NotEmpty(t, forVals)
}

func TestAnnotateContext_BinaryHeader(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	binaryValue := []byte{0x00, 0x01, 0x02}
	req.Header.Set("Grpc-Metadata-Data-Bin", base64.StdEncoding.EncodeToString(binaryValue))

	_, md, err := AnnotateContext(ctx, req, DefaultHeaderMatcher, nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, md)
}

func TestDefaultHeaderMatcher(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantMatch bool
	}{
		{"permanent header", "Accept", "grpcgateway-Accept", true},
		{"grpc metadata prefix", "Grpc-Metadata-Foo", "Foo", true},
		{"unknown header", "X-Custom", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, ok := DefaultHeaderMatcher(tt.input)
			assert.Equal(t, tt.wantMatch, ok)
			if ok {
				assert.Equal(t, tt.wantKey, key)
			}
		})
	}
}

func TestIsValidGRPCMetadataKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid lowercase", "foo-bar_baz.qux", true},
		{"valid uppercase", "Foo-Bar", true},
		{"invalid space", "foo bar", false},
		{"invalid unicode", "foo\u00e9", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidGRPCMetadataKey(tt.input))
		})
	}
}

func TestIsValidGRPCMetadataTextValue(t *testing.T) {
	assert.True(t, IsValidGRPCMetadataTextValue("hello world"))
	assert.True(t, IsValidGRPCMetadataTextValue("test123"))
	assert.False(t, IsValidGRPCMetadataTextValue("hello\u0000world"))
	assert.False(t, IsValidGRPCMetadataTextValue("hello\xffworld"))
}

func TestIsPermanentHTTPHeader(t *testing.T) {
	assert.True(t, IsPermanentHTTPHeader("Accept"))
	assert.True(t, IsPermanentHTTPHeader("Authorization"))
	assert.True(t, IsPermanentHTTPHeader("Content-Type"))
	assert.False(t, IsPermanentHTTPHeader("X-Custom-Header"))
	assert.False(t, IsPermanentHTTPHeader("Grpc-Metadata-Foo"))
}

func TestIsMalformedHTTPHeader(t *testing.T) {
	assert.True(t, IsMalformedHTTPHeader("connection"))
	assert.True(t, IsMalformedHTTPHeader("Connection"))
	assert.False(t, IsMalformedHTTPHeader("content-type"))
}

func TestDecodeTimeout(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"hours", "1H", time.Hour, false},
		{"minutes", "30M", 30 * time.Minute, false},
		{"seconds", "10S", 10 * time.Second, false},
		{"milliseconds", "500m", 500 * time.Millisecond, false},
		{"microseconds", "100u", 100 * time.Microsecond, false},
		{"nanoseconds", "50n", 50 * time.Nanosecond, false},
		{"too short", "H", 0, true},
		{"invalid unit", "10X", 0, true},
		{"invalid number", "abS", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeTimeout(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeTimeout(t *testing.T) {
	tests := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{"hours", time.Hour, "1H"},
		{"minutes", 30 * time.Minute, "30M"},
		{"seconds", 10 * time.Second, "10S"},
		{"milliseconds", 500 * time.Millisecond, "500m"},
		{"nanoseconds", 50, "50n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EncodeTimeout(tt.input))
		})
	}
}

func TestAnnotateIncoming(t *testing.T) {
	ctx := context.Background()
	md := grpcmeta.Pairs("key", "value")

	result := AnnotateIncoming(ctx, md)
	existing, ok := grpcmeta.FromIncomingContext(result)
	assert.True(t, ok)
	assert.Equal(t, "value", existing.Get("key")[0])
}

func TestServerMetadata(t *testing.T) {
	md := ServerMetadata{
		HeaderMD:  grpcmeta.Pairs("header-key", "header-val"),
		TrailerMD: grpcmeta.Pairs("trailer-key", "trailer-val"),
	}

	ctx := WithServerMetadata(context.Background(), md)
	got, ok := FromServerMetadata(ctx)
	assert.True(t, ok)
	assert.Equal(t, md.HeaderMD, got.HeaderMD)
	assert.Equal(t, md.TrailerMD, got.TrailerMD)
}

func TestServerTransportStream(t *testing.T) {
	s := NewServerTransportStream()
	assert.Equal(t, "", s.Method())

	err := s.SetHeader(grpcmeta.Pairs("h1", "v1"))
	assert.NoError(t, err)

	err = s.SetTrailer(grpcmeta.Pairs("t1", "v1"))
	assert.NoError(t, err)

	assert.Equal(t, []string{"v1"}, s.Header().Get("h1"))
	assert.Equal(t, []string{"v1"}, s.Trailer().Get("t1"))
}
