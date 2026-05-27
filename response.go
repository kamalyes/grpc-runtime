/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\response.go
 * @Description: 响应处理兼容层，委托给 response 子包
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/kamalyes/grpc-runtime/response"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// --- 类型别名 ---

type HTTPStatus = response.HTTPStatus

// --- mux 依赖的错误处理函数类型 ---

type ErrorHandlerFunc func(context.Context, *ServeMux, Marshaler, http.ResponseWriter, *http.Request, error)
type StreamErrorHandlerFunc func(context.Context, error) *status.Status
type RoutingErrorHandlerFunc func(context.Context, *ServeMux, Marshaler, http.ResponseWriter, *http.Request, int)

// HTTPStatusError 用于传递特定 HTTP 状态码
type HTTPStatusError struct {
	HTTPStatus int
	Err        error
}

func (e *HTTPStatusError) Error() string { return e.Err.Error() }

// --- 委托给 response 子包 ---

func HTTPStatusFromCode(code codes.Code) int { return response.HTTPStatusFromCode(code) }

func CodeFromHTTPStatus(httpStatus int) codes.Code { return response.CodeFromHTTPStatus(httpStatus) }

func IsHTTPSuccess(httpStatus int) bool        { return response.IsHTTPSuccess(httpStatus) }
func IsHTTPClientError(httpStatus int) bool     { return response.IsHTTPClientError(httpStatus) }
func IsHTTPServerError(httpStatus int) bool     { return response.IsHTTPServerError(httpStatus) }
func IsHTTPRedirect(httpStatus int) bool        { return response.IsHTTPRedirect(httpStatus) }

// HTTPError 使用 mux 配置的错误处理器
func HTTPError(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	mux.errorHandler(ctx, mux, marshaler, w, r, err)
}

// HTTPStreamError 使用 mux 配置的流错误处理器
func HTTPStreamError(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	st := mux.streamErrorHandler(ctx, err)
	msg := errorChunk(st)
	buf, err := marshaler.Marshal(msg)
	if err != nil {
		logErrorf("Failed to marshal an error: %v", err)
		return
	}
	if _, err := w.Write(buf); err != nil {
		logErrorf("Failed to notify error to client: %v", err)
		return
	}
}

// DefaultHTTPErrorHandler 默认错误处理器
func DefaultHTTPErrorHandler(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	const fallback = `{"code": 13, "message": "failed to marshal error message"}`
	const fallbackRewriter = `{"code": 13, "message": "failed to rewrite error message"}`

	var customStatus *HTTPStatusError
	if errors.As(err, &customStatus) {
		err = customStatus.Err
	}

	s := status.Convert(err)

	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")

	respRw, err := mux.forwardResponseRewriter(ctx, s.Proto())
	if err != nil {
		logErrorf("Failed to rewrite error message %q: %v", s, err)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallbackRewriter); err != nil {
			logErrorf("Failed to write response: %v", err)
		}
		return
	}

	contentType := marshaler.ContentType(respRw)
	w.Header().Set("Content-Type", contentType)

	if s.Code() == codes.Unauthenticated {
		w.Header().Set("WWW-Authenticate", s.Message())
	}

	buf, merr := marshaler.Marshal(respRw)
	if merr != nil {
		logErrorf("Failed to marshal error message %q: %v", s, merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			logErrorf("Failed to write response: %v", err)
		}
		return
	}

	md, ok := ServerMetadataFromContext(ctx)
	if ok {
		handleForwardResponseServerMetadata(w, mux, md)
		doForwardTrailers := requestAcceptsTrailers(r)
		if doForwardTrailers {
			handleForwardResponseTrailerHeader(w, mux, md)
			w.Header().Set("Transfer-Encoding", "chunked")
		}
	}

	st := HTTPStatusFromCode(s.Code())
	if customStatus != nil {
		st = customStatus.HTTPStatus
	}

	w.WriteHeader(st)
	if _, err := w.Write(buf); err != nil {
		logErrorf("Failed to write response: %v", err)
	}

	if ok && requestAcceptsTrailers(r) {
		handleForwardResponseTrailer(w, mux, md)
	}
}

func DefaultStreamErrorHandler(_ context.Context, err error) *status.Status {
	return status.Convert(err)
}

