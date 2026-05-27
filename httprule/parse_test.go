/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:55:15
 * @FilePath: \grpc-runtime\httprule\parse_test.go
 * @Description: 测试 HTTP 规则解析器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package httprule

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/grpclog"
)

func TestTokenize(t *testing.T) {
	for _, spec := range []struct {
		src    string
		tokens []string
		verb   string
	}{
		{src: "", tokens: []string{eof}},
		{src: "v1", tokens: []string{"v1", eof}},
		{src: "v1/b", tokens: []string{"v1", "/", "b", eof}},
		{src: "v1/endpoint/*", tokens: []string{"v1", "/", "endpoint", "/", "*", eof}},
		{src: "v1/endpoint/**", tokens: []string{"v1", "/", "endpoint", "/", "**", eof}},
		{src: "v1/b/{bucket_name=*}", tokens: []string{"v1", "/", "b", "/", "{", "bucket_name", "=", "*", "}", eof}},
		{src: "v1/b/{bucket_name=buckets/*}", tokens: []string{"v1", "/", "b", "/", "{", "bucket_name", "=", "buckets", "/", "*", "}", eof}},
		{src: "v1/b/{bucket_name=buckets/*}/o", tokens: []string{"v1", "/", "b", "/", "{", "bucket_name", "=", "buckets", "/", "*", "}", "/", "o", eof}},
		{src: "v1/b/{bucket_name=buckets/*}/o/{name}", tokens: []string{"v1", "/", "b", "/", "{", "bucket_name", "=", "buckets", "/", "*", "}", "/", "o", "/", "{", "name", "}", eof}},
		{src: "v1/a=b&c=d;e=f:g/endpoint.rdf", tokens: []string{"v1", "/", "a=b&c=d;e=f:g", "/", "endpoint.rdf", eof}},
		{src: "v1/a/{endpoint}:a", tokens: []string{"v1", "/", "a", "/", "{", "endpoint", "}", eof}, verb: "a"},
		{src: "v1/a/{endpoint}:b:c", tokens: []string{"v1", "/", "a", "/", "{", "endpoint", "}", eof}, verb: "b:c"},
	} {
		tokens, verb := tokenize(spec.src)
		assert.Equal(t, spec.tokens, tokens, "tokenize(%q) tokens", spec.src)

		switch {
		case spec.verb != "":
			assert.Equal(t, spec.verb, verb, "tokenize(%q) verb", spec.src)
		default:
			assert.Equal(t, "", verb, "tokenize(%q) verb", spec.src)

			src := fmt.Sprintf("%s:%s", spec.src, "LOCK")
			tokens, verb = tokenize(src)
			assert.Equal(t, spec.tokens, tokens, "tokenize(%q) tokens", src)
			assert.Equal(t, "LOCK", verb, "tokenize(%q) verb", src)
		}
	}
}

func TestParseSegments(t *testing.T) {
	for _, spec := range []struct {
		tokens []string
		want   []segment
	}{
		{tokens: []string{eof}, want: []segment{literal(eof)}},
		{tokens: []string{eof, "v1", eof}, want: []segment{literal(eof)}},
		{tokens: []string{"v1", eof}, want: []segment{literal("v1")}},
		{tokens: []string{"/", eof}, want: []segment{wildcard{}}},
		{tokens: []string{"-._~!$&'()*+,;=:@", eof}, want: []segment{literal("-._~!$&'()*+,;=:@")}},
		{tokens: []string{"%e7%ac%ac%e4%b8%80%e7%89%88", eof}, want: []segment{literal("%e7%ac%ac%e4%b8%80%e7%89%88")}},
		{tokens: []string{"v1", "/", "*", eof}, want: []segment{literal("v1"), wildcard{}}},
		{tokens: []string{"v1", "/", "**", eof}, want: []segment{literal("v1"), deepWildcard{}}},
		{tokens: []string{"{", "name", "}", eof}, want: []segment{variable{path: "name", segments: []segment{wildcard{}}}}},
		{tokens: []string{"{", "name", "=", "*", "}", eof}, want: []segment{variable{path: "name", segments: []segment{wildcard{}}}}},
		{tokens: []string{"{", "field", ".", "nested", ".", "nested2", "=", "*", "}", eof}, want: []segment{variable{path: "field.nested.nested2", segments: []segment{wildcard{}}}}},
		{tokens: []string{"{", "name", "=", "a", "/", "b", "/", "*", "}", eof}, want: []segment{variable{path: "name", segments: []segment{literal("a"), literal("b"), wildcard{}}}}},
		{
			tokens: []string{"v1", "/", "{", "name", ".", "nested", ".", "nested2", "=", "a", "/", "b", "/", "*", "}", "/", "o", "/", "{", "another_name", "=", "a", "/", "b", "/", "*", "/", "c", "}", "/", "**", eof},
			want:   []segment{literal("v1"), variable{path: "name.nested.nested2", segments: []segment{literal("a"), literal("b"), wildcard{}}}, literal("o"), variable{path: "another_name", segments: []segment{literal("a"), literal("b"), wildcard{}, literal("c")}}, deepWildcard{}},
		},
	} {
		p := parser{tokens: spec.tokens}
		segs, err := p.topLevelSegments()
		assert.NoError(t, err, "parser{%q}.segments()", spec.tokens)
		assert.Equal(t, spec.want, segs, "parser{%q}.segments()", spec.tokens)
		assert.Empty(t, p.tokens, "p.tokens; spec.tokens=%q", spec.tokens)
	}
}

