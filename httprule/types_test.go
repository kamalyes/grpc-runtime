/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:02:15
 * @FilePath: \grpc-runtime\httprule\types_test.go
 * @Description: 测试 HTTP 规则类型
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package httprule

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateStringer(t *testing.T) {
	for _, spec := range []struct {
		segs []segment
		want string
	}{
		{segs: []segment{literal("v1")}, want: "/v1"},
		{segs: []segment{wildcard{}}, want: "/*"},
		{segs: []segment{deepWildcard{}}, want: "/**"},
		{segs: []segment{variable{path: "name", segments: []segment{literal("a")}}}, want: "/{name=a}"},
		{segs: []segment{variable{path: "name", segments: []segment{literal("a"), wildcard{}, literal("b")}}}, want: "/{name=a/*/b}"},
		{segs: []segment{literal("v1"), variable{path: "name", segments: []segment{literal("a"), wildcard{}, literal("b")}}, literal("c"), variable{path: "field.nested", segments: []segment{wildcard{}, literal("d")}}, wildcard{}, literal("e"), deepWildcard{}}, want: "/v1/{name=a/*/b}/c/{field.nested=*/d}/*/e/**"},
	} {
		tmpl := template{segments: spec.segs}
		assert.Equal(t, spec.want, tmpl.String(), "%#v.String()", tmpl)

		tmpl.verb = "LOCK"
		assert.Equal(t, fmt.Sprintf("%s:LOCK", spec.want), tmpl.String(), "%#v.String() with verb", tmpl)
	}
}
