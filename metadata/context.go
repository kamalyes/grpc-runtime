/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-27 00:09:44
 * @FilePath: \grpc-runtime\metadata\context.go
 * @Description: 上下文注解核心逻辑，将 HTTP 请求转为 gRPC context
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package metadata

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MetadataHeaderPrefix HTTP 自定义 metadata 前缀
const MetadataHeaderPrefix = "Grpc-Metadata-"

// MetadataPrefix IANA 永久 HTTP header 在 gRPC context 中的前缀
const MetadataPrefix = "grpcgateway-"

// MetadataTrailerPrefix gRPC trailer 在 HTTP 响应中的前缀
const MetadataTrailerPrefix = "Grpc-Trailer-"

// MetadataGrpcTimeout gRPC 超时 header
const MetadataGrpcTimeout = "Grpc-Timeout"

// MetadataHeaderBinarySuffix 二进制 metadata 后缀
const MetadataHeaderBinarySuffix = "-Bin"

const xForwardedFor = "X-Forwarded-For"
const xForwardedHost = "X-Forwarded-Host"

// MalformedHTTPHeaders 可能被 gRPC 服务端拒绝的 header 列表
var MalformedHTTPHeaders = map[string]struct{}{
	"connection": {},
}

// HeaderMatcherFunc 检查 header key 是否应转发到/从 gRPC context
type HeaderMatcherFunc func(string) (string, bool)

// AnnotateContext 将 HTTP 请求信息注入 gRPC context
// 包括 RemoteAddr、forwarded header、自定义 metadata、超时等
func AnnotateContext(ctx context.Context, req *http.Request, incomingHeaderMatcher HeaderMatcherFunc, metadataAnnotators []func(context.Context, *http.Request) metadata.MD, defaultTimeout int) (context.Context, metadata.MD, error) {
	timeout := time.Duration(defaultTimeout) * time.Second
	if tm := req.Header.Get(MetadataGrpcTimeout); tm != "" {
		var err error
		timeout, err = DecodeTimeout(tm)
		if err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument, "invalid grpc-timeout: %s", tm)
		}
	}

	var pairs []string
	for key, vals := range req.Header {
		key = textproto.CanonicalMIMEHeaderKey(key)
		switch key {
		case xForwardedFor, xForwardedHost:
			continue
		}

		for _, val := range vals {
			if key == "Authorization" {
				pairs = append(pairs, "authorization", val)
			}
			if h, ok := incomingHeaderMatcher(key); ok {
				if !IsValidGRPCMetadataKey(h) {
					continue
				}
				if strings.HasSuffix(key, MetadataHeaderBinarySuffix) {
					b, err := decodeBinHeader(val)
					if err != nil {
						return nil, nil, status.Errorf(codes.InvalidArgument, "invalid binary header %s: %s", key, err)
					}
					val = string(b)
				} else if !IsValidGRPCMetadataTextValue(val) {
					continue
				}
				pairs = append(pairs, h, val)
			}
		}
	}

	if host := req.Header.Get(xForwardedHost); host != "" {
		pairs = append(pairs, strings.ToLower(xForwardedHost), host)
	} else if req.Host != "" {
		pairs = append(pairs, strings.ToLower(xForwardedHost), req.Host)
	}

	xff := req.Header.Values(xForwardedFor)
	if addr := req.RemoteAddr; addr != "" {
		if remoteIP, _, err := net.SplitHostPort(addr); err == nil {
			xff = append(xff, remoteIP)
		}
	}
	if len(xff) > 0 {
		pairs = append(pairs, strings.ToLower(xForwardedFor), strings.Join(xff, ", "))
	}

	if timeout != 0 {
		ctx, _ = context.WithTimeout(ctx, timeout)
	}

	if len(pairs) == 0 {
		return ctx, nil, nil
	}

	md := metadata.Pairs(pairs...)
	for _, mda := range metadataAnnotators {
		md = metadata.Join(md, mda(ctx, req))
	}
	return ctx, md, nil
}

// DefaultHeaderMatcher 默认入站 header 匹配器
// IANA 永久 header 加 grpcgateway- 前缀，Grpc-Metadata- 前缀的 header 去掉前缀
func DefaultHeaderMatcher(key string) (string, bool) {
	switch key = textproto.CanonicalMIMEHeaderKey(key); {
	case IsPermanentHTTPHeader(key):
		return MetadataPrefix + key, true
	case strings.HasPrefix(key, MetadataHeaderPrefix):
		return key[len(MetadataHeaderPrefix):], true
	}
	return "", false
}

// DefaultOutgoingHeaderMatcher 默认出站 header 匹配器
func DefaultOutgoingHeaderMatcher(key string) (string, bool) {
	return fmt.Sprintf("%s%s", MetadataHeaderPrefix, key), true
}

// DefaultOutgoingTrailerMatcher 默认出站 trailer 匹配器
func DefaultOutgoingTrailerMatcher(key string) (string, bool) {
	return fmt.Sprintf("%s%s", MetadataTrailerPrefix, key), true
}

// decodeBinHeader 解码二进制 header
func decodeBinHeader(v string) ([]byte, error) {
	if len(v)%4 == 0 {
		return base64.StdEncoding.DecodeString(v)
	}
	return base64.RawStdEncoding.DecodeString(v)
}

// IsValidGRPCMetadataKey 检查 key 是否为合法 gRPC metadata key
func IsValidGRPCMetadataKey(key string) bool {
	for _, ch := range []byte(key) {
		validLower := ch >= 'a' && ch <= 'z'
		validUpper := ch >= 'A' && ch <= 'Z'
		validDigit := ch >= '0' && ch <= '9'
		validOther := ch == '.' || ch == '-' || ch == '_'
		if !validLower && !validUpper && !validDigit && !validOther {
			return false
		}
	}
	return true
}

// IsValidGRPCMetadataTextValue 检查 value 是否为合法 gRPC metadata text value
func IsValidGRPCMetadataTextValue(textValue string) bool {
	for _, ch := range []byte(textValue) {
		if ch < 0x20 || ch > 0x7E {
			return false
		}
	}
	return true
}

// IsPermanentHTTPHeader 检查 header 是否属于 IANA 永久 header 列表
func IsPermanentHTTPHeader(hdr string) bool {
	switch hdr {
	case "Accept", "Accept-Charset", "Accept-Language", "Accept-Ranges",
		"Authorization", "Cache-Control", "Content-Type", "Cookie",
		"Date", "Expect", "From", "Host", "If-Match", "If-Modified-Since",
		"If-None-Match", "If-Schedule-Tag-Match", "If-Unmodified-Since",
		"Max-Forwards", "Origin", "Pragma", "Referer", "User-Agent",
		"Via", "Warning":
		return true
	}
	return false
}

// IsMalformedHTTPHeader 检查 header 是否属于可能被 gRPC 服务端拒绝的 header
func IsMalformedHTTPHeader(header string) bool {
	_, isMalformed := MalformedHTTPHeaders[strings.ToLower(header)]
	return isMalformed
}