func DefaultRoutingErrorHandler(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, r *http.Request, httpStatus int) {
	sterr := status.Error(codes.Internal, "Unexpected routing error")
	switch httpStatus {
	case http.StatusBadRequest:
		sterr = status.Error(codes.InvalidArgument, http.StatusText(httpStatus))
	case http.StatusMethodNotAllowed:
		sterr = status.Error(codes.Unimplemented, http.StatusText(httpStatus))
	case http.StatusNotFound:
		sterr = status.Error(codes.NotFound, http.StatusText(httpStatus))
	}
	mux.errorHandler(ctx, mux, marshaler, w, r, sterr)
}

// --- 响应转发 ---

type responseBody interface {
	XXX_ResponseBody() interface{}
}

// ForwardResponseMessage 转发 gRPC 响应消息
func ForwardResponseMessage(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, req *http.Request, resp proto.Message, opts ...func(context.Context, http.ResponseWriter, proto.Message) error) {
	md, ok := ServerMetadataFromContext(ctx)
	if ok {
		handleForwardResponseServerMetadata(w, mux, md)
	}

	doForwardTrailers := requestAcceptsTrailers(req)
	if ok && doForwardTrailers {
		handleForwardResponseTrailerHeader(w, mux, md)
		w.Header().Set("Transfer-Encoding", "chunked")
	}

	contentType := marshaler.ContentType(resp)
	w.Header().Set("Content-Type", contentType)

	if err := handleForwardResponseOptions(ctx, w, resp, opts); err != nil {
		HTTPError(ctx, mux, marshaler, w, req, err)
		return
	}
	respRw, err := mux.forwardResponseRewriter(ctx, resp)
	if err != nil {
		logErrorf("Rewrite error: %v", err)
		HTTPError(ctx, mux, marshaler, w, req, err)
		return
	}
	var buf []byte
	if rb, ok := respRw.(responseBody); ok {
		buf, err = marshaler.Marshal(rb.XXX_ResponseBody())
	} else {
		buf, err = marshaler.Marshal(respRw)
	}
	if err != nil {
		logErrorf("Marshal error: %v", err)
		HTTPError(ctx, mux, marshaler, w, req, err)
		return
	}

	if !doForwardTrailers && mux.writeContentLength {
		w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	}

	if _, err = w.Write(buf); err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		logErrorf("Failed to write response: %v", err)
	}

	if ok && doForwardTrailers {
		handleForwardResponseTrailer(w, mux, md)
	}
}

// ForwardResponseStream 流式转发 gRPC 响应
func ForwardResponseStream(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, req *http.Request, recv func() (proto.Message, error), opts ...func(context.Context, http.ResponseWriter, proto.Message) error) {
	rc := http.NewResponseController(w)
	md, ok := ServerMetadataFromContext(ctx)
	if !ok {
		logErrorf("Failed to extract ServerMetadata from context")
		http.Error(w, "unexpected error", http.StatusInternalServerError)
		return
	}
	handleForwardResponseServerMetadata(w, mux, md)

	w.Header().Set("Transfer-Encoding", "chunked")
	if err := handleForwardResponseOptions(ctx, w, nil, opts); err != nil {
		HTTPError(ctx, mux, marshaler, w, req, err)
		return
	}

	var delimiter []byte
	if d, ok := marshaler.(Delimited); ok {
		delimiter = d.Delimiter()
	} else {
		delimiter = []byte("\n")
	}

	var wroteHeader bool
	for {
		resp, err := recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			handleForwardResponseStreamError(ctx, wroteHeader, marshaler, w, req, mux, err, delimiter)
			return
		}
		if err := handleForwardResponseOptions(ctx, w, resp, opts); err != nil {
			handleForwardResponseStreamError(ctx, wroteHeader, marshaler, w, req, mux, err, delimiter)
			return
		}

		respRw, err := mux.forwardResponseRewriter(ctx, resp)
		if err != nil {
			logErrorf("Rewrite error: %v", err)
			handleForwardResponseStreamError(ctx, wroteHeader, marshaler, w, req, mux, err, delimiter)
			return
		}

		if !wroteHeader {
			var contentType string
			if sct, ok := marshaler.(StreamContentType); ok {
				contentType = sct.StreamContentType(respRw)
			} else {
				contentType = marshaler.ContentType(respRw)
			}
			w.Header().Set("Content-Type", contentType)
		}

		var buf []byte
		httpBody, isHTTPBody := respRw.(*httpbody.HttpBody)
		switch {
		case respRw == nil:
			buf, err = marshaler.Marshal(errorChunk(status.New(codes.Internal, "empty response")))
		case isHTTPBody:
			buf = httpBody.GetData()
		default:
			result := map[string]interface{}{"result": respRw}
			if rb, ok := respRw.(responseBody); ok {
				result["result"] = rb.XXX_ResponseBody()
			}
			buf, err = marshaler.Marshal(result)
		}

		if err != nil {
			logErrorf("Failed to marshal response chunk: %v", err)
			handleForwardResponseStreamError(ctx, wroteHeader, marshaler, w, req, mux, err, delimiter)
			return
		}
		if _, err := w.Write(buf); err != nil {
			logErrorf("Failed to send response chunk: %v", err)
			return
		}
		wroteHeader = true
		if _, err := w.Write(delimiter); err != nil {
			logErrorf("Failed to send delimiter chunk: %v", err)
			return
		}
		err = rc.Flush()
		if err != nil {
			if errors.Is(err, http.ErrNotSupported) {
				logErrorf("Flush not supported in %T", w)
				http.Error(w, "unexpected type of web server", http.StatusInternalServerError)
				return
			}
			logErrorf("Failed to flush response to client: %v", err)
			return
		}
	}
}

