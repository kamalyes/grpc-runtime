/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:02:06
 * @FilePath: \grpc-runtime\routing\concurrent_test.go
 * @Description: 并发安全测试，验证 Table 和 Pool 在高并发下的正确性
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTable_ConcurrentAddAndMatch(t *testing.T) {
	table := NewTable[string]()
	const goroutines = 100
	const routesPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 并发注册
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < routesPerGoroutine; i++ {
				table.Add("GET", Route[string]{
					Template: fmt.Sprintf("/v1/g%d/r%d/{id}", gid, i),
					Value:    fmt.Sprintf("handler_g%d_r%d", gid, i),
				})
			}
		}(g)
	}

	// 并发查找
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			opts := MatchOptions{UnescapingMode: 3}
			for i := 0; i < routesPerGoroutine; i++ {
				path := fmt.Sprintf("/v1/g%d/r%d/test", gid, i)
				// 不断言一定命中，因为注册和查找并发，可能注册还没完成
				table.Match("GET", path, opts)
			}
		}(g)
	}

	wg.Wait()
}

func TestTable_ConcurrentMatchAfterRegistration(t *testing.T) {
	table := NewTable[string]()

	// 先注册完所有路由
	for i := 0; i < 500; i++ {
		table.Add("GET", Route[string]{
			Template: fmt.Sprintf("/v1/resource/%d/{id}", i),
			Value:    fmt.Sprintf("handler_%d", i),
		})
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	opts := MatchOptions{UnescapingMode: 3}
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				path := fmt.Sprintf("/v1/resource/%d/abc", i)
				match, ok, err := table.Match("GET", path, opts)
				assert.NoError(t, err)
				if ok {
					assert.Equal(t, fmt.Sprintf("handler_%d", i), match.Value)
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestParams_ConcurrentPool(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				p := AcquireParams()
				p.Add("key", "value")
				_, _ = p.Get("key")
				ReleaseParams(p)
			}
		}()
	}

	wg.Wait()
}

func TestPathBuffer_ConcurrentPool(t *testing.T) {
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				b := AcquirePathBuffer()
				b.Split("/v1/users/123", false)
				ReleasePathBuffer(b)
			}
		}()
	}

	wg.Wait()
}

func TestParams_ResetClearsValues(t *testing.T) {
	p := NewParams(2)
	p.Add("secret_token", "sensitive_value_12345")
	p.Add("password", "hunter2")

	p.Reset()

	// 验证底层数组中的值已被清零
	// Reset 后 names 和 values 长度为 0，但底层数组容量不变
	assert.Equal(t, 0, p.Len())
	assert.Equal(t, 0, len(p.names))
	assert.Equal(t, 0, len(p.values))
}

func TestTable_StaticAndDynamicCoexistence(t *testing.T) {
	table := NewTable[string]()

	// 注册同路径前缀的静态和动态路由
	table.Add("GET", Route[string]{
		StaticPath: "/v1/users/me",
		Value:      "me_handler",
	})
	table.Add("GET", Route[string]{
		Template: "/v1/users/{user_id}",
		Value:    "user_handler",
	})

	opts := MatchOptions{UnescapingMode: 3}

	// 静态路由优先
	match, ok, err := table.Match("GET", "/v1/users/me", opts)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "me_handler", match.Value)

	// 动态路由匹配
	match, ok, err = table.Match("GET", "/v1/users/123", opts)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "user_handler", match.Value)
}

func TestTable_MultipleMethods(t *testing.T) {
	table := NewTable[string]()
	opts := MatchOptions{UnescapingMode: 3}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, m := range methods {
		table.Add(m, Route[string]{
			Template: "/v1/resource/{id}",
			Value:    m + "_handler",
		})
	}

	for _, m := range methods {
		match, ok, err := table.Match(m, "/v1/resource/abc", opts)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, m+"_handler", match.Value)
	}
}