func TestParse(t *testing.T) {
	for _, spec := range []struct {
		input       string
		wantFields  []string
		wantOpCodes []int
		wantPool    []string
		wantVerb    string
	}{
		{input: "/v1/{name}:bla:baa", wantFields: []string{"name"}, wantPool: []string{"v1", "name"}, wantVerb: "bla:baa"},
		{input: "/v1/{name}:", wantFields: []string{"name"}, wantPool: []string{"v1", "name"}, wantVerb: ""},
		{input: "/v1/{name=segment/wi:th}", wantFields: []string{"name"}, wantPool: []string{"v1", "segment", "wi:th", "name"}, wantVerb: ""},
	} {
		f, err := Parse(spec.input)
		assert.NoError(t, err, "Parse(%q)", spec.input)
		tmpl := f.Compile()
		assert.Equal(t, spec.wantFields, tmpl.Fields, "Parse(%q).Fields", spec.input)
		assert.Equal(t, spec.wantPool, tmpl.Pool, "Parse(%q).Pool", spec.input)
		assert.Equal(t, spec.input, tmpl.Template, "Parse(%q).Template", spec.input)
		assert.Equal(t, spec.wantVerb, tmpl.Verb, "Parse(%q).Verb", spec.input)
	}
}

func TestParseError(t *testing.T) {
	for _, spec := range []struct {
		input     string
		wantError error
	}{
		{input: "v1/{name}", wantError: InvalidTemplateError{tmpl: "v1/{name}", msg: "no leading /"}},
	} {
		_, err := Parse(spec.input)
		assert.Error(t, err, "Parse(%q) unexpectedly did not fail", spec.input)
		assert.True(t, errors.Is(err, spec.wantError), "Error did not match expected error: got %v wanted %v", err, spec.wantError)
	}
}

func TestParseSegmentsWithErrors(t *testing.T) {
	for _, spec := range []struct {
		tokens []string
	}{
		{tokens: []string{"//", eof}},
		{tokens: []string{"a?b", eof}},
		{tokens: []string{"%", eof}},
		{tokens: []string{"%2", eof}},
		{tokens: []string{"a%2z", eof}},
		{tokens: []string{"{", "name", eof}},
		{tokens: []string{"{", "name", "=", eof}},
		{tokens: []string{"{", "name", "=", "*", eof}},
		{tokens: []string{"{", "name", ".", "}", eof}},
		{tokens: []string{"{", "name", ".", ".", "nested", "}", eof}},
		{tokens: []string{"{", "field-name", "}", eof}},
		{tokens: []string{"v1", "endpoint", eof}},
		{tokens: []string{"v1", "{", "name", "}", eof}},
	} {
		p := parser{tokens: spec.tokens}
		segs, err := p.topLevelSegments()
		assert.Error(t, err, "parser{%q}.segments() succeeded; want InvalidTemplateError; accepted %#v", spec.tokens, segs)
		if grpclog.V(1) {
			grpclog.Info(err)
		}
	}
}
