/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\static_index_test.go
 * @Description: StaticIndex 单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStaticIndex(t *testing.T) {
	idx := NewStaticIndex[string]()
	assert.NotNil(t, idx)
}

func TestStaticIndexStoreAndLookup(t *testing.T) {
	idx := NewStaticIndex[string]()

	idx.Store("GET", "/v1/users", "handler1")

	val, ok := idx.Lookup("GET", "/v1/users")
	assert.True(t, ok)
	assert.Equal(t, "handler1", val)
}

func TestStaticIndexLookupNotFound(t *testing.T) {
	idx := NewStaticIndex[string]()

	_, ok := idx.Lookup("GET", "/v1/users")
	assert.False(t, ok)
}

func TestStaticIndexLookupWrongMethod(t *testing.T) {
	idx := NewStaticIndex[string]()
	idx.Store("GET", "/v1/users", "handler1")

	_, ok := idx.Lookup("POST", "/v1/users")
	assert.False(t, ok)
}

func TestStaticIndexStoreOverwrite(t *testing.T) {
	idx := NewStaticIndex[string]()
	idx.Store("GET", "/v1/users", "handler1")
	idx.Store("GET", "/v1/users", "handler2")

	val, ok := idx.Lookup("GET", "/v1/users")
	assert.True(t, ok)
	assert.Equal(t, "handler2", val)
}

func TestStaticIndexLen(t *testing.T) {
	idx := NewStaticIndex[string]()
	assert.Equal(t, 0, idx.Len("GET"))

	idx.Store("GET", "/v1/users", "handler1")
	idx.Store("GET", "/v1/tenants", "handler2")
	assert.Equal(t, 2, idx.Len("GET"))
	assert.Equal(t, 0, idx.Len("POST"))
}

func TestStaticIndexLookupNil(t *testing.T) {
	var idx *StaticIndex[string]
	_, ok := idx.Lookup("GET", "/v1/users")
	assert.False(t, ok)
	assert.Equal(t, 0, idx.Len("GET"))
}
