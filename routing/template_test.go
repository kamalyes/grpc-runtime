/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:54:19
 * @FilePath: \grpc-runtime\routing\template_test.go
 * @Description: 模板编译单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompileTemplateEmpty(t *testing.T) {
	ct := compileTemplate("")
	assert.NotNil(t, ct)
	assert.Equal(t, 0, len(ct.segments))

	ct = compileTemplate("/")
	assert.NotNil(t, ct)
	assert.Equal(t, 0, len(ct.segments))
}

func TestCompileTemplateStatic(t *testing.T) {
	ct := compileTemplate("/v1/users")
	assert.NotNil(t, ct)
	assert.Equal(t, 2, len(ct.segments))
	assert.Equal(t, segLiteral, ct.segments[0].kind)
	assert.Equal(t, "v1", ct.segments[0].literal)
	assert.Equal(t, segLiteral, ct.segments[1].kind)
	assert.Equal(t, "users", ct.segments[1].literal)
}

func TestCompileTemplateParam(t *testing.T) {
	ct := compileTemplate("/v1/users/{user_id}")
	assert.NotNil(t, ct)
	assert.Equal(t, 3, len(ct.segments))

	assert.Equal(t, segLiteral, ct.segments[0].kind)
	assert.Equal(t, "v1", ct.segments[0].literal)

	assert.Equal(t, segLiteral, ct.segments[1].kind)
	assert.Equal(t, "users", ct.segments[1].literal)

	assert.Equal(t, segParam, ct.segments[2].kind)
	assert.Equal(t, 0, ct.segments[2].paramIdx)
	assert.Equal(t, []string{"user_id"}, ct.paramNames)
}

func TestCompileTemplateWildcard(t *testing.T) {
	ct := compileTemplate("/v1/{path=**}")
	assert.NotNil(t, ct)
	assert.Equal(t, 2, len(ct.segments))

	assert.Equal(t, segLiteral, ct.segments[0].kind)
	assert.Equal(t, "v1", ct.segments[0].literal)

	assert.Equal(t, segWildcard, ct.segments[1].kind)
	assert.Equal(t, []string{"path"}, ct.paramNames)
}

func TestCompileTemplateWildcardExplicit(t *testing.T) {
	ct := compileTemplate("/v1/**")
	assert.NotNil(t, ct)
	assert.Equal(t, 2, len(ct.segments))
	assert.Equal(t, segWildcard, ct.segments[1].kind)
}

func TestCompileTemplateVerb(t *testing.T) {
	ct := compileTemplate("/v1/users:list")
	assert.NotNil(t, ct)
	assert.Equal(t, "list", ct.verb)
	assert.Equal(t, 2, len(ct.segments))
}

func TestCompileTemplateVerbWithParam(t *testing.T) {
	ct := compileTemplate("/v1/users/{user_id}:get")
	assert.NotNil(t, ct)
	assert.Equal(t, "get", ct.verb)
	assert.Equal(t, 3, len(ct.segments))
	assert.Equal(t, segParam, ct.segments[2].kind)
}

func TestParseParamPart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantExpr string
		wantOK   bool
	}{
		{"SimpleParam", "{user_id}", "user_id", "", true},
		{"ParamWithPattern", "{name=projects/*}", "name", "projects/*", true},
		{"DeepWildcard", "{path=**}", "path", "**", true},
		{"NoBraces", "plain", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, expr, ok := parseParamPart(tt.input)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantExpr, expr)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestCompileTemplateComplexPatternFallsBack(t *testing.T) {
	assert.Nil(t, compileTemplate("/v1/{name=projects/*}"))
	assert.Nil(t, compileTemplate("/v1/{name=prefix/**}"))
}

func TestFindVerbSeparator(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"NoVerb", "v1/users", -1},
		{"VerbAtEnd", "v1/users:list", 8},
		{"VerbInsideBraces", "v1/{name:test}", -1},
		{"Empty", "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, findVerbSeparator(tt.input))
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"Empty", "", nil},
		{"Single", "v1", []string{"v1"}},
		{"Multiple", "v1/users/123", []string{"v1", "users", "123"}},
		{"TrailingSlash", "v1/users/", []string{"v1", "users"}},
		{"LeadingSlash", "/v1/users", []string{"v1", "users"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitPath(tt.input))
		})
	}
}

func TestJoinPath(t *testing.T) {
	parts := []string{"v1", "users", "123"}
	assert.Equal(t, "v1/users/123", joinPath(parts))

	parts = []string{"single"}
	assert.Equal(t, "single", joinPath(parts))

	parts = []string{}
	assert.Equal(t, "", joinPath(parts))
}

func TestCompiledTemplateMatch(t *testing.T) {
	ct := compileTemplate("/v1/users/{user_id}")
	assert.NotNil(t, ct)

	params, err := ct.match("/v1/users/123", MatchOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, params)
	val, ok := params.Get("user_id")
	assert.True(t, ok)
	assert.Equal(t, "123", val)
}

func TestCompiledTemplateMatchNotFound(t *testing.T) {
	ct := compileTemplate("/v1/users/{user_id}")

	params, err := ct.match("/v2/users/123", MatchOptions{})
	assert.Error(t, err)
	assert.Nil(t, params)
}

func TestCompiledTemplateMatchStatic(t *testing.T) {
	ct := compileTemplate("/v1/users")

	params, err := ct.match("/v1/users", MatchOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, 0, params.Len())
}

func TestCompiledTemplateMatchVerb(t *testing.T) {
	ct := compileTemplate("/v1/users/{user_id}:get")

	params, err := ct.match("/v1/users/123:get", MatchOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, params)
	val, ok := params.Get("user_id")
	assert.True(t, ok)
	assert.Equal(t, "123", val)
}

func TestCompiledTemplateMatchWildcard(t *testing.T) {
	ct := compileTemplate("/v1/{path=**}")

	params, err := ct.match("/v1/a/b/c", MatchOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, params)
	val, ok := params.Get("path")
	assert.True(t, ok)
	assert.Equal(t, "a/b/c", val)
}
