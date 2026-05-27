/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\routing\route_table_test.go
 * @Description: 路由表单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTable(t *testing.T) {
	table := NewTable[string]()
	assert.NotNil(t, table)
}

func TestTableStaticRoute(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		StaticPath: "/v1/users",
		Value:      "users_handler",
	})

	match, ok, err := table.Match("GET", "/v1/users", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "users_handler", match.Value)
	assert.Equal(t, "GET", match.Method)
}

func TestTableDynamicRoute(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/users/{user_id}",
		Value:    "user_handler",
	})

	match, ok, err := table.Match("GET", "/v1/users/123", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "user_handler", match.Value)

	val, found := match.Params.Get("user_id")
	assert.True(t, found)
	assert.Equal(t, "123", val)
}

func TestTableWildcardRoute(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/{path=**}",
		Value:    "wildcard_handler",
	})

	match, ok, err := table.Match("GET", "/v1/a/b/c", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "wildcard_handler", match.Value)

	val, found := match.Params.Get("path")
	assert.True(t, found)
	assert.Equal(t, "a/b/c", val)
}

func TestTableFallbackRoute(t *testing.T) {
	table := NewTable[string]()
	called := false
	table.Add("GET", Route[string]{
		Value: "fallback_handler",
		Match: func(path string, opts MatchOptions) (*Params, error) {
			called = true
			if path == "/custom/match" {
				return NewParams(0), nil
			}
			return nil, errNotMatch
		},
	})

	match, ok, err := table.Match("GET", "/custom/match", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "fallback_handler", match.Value)
	assert.True(t, called)
}

func TestTableNoMatch(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		StaticPath: "/v1/users",
		Value:      "users_handler",
	})

	_, ok, err := table.Match("GET", "/v1/tenants", MatchOptions{})
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestTableMatchOther(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		StaticPath: "/v1/users",
		Value:      "get_handler",
	})
	table.Add("POST", Route[string]{
		StaticPath: "/v1/users",
		Value:      "post_handler",
	})

	match, ok, err := table.MatchOther("GET", "/v1/users", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "post_handler", match.Value)
}

func TestTableStaticRouteNotDuplicatedIntoFallback(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		StaticPath: "/v1/users",
		Value:      "get_handler",
	})
	table.Add("POST", Route[string]{
		StaticPath: "/v1/users",
		Value:      "post_handler",
	})

	assert.Empty(t, table.fallback["GET"])
	assert.Empty(t, table.fallback["POST"])

	match, ok, err := table.MatchOther("GET", "/v1/users", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "post_handler", match.Value)
}

func TestTableVerbRoute(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/users/{user_id}:get",
		Value:    "verb_handler",
	})

	match, ok, err := table.Match("GET", "/v1/users/123:get", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "verb_handler", match.Value)

	val, found := match.Params.Get("user_id")
	assert.True(t, found)
	assert.Equal(t, "123", val)

	_, ok, err = table.Match("GET", "/v1/users/123", MatchOptions{})
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestTableComplexTemplateFallsBackToMatcher(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/{name=projects/*}",
		Value:    "complex_handler",
		Match: func(path string, opts MatchOptions) (*Params, error) {
			if path != "/v1/projects/apex" {
				return nil, ErrNotMatch()
			}
			params := NewParams(1)
			params.Add("name", "projects/apex")
			return params, nil
		},
	})

	match, ok, err := table.Match("GET", "/v1/projects/apex", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "complex_handler", match.Value)

	val, found := match.Params.Get("name")
	assert.True(t, found)
	assert.Equal(t, "projects/apex", val)

	_, ok, err = table.Match("GET", "/v1/tenants/apex", MatchOptions{})
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestTableNil(t *testing.T) {
	var table *Table[string]
	_, ok, err := table.Match("GET", "/v1/users", MatchOptions{})
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestIsNotMatch(t *testing.T) {
	assert.True(t, IsNotMatch(errNotMatch))
	assert.False(t, IsNotMatch(nil))
	assert.False(t, IsNotMatch(errors.New("other error")))
}

func TestErrNotMatch(t *testing.T) {
	err := ErrNotMatch()
	assert.Equal(t, errNotMatch, err)
}

func TestSplitRequestPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"Empty", "", nil},
		{"Root", "/", nil},
		{"Simple", "/v1/users", []string{"v1", "users"}},
		{"TrailingSlash", "/v1/users/", []string{"v1", "users"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitRequestPath(tt.input))
		})
	}
}

func TestTableMultipleDynamicRoutes(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/users/{user_id}",
		Value:    "user_handler",
	})
	table.Add("GET", Route[string]{
		Template: "/v1/tenants/{tenant_id}",
		Value:    "tenant_handler",
	})

	// 匹配 users
	match, ok, err := table.Match("GET", "/v1/users/123", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "user_handler", match.Value)

	// 匹配 tenants
	match, ok, err = table.Match("GET", "/v1/tenants/acme", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "tenant_handler", match.Value)
}

func TestTableStaticPriorityOverDynamic(t *testing.T) {
	table := NewTable[string]()
	table.Add("GET", Route[string]{
		Template: "/v1/users/{user_id}",
		Value:    "dynamic_handler",
	})
	table.Add("GET", Route[string]{
		StaticPath: "/v1/users/me",
		Value:      "static_handler",
	})

	// 静态路由优先
	match, ok, err := table.Match("GET", "/v1/users/me", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "static_handler", match.Value)

	// 动态路由匹配
	match, ok, err = table.Match("GET", "/v1/users/123", MatchOptions{})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "dynamic_handler", match.Value)
}
