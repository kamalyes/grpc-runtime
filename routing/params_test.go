/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\params_test.go
 * @Description: Params 单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParams(t *testing.T) {
	p := NewParams(4)
	assert.NotNil(t, p)
	assert.Equal(t, 0, p.Len())
}

func TestParamsAdd(t *testing.T) {
	p := NewParams(2)
	p.Add("user_id", "123")
	p.Add("tenant_id", "acme")

	assert.Equal(t, 2, p.Len())
}

func TestParamsGet(t *testing.T) {
	p := NewParams(2)
	p.Add("user_id", "123")
	p.Add("tenant_id", "acme")

	val, ok := p.Get("user_id")
	assert.True(t, ok)
	assert.Equal(t, "123", val)

	val, ok = p.Get("tenant_id")
	assert.True(t, ok)
	assert.Equal(t, "acme", val)

	_, ok = p.Get("nonexistent")
	assert.False(t, ok)
}

func TestParamsGetNil(t *testing.T) {
	var p *Params
	_, ok := p.Get("key")
	assert.False(t, ok)
	assert.Equal(t, 0, p.Len())
}

func TestParamsMap(t *testing.T) {
	p := NewParams(2)
	p.Add("user_id", "123")
	p.Add("tenant_id", "acme")

	m := p.Map()
	assert.Equal(t, 2, len(m))
	assert.Equal(t, "123", m["user_id"])
	assert.Equal(t, "acme", m["tenant_id"])
}

func TestParamsMapEmpty(t *testing.T) {
	p := NewParams(0)
	m := p.Map()
	assert.Equal(t, 0, len(m))
}

func TestParamsMapNil(t *testing.T) {
	var p *Params
	m := p.Map()
	assert.Equal(t, 0, len(m))
}
