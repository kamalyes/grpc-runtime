/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:57:59
 * @FilePath: \grpc-runtime\routing\route_table.go
 * @Description: 路由表，集成静态索引、radix trie 和回退线性扫描三层匹配
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"errors"
	"sync"
)

var errNotMatch = errors.New("route does not match")

// ErrNotMatch 返回路由不匹配错误
func ErrNotMatch() error { return errNotMatch }

// MatchOptions 请求级路由匹配选项
type MatchOptions struct {
	UnescapingMode int
}

// MatchFunc 路由匹配函数，匹配请求路径并返回捕获的参数
type MatchFunc func(path string, opts MatchOptions) (*Params, error)

// Route 编译后的路由条目
type Route[T any] struct {
	StaticPath string
	Template   string // 原始模板字符串，用于 trie 编译
	Value      T
	Match      MatchFunc
}

// Match 路由查找结果
type Match[T any] struct {
	Value  T
	Params *Params
	Method string
}

// Release 释放 Match 持有的 Params 到对象池
// 调用方在处理完 Match 后应调用此方法，避免 Params 被 GC 回收而无法复用
func (m *Match[T]) Release() {
	if m.Params != nil {
		ReleaseParams(m.Params)
		m.Params = nil
	}
}

// Table 按 HTTP method 存储路由
// 静态路由使用 StaticIndex 做 O(1) 查找
// 动态路由使用 radix trie 按 path segment 深度匹配，替代线性扫描
// 对于无法编译到 trie 的旧 Pattern 路由，回退到线性扫描
// 注册期通过写锁保护并发写入，请求期通过读锁保护路由快照
type Table[T any] struct {
	mu       sync.RWMutex
	static   *StaticIndex[Route[T]]
	tries    map[string]*trieNode[T] // method → radix trie root
	fallback map[string][]Route[T]   // method → 旧 Pattern 路由列表（线性扫描）
	methods  map[string]struct{}     // method 集合，用于 MethodNotAllowed fallback
}

// NewTable 创建空路由表
func NewTable[T any]() *Table[T] {
	return &Table[T]{
		static:   NewStaticIndex[Route[T]](),
		tries:    make(map[string]*trieNode[T]),
		fallback: make(map[string][]Route[T]),
		methods:  make(map[string]struct{}),
	}
}

// Add 在指定 method 下注册路由
// 并发安全，通过写锁保护注册期写入
func (t *Table[T]) Add(method string, route Route[T]) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.methods[method] = struct{}{}

	if route.StaticPath != "" {
		t.static.Store(method, route.StaticPath, route)
		if route.Template == "" {
			return
		}
	}

	// 尝试编译到 trie
	if route.Template != "" {
		ct := compileTemplate(route.Template)
		if ct != nil && len(ct.segments) > 0 {
			root := t.tries[method]
			if root == nil {
				root = newTrieNode[T]()
				t.tries[method] = root
			}
			root.insert(ct.segments, ct.paramNames, ct.verb, route.Value, route.Match)
			if route.Match == nil {
				return
			}
		}
	}

	// 无法编译到 trie，或需要旧 matcher 兜底校准语义时，放入 fallback 线性列表
	t.fallback[method] = append([]Route[T]{route}, t.fallback[method]...)
}

// Match 查找指定 method 和 path 的路由
// 并发安全，通过读锁保护请求期读取
func (t *Table[T]) Match(method string, path string, opts MatchOptions) (Match[T], bool, error) {
	var zero Match[T]
	if t == nil {
		return zero, false, nil
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	// 1 静态路由 O(1) 查找
	if route, ok := t.static.Lookup(method, path); ok {
		return Match[T]{
			Value:  route.Value,
			Params: AcquireParams(),
			Method: method,
		}, true, nil
	}

	// 2 radix trie 匹配
	if root, ok := t.tries[method]; ok {
		parts, verb := splitRequestPathAndVerb(path)
		result := root.lookup(parts, verb, opts)
		if result.match && result.value != nil {
			params := result.params
			if result.matchFunc != nil {
				trieParams := result.params
				var err error
				params, err = result.matchFunc(path, opts)
				if err != nil {
					ReleaseParams(trieParams)
					if !IsNotMatch(err) {
						return zero, false, err
					}
					return t.matchFallback(method, path, opts)
				}
				ReleaseParams(trieParams)
			}
			return Match[T]{
				Value:  *result.value,
				Params: params,
				Method: method,
			}, true, nil
		}
	}

	// 3 回退到线性扫描（旧 Pattern 路由）
	return t.matchFallback(method, path, opts)
}

// MatchOther 在除 excludedMethod 外的所有 method 中查找路由
// 并发安全，通过读锁保护请求期读取
func (t *Table[T]) MatchOther(excludedMethod string, path string, opts MatchOptions) (Match[T], bool, error) {
	var zero Match[T]
	if t == nil {
		return zero, false, nil
	}

	t.mu.RLock()
	// 收集所有 method，避免为了 MethodNotAllowed 把静态路由重复放入 fallback
	seen := make(map[string]bool)
	for m := range t.methods {
		seen[m] = true
	}
	t.mu.RUnlock()

	for method := range seen {
		if method == excludedMethod {
			continue
		}
		if match, ok, err := t.Match(method, path, opts); err != nil || ok {
			return match, ok, err
		}
	}
	return zero, false, nil
}

// matchFallback 线性扫描旧 Pattern 路由
func (t *Table[T]) matchFallback(method string, path string, opts MatchOptions) (Match[T], bool, error) {
	var zero Match[T]
	for _, route := range t.fallback[method] {
		if route.Match == nil {
			continue
		}
		params, err := route.Match(path, opts)
		if err != nil {
			if IsNotMatch(err) {
				continue
			}
			return zero, false, err
		}
		return Match[T]{
			Value:  route.Value,
			Params: params,
			Method: method,
		}, true, nil
	}
	return zero, false, nil
}

// splitRequestPath 将请求路径按 / 分割为 segments
func splitRequestPath(path string) []string {
	parts, _ := splitRequestPathAndVerb(path)
	return parts
}

// splitRequestPathAndVerb 将请求路径拆成 path segments 和尾部 verb
func splitRequestPathAndVerb(path string) ([]string, string) {
	if path == "" || path == "/" {
		return nil, ""
	}
	// 去掉前导 /
	rest := path
	if rest[0] == '/' {
		rest = rest[1:]
	}
	verb := ""
	if idx := findVerbInPath(rest); idx >= 0 {
		verb = rest[idx+1:]
		rest = rest[:idx]
	}
	return splitPath(rest), verb
}

// NotMatchError 路由不匹配错误接口
type NotMatchError interface {
	error
	NotMatch() bool
}

// IsNotMatch 判断 err 是否为路由未命中
func IsNotMatch(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errNotMatch) {
		return true
	}
	notMatch, ok := err.(NotMatchError)
	return ok && notMatch.NotMatch()
}
