/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\pool.go
 * @Description: Params 和 PathBuffer 的对象池化，减少请求热路径上的 GC 压力
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import "github.com/kamalyes/go-toolbox/pkg/syncx"

// paramsPool Params 对象池，复用 Params 实例避免每次请求分配
var paramsPool = syncx.NewPool[*Params](func() *Params {
	return NewParams(4)
})

// AcquireParams 从池中获取一个已重置的 Params 实例
func AcquireParams() *Params {
	p := paramsPool.Get()
	p.Reset()
	return p
}

// ReleaseParams 将 Params 实例放回池中
func ReleaseParams(p *Params) {
	if p != nil {
		paramsPool.Put(p)
	}
}

// pathBufferPool PathBuffer 对象池，复用 PathBuffer 实例避免每次请求分配
var pathBufferPool = syncx.NewPool[*PathBuffer](func() *PathBuffer {
	return &PathBuffer{}
})

// AcquirePathBuffer 从池中获取一个已重置的 PathBuffer 实例
func AcquirePathBuffer() *PathBuffer {
	b := pathBufferPool.Get()
	b.Reset()
	return b
}

// ReleasePathBuffer 将 PathBuffer 实例放回池中
func ReleasePathBuffer(b *PathBuffer) {
	if b != nil {
		pathBufferPool.Put(b)
	}
}
