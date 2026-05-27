/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:59:27
 * @FilePath: \grpc-runtime\routing\params.go
 * @Description: 路径参数容器，使用 slice 存储避免热路径分配 map
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

// Params 存储路由匹配时捕获的路径参数
// 使用 slice 存储，热路径避免分配 map；仅在兼容旧 HandlerFunc 时才构建 map 视图
// index 和 mmap 均为懒加载，首次调用 Get 或 Map 时按需创建
type Params struct {
	names  []string
	values []string
	index  map[string]int    // 懒加载，按名称快速定位
	mmap   map[string]string // 懒加载，仅 Map() 调用时创建
}

// NewParams 创建空参数集，预分配 size 个槽位
func NewParams(size int) *Params {
	return &Params{
		names:  make([]string, 0, size),
		values: make([]string, 0, size),
	}
}

// Add 追加一个捕获的路径参数
func (p *Params) Add(name string, value string) {
	p.names = append(p.names, name)
	p.values = append(p.values, value)
	// 使懒加载缓存失效
	p.index = nil
	p.mmap = nil
}

// Len 返回已捕获的路径参数数量
func (p *Params) Len() int {
	if p == nil {
		return 0
	}
	return len(p.names)
}

// Get 返回指定名称的参数值
// 首次调用时构建 index，后续调用 O(1)
func (p *Params) Get(name string) (string, bool) {
	if p == nil {
		return "", false
	}
	if p.index == nil {
		p.buildIndex()
	}
	idx, ok := p.index[name]
	if !ok {
		return "", false
	}
	return p.values[idx], true
}

// Map 返回参数的 map 视图，用于兼容旧 HandlerFunc(map[string]string) 签名
// 首次调用时构建 mmap，后续调用直接返回
func (p *Params) Map() map[string]string {
	if p == nil || len(p.names) == 0 {
		return map[string]string{}
	}
	if p.mmap == nil {
		p.mmap = make(map[string]string, len(p.names))
		for i, name := range p.names {
			p.mmap[name] = p.values[i]
		}
	}
	return p.mmap
}

// Reset 清空参数集，复用底层 slice 避免重新分配
// 清零残留值防止敏感信息泄漏
func (p *Params) Reset() {
	// 清零残留值，防止通过复用的 slice 泄漏前一次请求的参数
	for i := range p.values {
		p.values[i] = ""
	}
	p.names = p.names[:0]
	p.values = p.values[:0]
	p.index = nil
	p.mmap = nil
}

// buildIndex 构建名称到索引的映射
func (p *Params) buildIndex() {
	p.index = make(map[string]int, len(p.names))
	for i, name := range p.names {
		p.index[name] = i
	}
}
