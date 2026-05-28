/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:59:31
 * @FilePath: \grpc-runtime\mux.go
 * @Description: 核心路由分发器，匹配 HTTP 请求到 gRPC handler
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/kamalyes/grpc-runtime/httprule"
	runtimemeta "github.com/kamalyes/grpc-runtime/metadata"
	"github.com/kamalyes/grpc-runtime/routing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// UnescapingMode 定义 ServeMux 对路径参数 unescape 的行为
type UnescapingMode int

const (
	// UnescapingModeLegacy 默认 V2 行为，在路由前对整个路径做 unescape
	UnescapingModeLegacy UnescapingMode = iota

	// UnescapingModeAllExceptReserved unescape 所有路径参数，但保留 RFC 6570 保留字符
	UnescapingModeAllExceptReserved

	// UnescapingModeAllExceptSlash unescape URL 路径参数，但保留路径分隔符 "%2F"
	UnescapingModeAllExceptSlash

	// UnescapingModeAllCharacters unescape 所有 URL 路径参数
	UnescapingModeAllCharacters

	// UnescapingModeDefault 默认 unescape 类型
	UnescapingModeDefault = UnescapingModeLegacy
)

// splitPathEncoded 将路径按 / 和 %2F 分割为组件
// 替代 regexp.Split，手动扫描避免 regex 开销
func splitPathEncoded(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
			i++
		} else if i+2 < len(path) && path[i] == '%' && path[i+1] == '2' && (path[i+2] == 'F' || path[i+2] == 'f') {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 3
			i += 3
		} else {
			i++
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

// HandlerFunc 处理特定路径模式和 HTTP 方法的函数
type HandlerFunc func(w http.ResponseWriter, r *http.Request, pathParams map[string]string)

// Middleware 包装 HandlerFunc 做请求前/后处理的中间件
// 使用直连注册方法时的 gRPC 拦截器替代方案，推荐优先使用 gRPC 拦截器
type Middleware func(HandlerFunc) HandlerFunc

// ServeMux gRPC-gateway 请求多路复用器
// 将 HTTP 请求匹配到路径模式并调用对应的 handler
type ServeMux struct {
	// handlers 按 HTTP 方法分组的 handler 列表
	handlers                  map[string][]handler
	staticHandlers            *routing.StaticIndex[handler]
	routes                    *routing.Table[handler]
	middlewares               []Middleware
	forwardResponseOptions    []func(context.Context, http.ResponseWriter, proto.Message) error
	forwardResponseRewriter   ForwardResponseRewriter
	marshalers                marshalerRegistry
	incomingHeaderMatcher     HeaderMatcherFunc
	outgoingHeaderMatcher     HeaderMatcherFunc
	outgoingTrailerMatcher    HeaderMatcherFunc
	metadataAnnotators        []func(context.Context, *http.Request) metadata.MD
	errorHandler              ErrorHandlerFunc
	streamErrorHandler        StreamErrorHandlerFunc
	routingErrorHandler       RoutingErrorHandlerFunc
	disablePathLengthFallback bool
	unescapingMode            UnescapingMode
	writeContentLength        bool
	requestValidator          RequestValidator
	validationErrorFormatter  ValidationErrorFormatter
	validationSkipper         ValidationSkipper
}

// ServeMuxOption ServeMux 构造选项
type ServeMuxOption func(*ServeMux)

// ForwardResponseRewriter 在转发前重写响应消息的函数签名
type ForwardResponseRewriter func(ctx context.Context, response proto.Message) (any, error)

// WithForwardResponseRewriter 返回允许插入响应重写逻辑的 ServeMuxOption
// 重写函数在 unary/流式/错误转发时均会调用
// 注意：使用此选项可能导致 protoc-gen-openapiv2 生成的文档不准确
func WithForwardResponseRewriter(fwdResponseRewriter ForwardResponseRewriter) ServeMuxOption {
	return func(sm *ServeMux) {
		sm.forwardResponseRewriter = fwdResponseRewriter
	}
}

// WithForwardResponseOption 返回关联响应转发选项的 ServeMuxOption
// 每次转发响应前都会调用 forwardResponseOption
// 当仅发送 header 时 msg 可能为 nil
func WithForwardResponseOption(forwardResponseOption func(context.Context, http.ResponseWriter, proto.Message) error) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.forwardResponseOptions = append(serveMux.forwardResponseOptions, forwardResponseOption)
	}
}

