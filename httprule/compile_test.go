/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:55:15
 * @FilePath: \grpc-runtime\httprule\compile_test.go
 * @Description: 测试 HTTP 规则编译器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package httprule

import (
	"testing"

	"github.com/kamalyes/grpc-runtime/utilities"
	"github.com/stretchr/testify/assert"
)

const (
	operandFiller = 0
)

func TestCompile(t *testing.T) {
	for _, spec := range []struct {
		segs   []segment
		verb   string
		ops    []int
		pool   []string
		fields []string
	}{
		{},
		{segs: []segment{literal(eof)}, ops: []int{int(utilities.OpLitPush), 0}, pool: []string{""}},
		{segs: []segment{wildcard{}}, ops: []int{int(utilities.OpPush), operandFiller}},
		{segs: []segment{deepWildcard{}}, ops: []int{int(utilities.OpPushM), operandFiller}},
		{segs: []segment{literal("v1")}, ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"v1"}},
		{segs: []segment{literal("v1")}, verb: "LOCK", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"v1"}},
		{
			segs: []segment{variable{path: "name.nested", segments: []segment{wildcard{}}}},
			ops:  []int{int(utilities.OpPush), operandFiller, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 0},
			pool: []string{"name.nested"}, fields: []string{"name.nested"},
		},
		{
			segs: []segment{literal("obj"), variable{path: "name.nested", segments: []segment{literal("a"), wildcard{}, literal("b")}}, variable{path: "obj", segments: []segment{deepWildcard{}}}},
			ops: []int{
				int(utilities.OpLitPush), 0,
				int(utilities.OpLitPush), 1,
				int(utilities.OpPush), operandFiller,
				int(utilities.OpLitPush), 2,
				int(utilities.OpConcatN), 3,
				int(utilities.OpCapture), 3,
				int(utilities.OpPushM), operandFiller,
				int(utilities.OpConcatN), 1,
				int(utilities.OpCapture), 0,
			},
			pool: []string{"obj", "a", "b", "name.nested"}, fields: []string{"name.nested", "obj"},
		},
	} {
		tmpl := template{segments: spec.segs, verb: spec.verb}
		compiled := tmpl.Compile()
		assert.Equal(t, opcodeVersion, compiled.Version, "tmpl.Compile().Version; segs=%#v, verb=%q", spec.segs, spec.verb)
		assert.Equal(t, spec.ops, compiled.OpCodes, "tmpl.Compile().OpCodes; segs=%#v, verb=%q", spec.segs, spec.verb)
		assert.Equal(t, spec.pool, compiled.Pool, "tmpl.Compile().Pool; segs=%#v, verb=%q", spec.segs, spec.verb)
		assert.Equal(t, spec.verb, compiled.Verb, "tmpl.Compile().Verb; segs=%#v, verb=%q", spec.segs, spec.verb)
		assert.Equal(t, spec.fields, compiled.Fields, "tmpl.Compile().Fields; segs=%#v, verb=%q", spec.segs, spec.verb)
	}
}
