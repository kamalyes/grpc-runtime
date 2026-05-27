/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:37:53
 * @FilePath: \grpc-runtime\response\status.go
 * @Description: gRPC 状态码与 HTTP 状态码映射
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package response

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// httpStatusFromCode 使用数组替代 switch，O(1) 查找
// 索引为 gRPC code 值，值为 HTTP 状态码
var httpStatusFromCode = [17]int{
	http.StatusInternalServerError, // 0  OK (不应出现)
	http.StatusBadRequest,          // 1  Canceled
	http.StatusInternalServerError, // 2  Unknown
	http.StatusBadRequest,          // 3  InvalidArgument
	http.StatusGatewayTimeout,      // 4  DeadlineExceeded
	http.StatusNotFound,            // 5  NotFound
	http.StatusConflict,            // 6  AlreadyExists
	http.StatusForbidden,           // 7  PermissionDenied
	http.StatusInsufficientStorage, // 8  ResourceExhausted
	http.StatusBadRequest,          // 9  FailedPrecondition
	http.StatusConflict,            // 10 Aborted
	http.StatusBadRequest,          // 11 OutOfRange
	http.StatusNotImplemented,      // 12 Unimplemented
	http.StatusInternalServerError, // 13 Internal
	http.StatusServiceUnavailable,  // 14 Unavailable
	http.StatusInternalServerError, // 15 DataLoss
	http.StatusUnauthorized,        // 16 Unauthenticated
}

// HTTPStatusFromCode 将 gRPC code 映射为 HTTP 状态码
func HTTPStatusFromCode(code codes.Code) int {
	if int(code) < len(httpStatusFromCode) {
		return httpStatusFromCode[code]
	}
	return http.StatusInternalServerError
}

// codeFromHTTPStatus 将常见 HTTP 状态码反向映射为 gRPC code
// 优先匹配最常用的状态码，未匹配时返回 codes.Unknown
var codeFromHTTPStatus = map[int]codes.Code{
	// 2xx
	http.StatusOK: codes.OK,
	// 4xx
	http.StatusBadRequest:            codes.InvalidArgument,
	http.StatusUnauthorized:          codes.Unauthenticated,
	http.StatusForbidden:             codes.PermissionDenied,
	http.StatusNotFound:              codes.NotFound,
	http.StatusMethodNotAllowed:      codes.Unimplemented,
	http.StatusRequestTimeout:        codes.Canceled,
	http.StatusConflict:              codes.AlreadyExists,
	http.StatusPreconditionFailed:    codes.FailedPrecondition,
	http.StatusRequestEntityTooLarge: codes.InvalidArgument,
	http.StatusTooManyRequests:       codes.ResourceExhausted,
	// 5xx
	http.StatusInternalServerError: codes.Internal,
	http.StatusNotImplemented:      codes.Unimplemented,
	http.StatusBadGateway:          codes.Unavailable,
	http.StatusServiceUnavailable:  codes.Unavailable,
	http.StatusGatewayTimeout:      codes.DeadlineExceeded,
	http.StatusInsufficientStorage: codes.ResourceExhausted,
}

// CodeFromHTTPStatus 将 HTTP 状态码反向映射为 gRPC code
func CodeFromHTTPStatus(httpStatus int) codes.Code {
	if code, ok := codeFromHTTPStatus[httpStatus]; ok {
		return code
	}
	if httpStatus >= 200 && httpStatus < 300 {
		return codes.OK
	}
	return codes.Unknown
}

// IsHTTPSuccess 判断 HTTP 状态码是否为成功（2xx）
func IsHTTPSuccess(httpStatus int) bool {
	return httpStatus >= 200 && httpStatus < 300
}

// IsHTTPClientError 判断 HTTP 状态码是否为客户端错误（4xx）
func IsHTTPClientError(httpStatus int) bool {
	return httpStatus >= 400 && httpStatus < 500
}

// IsHTTPServerError 判断 HTTP 状态码是否为服务端错误（5xx）
func IsHTTPServerError(httpStatus int) bool {
	return httpStatus >= 500 && httpStatus < 600
}

// IsHTTPRedirect 判断 HTTP 状态码是否为重定向（3xx）
func IsHTTPRedirect(httpStatus int) bool {
	return httpStatus >= 300 && httpStatus < 400
}