// WithUnescapingMode 设置 unescape 类型，详见 UnescapingMode 定义
func WithUnescapingMode(mode UnescapingMode) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.unescapingMode = mode
	}
}

// WithMiddlewares 为所有 handler 设置服务端中间件
// 使用直连注册方法且无法依赖 gRPC 拦截器时使用，推荐优先使用 gRPC 拦截器
func WithMiddlewares(middlewares ...Middleware) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.middlewares = append(serveMux.middlewares, middlewares...)
	}
}

// SetQueryParameterParser 设置 query 参数解析器
// 配置后生成的 OpenAPI 输出可能不再准确，需谨慎使用
func SetQueryParameterParser(queryParameterParser QueryParameterParser) ServeMuxOption {
	return func(serveMux *ServeMux) {
		currentQueryParser = queryParameterParser
	}
}

// HeaderMatcherFunc 检查 header key 是否应转发到/从 gRPC context
type HeaderMatcherFunc func(string) (string, bool)

// DefaultHeaderMatcher 默认入站 header 匹配器
// IANA 永久 HTTP header 加 grpcgateway- 前缀转发到 gRPC metadata
// Grpc-Metadata- 前缀的 header 去掉前缀后转发
// 其他 header 不转发
func DefaultHeaderMatcher(key string) (string, bool) {
	switch key = textproto.CanonicalMIMEHeaderKey(key); {
	case isPermanentHTTPHeader(key):
		return MetadataPrefix + key, true
	case strings.HasPrefix(key, MetadataHeaderPrefix):
		return key[len(MetadataHeaderPrefix):], true
	}
	return "", false
}

func defaultOutgoingHeaderMatcher(key string) (string, bool) {
	return fmt.Sprintf("%s%s", MetadataHeaderPrefix, key), true
}

func defaultOutgoingTrailerMatcher(key string) (string, bool) {
	return fmt.Sprintf("%s%s", MetadataTrailerPrefix, key), true
}

// WithIncomingHeaderMatcher 返回入站 header 匹配器的 ServeMuxOption
// 匹配器对每个 HTTP 请求 header 调用，返回 true 则转发到 gRPC context
// 可修改 header 名称后再转发
func WithIncomingHeaderMatcher(fn HeaderMatcherFunc) ServeMuxOption {
	for _, header := range fn.matchedMalformedHeaders() {
		logWarnf("The configured forwarding filter would allow %q to be sent to the gRPC server, which will likely cause errors. See https://github.com/grpc/grpc-go/pull/4803#issuecomment-986093310 for more information.", header)
	}

	return func(mux *ServeMux) {
		mux.incomingHeaderMatcher = fn
	}
}

// matchedMalformedHeaders 返回会被转发到 gRPC 服务端的畸形 header 列表
func (fn HeaderMatcherFunc) matchedMalformedHeaders() []string {
	if fn == nil {
		return nil
	}
	headers := make([]string, 0)
	for header := range runtimemeta.MalformedHTTPHeaders {
		out, accept := fn(header)
		if accept && isMalformedHTTPHeader(out) {
			headers = append(headers, out)
		}
	}
	return headers
}

// WithOutgoingHeaderMatcher 返回出站 header 匹配器的 ServeMuxOption
// 匹配器对响应 header 中的每个 metadata 调用，返回 true 则转发到 HTTP 响应
// 可修改 header 名称后再转发
func WithOutgoingHeaderMatcher(fn HeaderMatcherFunc) ServeMuxOption {
	return func(mux *ServeMux) {
		mux.outgoingHeaderMatcher = fn
	}
}

// WithOutgoingTrailerMatcher 返回出站 trailer 匹配器的 ServeMuxOption
// 匹配器对响应 trailer 中的每个 metadata 调用，返回 true 则转发到 HTTP 响应
// 可修改 header 名称后再转发
func WithOutgoingTrailerMatcher(fn HeaderMatcherFunc) ServeMuxOption {
	return func(mux *ServeMux) {
		mux.outgoingTrailerMatcher = fn
	}
}