// --- 内部辅助函数 ---

func handleForwardResponseServerMetadata(w http.ResponseWriter, mux *ServeMux, md ServerMetadata) {
	for k, vs := range md.HeaderMD {
		if h, ok := mux.outgoingHeaderMatcher(k); ok {
			for _, v := range vs {
				w.Header().Add(h, v)
			}
		}
	}
}

func handleForwardResponseTrailerHeader(w http.ResponseWriter, mux *ServeMux, md ServerMetadata) {
	for k := range md.TrailerMD {
		if h, ok := mux.outgoingTrailerMatcher(k); ok {
			w.Header().Add("Trailer", textproto.CanonicalMIMEHeaderKey(h))
		}
	}
}

func handleForwardResponseTrailer(w http.ResponseWriter, mux *ServeMux, md ServerMetadata) {
	for k, vs := range md.TrailerMD {
		if h, ok := mux.outgoingTrailerMatcher(k); ok {
			for _, v := range vs {
				w.Header().Add(h, v)
			}
		}
	}
}

func requestAcceptsTrailers(req *http.Request) bool {
	te := req.Header.Get("TE")
	return strings.Contains(strings.ToLower(te), "trailers")
}

func handleForwardResponseOptions(ctx context.Context, w http.ResponseWriter, resp proto.Message, opts []func(context.Context, http.ResponseWriter, proto.Message) error) error {
	if len(opts) == 0 {
		return nil
	}
	for _, opt := range opts {
		if err := opt(ctx, w, resp); err != nil {
			return fmt.Errorf("error handling ForwardResponseOptions: %w", err)
		}
	}
	return nil
}

func handleForwardResponseStreamError(ctx context.Context, wroteHeader bool, marshaler Marshaler, w http.ResponseWriter, req *http.Request, mux *ServeMux, err error, delimiter []byte) {
	st := mux.streamErrorHandler(ctx, err)
	msg := errorChunk(st)
	if !wroteHeader {
		w.Header().Set("Content-Type", marshaler.ContentType(msg))
		w.WriteHeader(HTTPStatusFromCode(st.Code()))
	}
	buf, err := marshaler.Marshal(msg)
	if err != nil {
		logErrorf("Failed to marshal an error: %v", err)
		return
	}
	if _, err := w.Write(buf); err != nil {
		logErrorf("Failed to notify error to client: %v", err)
		return
	}
	if _, err := w.Write(delimiter); err != nil {
		logErrorf("Failed to send delimiter chunk: %v", err)
		return
	}
}

func errorChunk(st *status.Status) map[string]proto.Message {
	return map[string]proto.Message{"error": st.Proto()}
}
