/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\bench_test.go
 * @Description: 路由性能基准测试，覆盖静态路由、动态路由、参数捕获、对象池等场景
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"fmt"
	"net/http"
	"testing"
)

// buildRouteTable 构建 n 条静态路由 + n 条动态路由的测试表
func buildRouteTable(n int) *Table[string] {
	table := NewTable[string]()

	// 注册 n 条静态路由
	for i := 0; i < n; i++ {
		table.Add("GET", Route[string]{
			StaticPath: fmt.Sprintf("/v1/static/%d", i),
			Value:      fmt.Sprintf("static_handler_%d", i),
		})
	}

	// 注册 n 条动态路由
	for i := 0; i < n; i++ {
		table.Add("GET", Route[string]{
			Template: fmt.Sprintf("/v1/dynamic/%d/{id}", i),
			Value:    fmt.Sprintf("dynamic_handler_%d", i),
		})
	}

	return table
}

// matchAndRelease 执行路由匹配并在完成后释放 Params，模拟真实请求生命周期
func matchAndRelease[T any](table *Table[T], method string, path string, opts MatchOptions) (Match[T], bool, error) {
	match, ok, err := table.Match(method, path, opts)
	if ok {
		match.Release()
	}
	return match, ok, err
}

// BenchmarkTable_StaticRoute_100 静态路由 100 条
func BenchmarkTable_StaticRoute_100(b *testing.B) {
	table := buildRouteTable(100)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/static/50", opts)
	}
}

// BenchmarkTable_StaticRoute_1000 静态路由 1000 条
func BenchmarkTable_StaticRoute_1000(b *testing.B) {
	table := buildRouteTable(1000)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/static/500", opts)
	}
}

// BenchmarkTable_StaticRoute_10000 静态路由 10000 条
func BenchmarkTable_StaticRoute_10000(b *testing.B) {
	table := buildRouteTable(10000)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/static/5000", opts)
	}
}

// BenchmarkTable_DynamicRoute_100 动态路由 100 条
func BenchmarkTable_DynamicRoute_100(b *testing.B) {
	table := buildRouteTable(100)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/dynamic/50/abc", opts)
	}
}

// BenchmarkTable_DynamicRoute_1000 动态路由 1000 条
func BenchmarkTable_DynamicRoute_1000(b *testing.B) {
	table := buildRouteTable(1000)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/dynamic/500/abc", opts)
	}
}

// BenchmarkTable_DynamicRoute_10000 动态路由 10000 条
func BenchmarkTable_DynamicRoute_10000(b *testing.B) {
	table := buildRouteTable(10000)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/dynamic/5000/abc", opts)
	}
}

// BenchmarkTable_PathParams 1/3/5 个路径参数
func BenchmarkTable_PathParams_1(b *testing.B) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/{a}",
		Value:    "handler",
	})
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/val1", opts)
	}
}

func BenchmarkTable_PathParams_3(b *testing.B) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/{a}/{b}/{c}",
		Value:    "handler",
	})
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/val1/val2/val3", opts)
	}
}

func BenchmarkTable_PathParams_5(b *testing.B) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/{a}/{b}/{c}/{d}/{e}",
		Value:    "handler",
	})
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/val1/val2/val3/val4/val5", opts)
	}
}

// BenchmarkTable_WildcardRoute 通配符路由
func BenchmarkTable_WildcardRoute(b *testing.B) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/{path=**}",
		Value:    "wildcard_handler",
	})
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/a/b/c/d/e", opts)
	}
}

// BenchmarkTable_VerbRoute verb 路由
func BenchmarkTable_VerbRoute(b *testing.B) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/users/{user_id}:get",
		Value:    "verb_handler",
	})
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchAndRelease(table, "GET", "/v1/users/123:get", opts)
	}
}

// BenchmarkTable_NoMatch 未命中路由
func BenchmarkTable_NoMatch(b *testing.B) {
	table := buildRouteTable(1000)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.Match("GET", "/v1/nonexistent/path", opts)
	}
}

// BenchmarkParams_Pool Params 对象池
func BenchmarkParams_Pool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := AcquireParams()
		p.Add("key", "value")
		ReleaseParams(p)
	}
}

// BenchmarkParams_New 直接创建 Params
func BenchmarkParams_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParams(1)
		p.Add("key", "value")
		_ = p
	}
}

// BenchmarkParams_Get 懒加载 index 后的 Get 调用
func BenchmarkParams_Get(b *testing.B) {
	p := NewParams(5)
	p.Add("a", "1")
	p.Add("b", "2")
	p.Add("c", "3")
	p.Add("d", "4")
	p.Add("e", "5")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Get("c")
	}
}

// BenchmarkParams_Map 懒加载 mmap 后的 Map 调用
func BenchmarkParams_Map(b *testing.B) {
	p := NewParams(5)
	p.Add("a", "1")
	p.Add("b", "2")
	p.Add("c", "3")
	p.Add("d", "4")
	p.Add("e", "5")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Map()
	}
}

// BenchmarkStaticIndex_Lookup 静态索引查找
func BenchmarkStaticIndex_Lookup(b *testing.B) {
	idx := NewStaticIndex[string]()
	for i := 0; i < 1000; i++ {
		idx.Store("GET", fmt.Sprintf("/v1/resource/%d", i), fmt.Sprintf("handler_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Lookup("GET", "/v1/resource/500")
	}
}

// BenchmarkCompileTemplate 模板编译
func BenchmarkCompileTemplate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compileTemplate("/v1/users/{user_id}/posts/{post_id}")
	}
}

// BenchmarkTable_ConcurrentMatch 并发路由匹配
func BenchmarkTable_ConcurrentMatch(b *testing.B) {
	table := buildRouteTable(1000)
	opts := MatchOptions{UnescapingMode: 3}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			path := fmt.Sprintf("/v1/dynamic/%d/abc", i%1000)
			match, ok, _ := table.Match(http.MethodGet, path, opts)
			if ok {
				match.Release()
			}
			i++
		}
	})
}