// WithMetadata 返回传递 metadata 到 gRPC context 的 ServeMuxOption
// 用于从 HTTP 请求读取信息并修改 gRPC context，常见场景是从 cookie 读取 token 写入 context
func WithMetadata(annotator func(context.Context, *http.Request) metadata.MD) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.metadataAnnotators = append(serveMux.metadataAnnotators, annotator)
	}
}

// WithErrorHandler 返回自定义错误处理器的 ServeMuxOption
func WithErrorHandler(fn ErrorHandlerFunc) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.errorHandler = fn
	}
}

// WithStreamErrorHandler 返回自定义流错误处理器的 ServeMuxOption
// 允许自定义服务端流式调用的错误 trailer
// 在写入响应数据之前发生的流错误由 ErrorHandler 处理
// 写入数据后发生的错误必须通过响应体返回，最终消息包含流错误处理器的结果
func WithStreamErrorHandler(fn StreamErrorHandlerFunc) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.streamErrorHandler = fn
	}
}

// WithRoutingErrorHandler 返回自定义路由错误处理器的 ServeMuxOption
// 处理 gRPC 路由选择或执行前发生的错误
// 涉及的状态码：StatusMethodNotAllowed、StatusNotFound、StatusBadRequest
func WithRoutingErrorHandler(fn RoutingErrorHandlerFunc) ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.routingErrorHandler = fn
	}
}

// WithDisablePathLengthFallback 返回禁用路径长度回退的 ServeMuxOption
func WithDisablePathLengthFallback() ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.disablePathLengthFallback = true
	}
}

// WithWriteContentLength 返回启用非流式响应写入 Content-Length 的 ServeMuxOption
func WithWriteContentLength() ServeMuxOption {
	return func(serveMux *ServeMux) {
		serveMux.writeContentLength = true
	}
}

// WithHealthEndpointAt 返回在指定路径注册健康检查端点的 ServeMuxOption
// 调用时将请求转发到上游 gRPC 服务的健康检查（gRPC Health Checking Protocol）
// 如果定义了 service 查询参数，也会转发到 HealthCheckRequest
func WithHealthEndpointAt(healthCheckClient grpc_health_v1.HealthClient, endpointPath string) ServeMuxOption {
	return func(s *ServeMux) {
		// error can be ignored since pattern is definitely valid
		_ = s.HandlePath(
			http.MethodGet, endpointPath, func(w http.ResponseWriter, r *http.Request, _ map[string]string,
			) {
				_, outboundMarshaler := MarshalerForRequest(s, r)

				resp, err := healthCheckClient.Check(r.Context(), &grpc_health_v1.HealthCheckRequest{
					Service: r.URL.Query().Get("service"),
				})
				if err != nil {
					s.errorHandler(r.Context(), s, outboundMarshaler, w, r, err)
					return
				}

				w.Header().Set("Content-Type", "application/json")

				if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
					switch resp.GetStatus() {
					case grpc_health_v1.HealthCheckResponse_NOT_SERVING, grpc_health_v1.HealthCheckResponse_UNKNOWN:
						err = status.Error(codes.Unavailable, resp.String())
					case grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN:
						err = status.Error(codes.NotFound, resp.String())
					}

					s.errorHandler(r.Context(), s, outboundMarshaler, w, r, err)
					return
				}

				_ = outboundMarshaler.NewEncoder(w).Encode(resp)
			})
	}
}

// WithHealthzEndpoint 返回注册 /healthz 端点的 ServeMuxOption
// 通用实现见 WithHealthEndpointAt
func WithHealthzEndpoint(healthCheckClient grpc_health_v1.HealthClient) ServeMuxOption {
	return WithHealthEndpointAt(healthCheckClient, "/healthz")
}

// NewServeMux 创建内部映射为空的 ServeMux
func NewServeMux(opts ...ServeMuxOption) *ServeMux {
	serveMux := &ServeMux{
		handlers:                make(map[string][]handler),
		staticHandlers:          routing.NewStaticIndex[handler](),
		routes:                  routing.NewTable[handler](),
		forwardResponseOptions:  make([]func(context.Context, http.ResponseWriter, proto.Message) error, 0),
		forwardResponseRewriter: func(ctx context.Context, response proto.Message) (any, error) { return response, nil },
		marshalers:              makeMarshalerMIMERegistry(),
		errorHandler:            DefaultHTTPErrorHandler,
		streamErrorHandler:      DefaultStreamErrorHandler,
		routingErrorHandler:     DefaultRoutingErrorHandler,
		unescapingMode:          UnescapingModeDefault,
	}

	for _, opt := range opts {
		opt(serveMux)
	}

	if serveMux.incomingHeaderMatcher == nil {
		serveMux.incomingHeaderMatcher = DefaultHeaderMatcher
	}
	if serveMux.outgoingHeaderMatcher == nil {
		serveMux.outgoingHeaderMatcher = defaultOutgoingHeaderMatcher
	}
	if serveMux.outgoingTrailerMatcher == nil {
		serveMux.outgoingTrailerMatcher = defaultOutgoingTrailerMatcher
	}

	return serveMux
}

