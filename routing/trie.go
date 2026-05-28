/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:59:03
 * @FilePath: \grpc-runtime\routing\trie.go
 * @Description: Radix trie 实现，用于动态路由按 path segment 深度匹配，替代线性扫描
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import "strings"

// trieNode radix trie 节点，用于动态路由匹配
// 参数捕获通过 paramChild 和 wildcardChild 实现，literal 子节点通过 children 索引
type trieNode[T any] struct {
	// prefix 该节点对应的路径前缀段（不含 /）
	prefix string

	// children literal 子节点，按 prefix 字典序排列
	children []*trieNode[T]

	// paramChild 单段参数捕获子节点（{param}）
	paramChild *paramNode[T]

	// wildcardChild 深度通配符子节点（** 或 {*path}）
	wildcardChild *trieNode[T]

	// wildcardName 深度通配符捕获的参数名
	wildcardName string

	// value 匹配到该节点时返回的路由值
	value *T

	// verb 匹配到该节点时要求的 HTTP rule verb
	verb string

	// matchFunc 匹配到该节点时使用的自定义匹配函数，仅用于兼容旧 Pattern 路由
	matchFunc MatchFunc
}

// paramNode 包装 trieNode 并关联参数名，用于单段参数捕获
type paramNode[T any] struct {
	name string
	node *trieNode[T]
}

// newTrieNode 创建空的 trie 节点
func newTrieNode[T any]() *trieNode[T] {
	return &trieNode[T]{}
}

// insert 向 trie 中插入一条路由
// 注册期调用，非并发安全；请求期只做 lookup，不修改 trie 结构
func (n *trieNode[T]) insert(segments []segment, paramNames []string, verb string, value T, matchFunc MatchFunc) {
	current := n
	for _, seg := range segments {
		switch seg.kind {
		case segLiteral:
			current = current.findOrCreateLiteralChild(seg.literal)
		case segParam:
			if current.paramChild == nil {
				current.paramChild = &paramNode[T]{
					name: paramNames[seg.paramIdx],
					node: newTrieNode[T](),
				}
			}
			current = current.paramChild.node
		case segWildcard:
			if current.wildcardChild == nil {
				current.wildcardChild = newTrieNode[T]()
			}
			current.wildcardName = paramNames[seg.paramIdx]
			current = current.wildcardChild
		}
	}

	// 在叶节点存储值
	stored := value
	current.value = &stored
	current.verb = verb
	current.matchFunc = matchFunc
}

// findOrCreateLiteralChild 查找或创建指定 prefix 的 literal 子节点
func (n *trieNode[T]) findOrCreateLiteralChild(prefix string) *trieNode[T] {
	// 二分查找
	lo, hi := 0, len(n.children)
	for lo < hi {
		mid := lo + (hi-lo)/2
		cmp := strings.Compare(n.children[mid].prefix, prefix)
		if cmp == 0 {
			return n.children[mid]
		}
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	// 未找到，创建新节点并插入
	child := newTrieNode[T]()
	child.prefix = prefix
	n.children = append(n.children, nil)
	copy(n.children[lo+1:], n.children[lo:])
	n.children[lo] = child
	return child
}

// lookupChild 查找与指定 path segment 精确匹配的 literal 子节点
func (n *trieNode[T]) lookupChild(seg string) *trieNode[T] {
	// 二分查找精确匹配
	lo, hi := 0, len(n.children)
	for lo < hi {
		mid := lo + (hi-lo)/2
		cmp := strings.Compare(n.children[mid].prefix, seg)
		if cmp == 0 {
			return n.children[mid]
		}
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return nil
}

// lookupResult trie 查找结果
type lookupResult[T any] struct {
	value     *T
	params    *Params
	match     bool
	matchFunc MatchFunc
}

// lookup 将请求路径与 trie 匹配
// parts 为按 / 分割后的 path segments（不含前导 /）
func (n *trieNode[T]) lookup(parts []string, verb string, opts MatchOptions) lookupResult[T] {
	var zero lookupResult[T]
	params := AcquireParams()
	result := n.lookupRecursive(parts, 0, verb, params, opts)
	if !result.match {
		ReleaseParams(params)
		return zero
	}
	return result
}

// lookupRecursive 递归匹配路径段
func (n *trieNode[T]) lookupRecursive(parts []string, partIdx int, verb string, params *Params, opts MatchOptions) lookupResult[T] {
	var zero lookupResult[T]

	// 所有 parts 已消费
	if partIdx >= len(parts) {
		if n.value != nil && n.verb == verb {
			return lookupResult[T]{value: n.value, params: params, match: true, matchFunc: n.matchFunc}
		}
		return zero
	}

	seg := parts[partIdx]

	// 1 尝试 literal 子节点（优先级最高）
	if child := n.lookupChild(seg); child != nil {
		result := child.lookupRecursive(parts, partIdx+1, verb, params, opts)
		if result.match {
			return result
		}
	}

	// 2 尝试 param 子节点
	if n.paramChild != nil {
		val, err := unescapeSegment(seg, opts)
		if err == nil {
			params.Add(n.paramChild.name, val)
			result := n.paramChild.node.lookupRecursive(parts, partIdx+1, verb, params, opts)
			if result.match {
				return result
			}
			// 回退：移除刚添加的参数
			params.names = params.names[:len(params.names)-1]
			params.values = params.values[:len(params.values)-1]
		}
	}

	// 3 尝试 wildcard 子节点
	if n.wildcardChild != nil {
		// wildcard 消费所有剩余 segments
		remaining := parts[partIdx:]
		val, err := unescapeSegment(joinPath(remaining), opts)
		if err == nil {
			params.Add(n.wildcardName, val)
			if n.wildcardChild.value != nil && n.wildcardChild.verb == verb {
				return lookupResult[T]{
					value:     n.wildcardChild.value,
					params:    params,
					match:     true,
					matchFunc: n.wildcardChild.matchFunc,
				}
			}
			params.values[len(params.values)-1] = ""
			params.names = params.names[:len(params.names)-1]
			params.values = params.values[:len(params.values)-1]
		}
	}

	return zero
}
