/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:26:27
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:05:19
 * @FilePath: \apex\grpc-runtime\validation.go
 * @Description: 请求验证器
 *
 */
package runtime

import (
	"context"
	"net/http"

	"github.com/kamalyes/grpc-runtime/validation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// RequestValidator validates a fully-bound request message
// 兼容旧代码，新代码应使用 validation.Validator
type RequestValidator = validation.Validator

// ValidationErrorFormatter formats request validation errors
// 兼容旧代码，新代码应使用 validation.ErrorFormatter
type ValidationErrorFormatter = validation.ErrorFormatter

// ValidationSkipper decides whether request validation should be skipped
type ValidationSkipper func(context.Context, *http.Request, proto.Message) bool

// WithRequestValidator configures request validation after body, path, and query binding.
func WithRequestValidator(validator RequestValidator) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.requestValidator = validator
	}
}

// WithValidationErrorFormatter configures validation error formatting.
func WithValidationErrorFormatter(formatter ValidationErrorFormatter) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.validationErrorFormatter = formatter
	}
}

// WithValidationSkipper configures a predicate that can skip validation.
func WithValidationSkipper(skipper ValidationSkipper) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.validationSkipper = skipper
	}
}

// ValidateRequest validates msg using the validator configured on mux.
func ValidateRequest(ctx context.Context, mux *ServeMux, r *http.Request, msg proto.Message) error {
	if mux == nil || mux.requestValidator == nil || msg == nil {
		return nil
	}
	if mux.validationSkipper != nil && mux.validationSkipper(ctx, r, msg) {
		return nil
	}
	if err := mux.requestValidator.Struct(msg); err != nil {
		message := err.Error()
		if mux.validationErrorFormatter != nil {
			message = mux.validationErrorFormatter(err)
		}
		return status.Error(codes.InvalidArgument, message)
	}
	return nil
}
