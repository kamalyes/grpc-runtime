/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:57:21
 * @FilePath: \grpc-runtime\routing\handler.go
 * @Description: 路由处理器类型定义，替代旧的 HandlerFunc(map[string]string) 签名
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import "net/http"

// RouteHandler 新版路由处理器签名
// 接收 *Params 替代 map[string]string，避免每次请求分配 map
// 旧 HandlerFunc 可通过 RouteHandlerFromFunc 适配
type RouteHandler func(w http.ResponseWriter, r *http.Request, params *Params)

// RouteHandlerFromFunc 将旧版 HandlerFunc(map[string]string) 适配为 RouteHandler
func RouteHandlerFromFunc(fn func(w http.ResponseWriter, r *http.Request, pathParams map[string]string)) RouteHandler {
	return func(w http.ResponseWriter, r *http.Request, params *Params) {
		fn(w, r, params.Map())
	}
}

// ToHandlerFunc 将 RouteHandler 转换为旧版 HandlerFunc 签名，用于兼容旧注册接口
func (h RouteHandler) ToHandlerFunc() func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		params := NewParams(len(pathParams))
		for k, v := range pathParams {
			params.Add(k, v)
		}
		h(w, r, params)
	}
}