// Handle 将 h 关联到 HTTP 方法和路径模式
func (s *ServeMux) Handle(meth string, pat Pattern, h HandlerFunc) {
	if len(s.middlewares) > 0 {
		h = chainMiddlewares(s.middlewares)(h)
	}
	hd := handler{pat: pat, h: h}
	s.registerHandler(meth, pat, hd)
}

// HandleRoute 将 RouteHandler 关联到 HTTP 方法和路径模式
// RouteHandler 直接接收 *Params，避免 map[string]string 分配
func (s *ServeMux) HandleRoute(meth string, pat Pattern, h routing.RouteHandler) {
	hd := handler{pat: pat, routeHandler: h}
	s.registerHandler(meth, pat, hd)
}

func (s *ServeMux) registerHandler(meth string, pat Pattern, hd handler) {
	if path, ok := pat.staticPath(); ok {
		s.staticHandlers.Store(meth, path, hd)
		s.routes.Add(meth, routing.Route[handler]{
			StaticPath: path,
			Value:      hd,
			Match:      matchHandlerPath(hd),
		})
	} else {
		s.routes.Add(meth, routing.Route[handler]{
			Template: pat.String(),
			Value:    hd,
			Match:    matchHandlerPath(hd),
		})
	}
	s.handlers[meth] = append([]handler{hd}, s.handlers[meth]...)
}

// HandlePath 允许用户配置自定义路径处理器
func (s *ServeMux) HandlePath(meth string, pathPattern string, h HandlerFunc) error {
	compiler, err := httprule.Parse(pathPattern)
	if err != nil {
		return fmt.Errorf("parsing path pattern: %w", err)
	}
	tp := compiler.Compile()
	pattern, err := NewPattern(tp.Version, tp.OpCodes, tp.Pool, tp.Verb)
	if err != nil {
		return fmt.Errorf("creating new pattern: %w", err)
	}
	s.Handle(meth, pattern, h)
	return nil
}

// ServeHTTP 将请求分发到第一个匹配 r.Method 和 r.URL.Path 的 handler
func (s *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		_, outboundMarshaler := MarshalerForRequest(s, r)
		s.routingErrorHandler(ctx, s, outboundMarshaler, w, r, http.StatusBadRequest)
		return
	}

	// TODO(v3): remove UnescapingModeLegacy
	if s.unescapingMode != UnescapingModeLegacy && r.URL.RawPath != "" {
		path = r.URL.RawPath
	}

	if override := r.Header.Get("X-HTTP-Method-Override"); override != "" && s.isPathLengthFallback(r) {
		if err := r.ParseForm(); err != nil {
			_, outboundMarshaler := MarshalerForRequest(s, r)
			sterr := status.Error(codes.InvalidArgument, err.Error())
			s.errorHandler(ctx, s, outboundMarshaler, w, r, sterr)
			return
		}
		r.Method = strings.ToUpper(override)
	}

	matchOpts := routing.MatchOptions{UnescapingMode: int(s.unescapingMode)}
	match, ok, err := s.routes.Match(r.Method, path, matchOpts)
	if err != nil {
		s.handleRouteMatchError(ctx, w, r, err)
		return
	}
	if ok {
		s.handleHandler(match.Value, w, r, match.Params)
		match.Release()
		return
	}

	// 如果没有找到匹配的 handler，查找其他方法
	// 处理 POST → GET 的路径长度回退
	// 注意：不急于检查请求，因为需要返回正确的 HTTP 状态码
	// 需要处理回退候选才能确定状态码
	match, ok, err = s.routes.MatchOther(r.Method, path, matchOpts)
	if err != nil {
		s.handleRouteMatchError(ctx, w, r, err)
		return
	}
	if ok {
		// X-HTTP-Method-Override 可选，始终允许回退到 POST
		// 仅考虑 POST → GET 回退，避免回退到 DELETE 等危险操作
		if s.isPathLengthFallback(r) && match.Method == http.MethodGet {
			if err := r.ParseForm(); err != nil {
				_, outboundMarshaler := MarshalerForRequest(s, r)
				sterr := status.Error(codes.InvalidArgument, err.Error())
				s.errorHandler(ctx, s, outboundMarshaler, w, r, sterr)
				match.Release()
				return
			}
			s.handleHandler(match.Value, w, r, match.Params)
			match.Release()
			return
		}
		_, outboundMarshaler := MarshalerForRequest(s, r)
		s.routingErrorHandler(ctx, s, outboundMarshaler, w, r, http.StatusMethodNotAllowed)
		match.Release()
		return
	}

	_, outboundMarshaler := MarshalerForRequest(s, r)
	s.routingErrorHandler(ctx, s, outboundMarshaler, w, r, http.StatusNotFound)
}

