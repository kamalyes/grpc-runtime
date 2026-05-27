/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:30:55
 * @FilePath: \grpc-runtime\route_desc.go
 * @Description: 生成器使用的路由描述 facade，隐藏旧 Pattern 注册细节
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"context"
	"io"
	"net/http"

	"github.com/kamalyes/grpc-runtime/binding"
	"github.com/kamalyes/grpc-runtime/utilities"
	"google.golang.org/protobuf/proto"
)

// IOReaderFactory 创建可重复读取同一数据源的 reader 工厂
func IOReaderFactory(r io.Reader) (func() io.Reader, error) {
	return utilities.IOReaderFactory(r)
}

// --- 类型别名，委托给 binding 子包 ---

// BodyBinding 描述 HTTP body 与 proto request 字段的绑定关系
type BodyBinding = binding.BodyBinding

// QueryParamFilter 是 query 参数过滤器，生成代码无需感知内部索引实现。
type QueryParamFilter = binding.QueryFilter

// NoBody 返回无 HTTP body 的绑定描述
func NoBody() BodyBinding {
	return binding.NoBody()
}

// Body 返回指定字段路径的 HTTP body 绑定描述
func Body(fieldPath string) BodyBinding {
	return binding.Body(fieldPath)
}

// QueryFilter 创建 query 参数过滤器。
func QueryFilter(fields ...string) QueryParamFilter {
	return binding.NewQueryFilter(fields...)
}

// NewQueryFilter 兼容旧调用，内部等价于 QueryFilter。
func NewQueryFilter(fields ...string) QueryParamFilter { return QueryFilter(fields...) }

// --- RouteInvoker 类型 ---

// RouteInvoker 是生成器产出的类型安全 invoker 函数签名
// 新生成代码只保留必须强类型的 invoker，不再生成 forward_Xxx
type RouteInvoker func(ctx context.Context, req proto.Message, target any) (proto.Message, ServerMetadata, error)

// --- RouteDesc ---

// RouteDesc 描述一条由生成器产出的 HTTP 到 gRPC 路由
// 新生成代码只依赖此结构体，不再直接生成 runtime.NewPattern / utilities.DoubleArray / forward_Xxx
type RouteDesc struct {
	// Method HTTP 方法，如 http.MethodGet
	Method string
	// Template 路径模板，如 "/v1/users/{user_id}"
	Template string
	// Operation gRPC 方法全限定名，如 "/apex.api.UserService/UserGet"
	Operation string
	// Request 创建新的请求消息实例的工厂函数
	Request func() proto.Message
	// Body HTTP body 绑定描述，使用 NoBody() 或 Body("field_path")
	Body BodyBinding
	// QueryFilter query 参数过滤器，使用 QueryFilter("field1", "field2") 构建
	QueryFilter QueryParamFilter
	// Invoker 类型安全的 gRPC 调用函数
	Invoker RouteInvoker
	// Handler 兼容旧生成代码的 HandlerFunc，优先使用 Invoker
	Handler HandlerFunc
}

// --- RegisterRoutes ---

// RegisterRoutes 将生成器产出的 RouteDesc 注册到 mux
// 每条路由的 handler 内部完成完整的请求构建 pipeline：
//
//	new request message → decode body → apply path params → apply query params → apply field mask → validate
//
// 最后调用 RouteInvoker 完成 gRPC 调用并转发响应
func RegisterRoutes(_ context.Context, mux *ServeMux, routes []RouteDesc) error {
	for _, route := range routes {
		if err := registerRoute(mux, route); err != nil {
			return err
		}
	}
	return nil
}

// registerRoute 将单条 RouteDesc 注册到 mux
// 如果 RouteDesc 提供了 Request + Invoker，则使用 BuildRequest pipeline
// 否则回退到简单的 HandlePath 注册（兼容旧 HandlerFunc 模式）
func registerRoute(mux *ServeMux, route RouteDesc) error {
	if route.Request != nil && route.Invoker != nil {
		return mux.HandlePath(route.Method, route.Template, newRouteHandler(mux, route))
	}
	// 兼容模式：只有 Method + Template + Handler
	if route.Handler != nil {
		return mux.HandlePath(route.Method, route.Template, route.Handler)
	}
	// 最小模式：只有 Method + Template，注册空 handler
	return mux.HandlePath(route.Method, route.Template, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {})
}

// newRouteHandler 根据 RouteDesc 创建完整的请求处理 handler
func newRouteHandler(mux *ServeMux, route RouteDesc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := r.Context()

		// 1. 创建新的请求消息
		msg := route.Request()

		// 2. 构建请求：decode body → path params → query params → field mask → validate
		if err := BuildRequest(ctx, mux, r, msg, pathParams, route.Body, route.QueryFilter); err != nil {
			_, outboundMarshaler := MarshalerForRequest(mux, r)
			mux.errorHandler(ctx, mux, outboundMarshaler, w, r, err)
			return
		}

		// 3. 调用 gRPC invoker
		resp, md, err := route.Invoker(ctx, msg, nil)
		if err != nil {
			_, outboundMarshaler := MarshalerForRequest(mux, r)
			mux.errorHandler(ctx, mux, outboundMarshaler, w, r, err)
			return
		}

		// 4. 将 ServerMetadata 注入 context
		ctx = NewServerMetadataContext(ctx, md)

		// 5. 转发响应
		_, outboundMarshaler := MarshalerForRequest(mux, r)
		ForwardResponseMessage(ctx, mux, outboundMarshaler, w, r, resp)
	}
}
