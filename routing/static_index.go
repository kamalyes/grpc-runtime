/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:56:36
 * @FilePath: \grpc-runtime\routing\static_index.go
 * @Description: 静态路由索引，按 HTTP method + path 做 O(1) 精确匹配
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

// StaticIndex 按 HTTP method 和 path 存储精确匹配路由
// 注册发生在启动阶段，请求处理只做两次 map 查找，零额外分配
type StaticIndex[T any] struct {
	byMethod map[string]map[string]T
}

// NewStaticIndex 创建空的静态路由索引
func NewStaticIndex[T any]() *StaticIndex[T] {
	return &StaticIndex[T]{
		byMethod: make(map[string]map[string]T),
	}
}

// Store 添加或替换一条精确匹配路由
func (i *StaticIndex[T]) Store(method string, path string, value T) {
	routes := i.byMethod[method]
	if routes == nil {
		routes = make(map[string]T)
		i.byMethod[method] = routes
	}
	routes[path] = value
}

// Lookup 查找精确匹配路由
func (i *StaticIndex[T]) Lookup(method string, path string) (T, bool) {
	var zero T
	if i == nil {
		return zero, false
	}
	routes := i.byMethod[method]
	if routes == nil {
		return zero, false
	}
	value, ok := routes[path]
	return value, ok
}

// Len 返回指定 method 下注册的路由数量
func (i *StaticIndex[T]) Len(method string) int {
	if i == nil {
		return 0
	}
	return len(i.byMethod[method])
}
