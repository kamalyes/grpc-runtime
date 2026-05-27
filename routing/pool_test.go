/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\pool_test.go
 * @Description: 对象池单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAcquireAndReleaseParams(t *testing.T) {
	p := AcquireParams()
	assert.NotNil(t, p)
	assert.Equal(t, 0, p.Len())

	p.Add("user_id", "123")
	assert.Equal(t, 1, p.Len())

	ReleaseParams(p)
}

func TestAcquireParamsReused(t *testing.T) {
	p1 := AcquireParams()
	p1.Add("key", "val1")
	ReleaseParams(p1)

	p2 := AcquireParams()
	// 复用后应已重置
	assert.Equal(t, 0, p2.Len())

	ReleaseParams(p2)
}

func TestReleaseParamsNil(t *testing.T) {
	ReleaseParams(nil) // 不应 panic
}

func TestAcquireAndReleasePathBuffer(t *testing.T) {
	b := AcquirePathBuffer()
	assert.NotNil(t, b)

	parts := b.Split("/v1/users", false)
	assert.Equal(t, []string{"v1", "users"}, parts)

	ReleasePathBuffer(b)
}

func TestAcquirePathBufferReused(t *testing.T) {
	b1 := AcquirePathBuffer()
	b1.Split("/v1/users", false)
	ReleasePathBuffer(b1)

	b2 := AcquirePathBuffer()
	// 复用后应已重置
	parts := b2.Split("/v2/tenants", false)
	assert.Equal(t, []string{"v2", "tenants"}, parts)

	ReleasePathBuffer(b2)
}

func TestReleasePathBufferNil(t *testing.T) {
	ReleasePathBuffer(nil) // 不应 panic
}