// GetForwardResponseOptions 返回 ServeMux 关联的 ForwardResponseOptions
func (s *ServeMux) GetForwardResponseOptions() []func(context.Context, http.ResponseWriter, proto.Message) error {
	return s.forwardResponseOptions
}

func (s *ServeMux) isPathLengthFallback(r *http.Request) bool {
	return !s.disablePathLengthFallback && r.Method == "POST" && r.Header.Get("Content-Type") == "application/x-www-form-urlencoded"
}

type handler struct {
	pat          Pattern
	h            HandlerFunc
	routeHandler routing.RouteHandler
}

func (s *ServeMux) handleHandler(h handler, w http.ResponseWriter, r *http.Request, params *routing.Params) {
	if h.routeHandler != nil {
		h.routeHandler(w, r.WithContext(withHTTPPattern(r.Context(), h.pat)), params)
		return
	}
	h.h(w, r.WithContext(withHTTPPattern(r.Context(), h.pat)), params.Map())
}

func (s *ServeMux) handleRouteMatchError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	if mse, ok := err.(MalformedSequenceError); ok {
		_, outboundMarshaler := MarshalerForRequest(s, r)
		s.errorHandler(ctx, s, outboundMarshaler, w, r, &HTTPStatusError{
			HTTPStatus: http.StatusBadRequest,
			Err:        mse,
		})
		return
	}
	_, outboundMarshaler := MarshalerForRequest(s, r)
	s.errorHandler(ctx, s, outboundMarshaler, w, r, err)
}

func matchHandlerPath(h handler) routing.MatchFunc {
	return func(path string, opts routing.MatchOptions) (*routing.Params, error) {
		pathParams, err := matchPatternPath(h.pat, path, UnescapingMode(opts.UnescapingMode))
		if err != nil {
			if err == ErrNotMatch {
				return nil, routing.ErrNotMatch()
			}
			return nil, err
		}
		params := routing.AcquireParams()
		for key, value := range pathParams {
			params.Add(key, value)
		}
		return params, nil
	}
}

func matchPatternPath(pat Pattern, path string, unescapingMode UnescapingMode) (map[string]string, error) {
	var pathComponents []string
	if unescapingMode == UnescapingModeAllCharacters {
		pathComponents = splitPathEncoded(path[1:])
	} else {
		pathComponents = strings.Split(path[1:], "/")
	}

	lastPathComponent := pathComponents[len(pathComponents)-1]
	var verb string
	patVerb := pat.Verb()

	idx := -1
	if patVerb != "" && strings.HasSuffix(lastPathComponent, ":"+patVerb) {
		idx = len(lastPathComponent) - len(patVerb) - 1
	}
	if idx == 0 {
		return nil, ErrNotMatch
	}

	comps := pathComponents
	if idx > 0 {
		comps = make([]string, len(pathComponents))
		copy(comps, pathComponents)
		comps[len(comps)-1], verb = lastPathComponent[:idx], lastPathComponent[idx+1:]
	}
	return pat.MatchAndEscape(comps, verb, unescapingMode)
}

func chainMiddlewares(mws []Middleware) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		for i := len(mws); i > 0; i-- {
			next = mws[i-1](next)
		}
		return next
	}
}
