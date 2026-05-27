/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 10:44:46
 * @FilePath: \grpc-runtime\pattern.go
 * @Description: HTTP 路径模式匹配，基于 opcode VM 实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kamalyes/grpc-runtime/utilities"
)

var (
	// ErrNotMatch HTTP 请求路径不匹配模式
	ErrNotMatch = errors.New("not match to the path pattern")
	// ErrInvalidPattern Pattern 定义无效
	ErrInvalidPattern = errors.New("invalid pattern")
)

type MalformedSequenceError string

func (e MalformedSequenceError) Error() string {
	return "malformed path escape " + strconv.Quote(string(e))
}

type op struct {
	code    utilities.OpCode
	operand int
}

// Pattern HTTP 请求路径模板模式
// 定义见 https://github.com/googleapis/googleapis/blob/master/google/api/http.proto
type Pattern struct {
	// ops 操作码序列
	ops []op
	// pool 常量池，按操作数索引
	pool []string
	// vars 此模式绑定的变量名列表
	vars []string
	// stacksize 栈最大深度
	stacksize int
	// tailLen 深度通配符后的固定段长度
	tailLen int
	// verb 路径模式的 VERB 部分，无 VERB 时为空
	verb string
}

// NewPattern 从定义值创建 Pattern
// ops 为操作码序列，pool 为常量池，verb 为模式的 VERB 部分
// version 当前必须为 1
// 定义无效时返回错误
func NewPattern(version int, ops []int, pool []string, verb string) (Pattern, error) {
	if version != 1 {
		logErrorf("unsupported version: %d", version)
		return Pattern{}, ErrInvalidPattern
	}

	l := len(ops)
	if l%2 != 0 {
		logErrorf("odd number of ops codes: %d", l)
		return Pattern{}, ErrInvalidPattern
	}

	var (
		typedOps        []op
		stack, maxstack int
		tailLen         int
		pushMSeen       bool
		vars            []string
	)
	for i := 0; i < l; i += 2 {
		op := op{code: utilities.OpCode(ops[i]), operand: ops[i+1]}
		switch op.code {
		case utilities.OpNop:
			continue
		case utilities.OpPush:
			if pushMSeen {
				tailLen++
			}
			stack++
		case utilities.OpPushM:
			if pushMSeen {
				logErrorf("pushM appears twice")
				return Pattern{}, ErrInvalidPattern
			}
			pushMSeen = true
			stack++
		case utilities.OpLitPush:
			if op.operand < 0 || len(pool) <= op.operand {
				logErrorf("negative literal index: %d", op.operand)
				return Pattern{}, ErrInvalidPattern
			}
			if pushMSeen {
				tailLen++
			}
			stack++
		case utilities.OpConcatN:
			if op.operand <= 0 {
				logErrorf("negative concat size: %d", op.operand)
				return Pattern{}, ErrInvalidPattern
			}
			stack -= op.operand
			if stack < 0 {
				logErrorf("stack underflow")
				return Pattern{}, ErrInvalidPattern
			}
			stack++
		case utilities.OpCapture:
			if op.operand < 0 || len(pool) <= op.operand {
				logErrorf("variable name index out of bound: %d", op.operand)
				return Pattern{}, ErrInvalidPattern
			}
			v := pool[op.operand]
			op.operand = len(vars)
			vars = append(vars, v)
			stack--
			if stack < 0 {
				logErrorf("stack underflow")
				return Pattern{}, ErrInvalidPattern
			}
		default:
			logErrorf("invalid opcode: %d", op.code)
			return Pattern{}, ErrInvalidPattern
		}

		if maxstack < stack {
			maxstack = stack
		}
		typedOps = append(typedOps, op)
	}
	return Pattern{
		ops:       typedOps,
		pool:      pool,
		vars:      vars,
		stacksize: maxstack,
		tailLen:   tailLen,
		verb:      verb,
	}, nil
}

// MustPattern 辅助函数，简化变量初始化时调用 NewPattern
func MustPattern(p Pattern, err error) Pattern {
	if err != nil {
		logFatalf("Pattern initialization failed: %v", err)
	}
	return p
}

// MatchAndEscape 检查组件是否匹配 Pattern
// 无 Pattern 匹配或匹配但包含畸形转义序列时返回错误
// 成功时返回字段路径到捕获值的映射
func (p Pattern) MatchAndEscape(components []string, verb string, unescapingMode UnescapingMode) (map[string]string, error) {
	if p.verb != verb {
		if p.verb != "" {
			return nil, ErrNotMatch
		}
		if len(components) == 0 {
			components = []string{":" + verb}
		} else {
			components = append([]string{}, components...)
			components[len(components)-1] += ":" + verb
		}
	}

	var pos int
	stack := make([]string, 0, p.stacksize)
	captured := make([]string, len(p.vars))
	l := len(components)
	for _, op := range p.ops {
		var err error

		switch op.code {
		case utilities.OpNop:
			continue
		case utilities.OpPush, utilities.OpLitPush:
			if pos >= l {
				return nil, ErrNotMatch
			}
			c := components[pos]
			if op.code == utilities.OpLitPush {
				if lit := p.pool[op.operand]; c != lit {
					return nil, ErrNotMatch
				}
			} else if op.code == utilities.OpPush {
				if c, err = unescape(c, unescapingMode, false); err != nil {
					return nil, err
				}
			}
			stack = append(stack, c)
			pos++
		case utilities.OpPushM:
			end := len(components)
			if end < pos+p.tailLen {
				return nil, ErrNotMatch
			}
			end -= p.tailLen
			c := strings.Join(components[pos:end], "/")
			if c, err = unescape(c, unescapingMode, true); err != nil {
				return nil, err
			}
			stack = append(stack, c)
			pos = end
		case utilities.OpConcatN:
			n := op.operand
			l := len(stack) - n
			stack = append(stack[:l], strings.Join(stack[l:], "/"))
		case utilities.OpCapture:
			n := len(stack) - 1
			captured[op.operand] = stack[n]
			stack = stack[:n]
		}
	}
	if pos < l {
		return nil, ErrNotMatch
	}
	bindings := make(map[string]string)
	for i, val := range captured {
		bindings[p.vars[i]] = val
	}
	return bindings, nil
}

// Match 检查组件是否匹配 Pattern（不执行逐段 unescape）
// 已废弃：请使用 MatchAndEscape
func (p Pattern) Match(components []string, verb string) (map[string]string, error) {
	return p.MatchAndEscape(components, verb, UnescapingModeDefault)
}

// Verb 返回 Pattern 的 verb 部分
func (p Pattern) Verb() string { return p.verb }

func (p Pattern) staticPath() (string, bool) {
	if len(p.vars) > 0 {
		return "", false
	}

	segments := make([]string, 0, len(p.ops))
	for _, op := range p.ops {
		switch op.code {
		case utilities.OpNop:
			continue
		case utilities.OpLitPush:
			if op.operand < 0 || len(p.pool) <= op.operand {
				return "", false
			}
			segments = append(segments, p.pool[op.operand])
		default:
			return "", false
		}
	}

	path := "/" + strings.Join(segments, "/")
	if p.verb != "" {
		path += ":" + p.verb
	}
	return path, true
}

func (p Pattern) String() string {
	var stack []string
	for _, op := range p.ops {
		switch op.code {
		case utilities.OpNop:
			continue
		case utilities.OpPush:
			stack = append(stack, "*")
		case utilities.OpLitPush:
			stack = append(stack, p.pool[op.operand])
		case utilities.OpPushM:
			stack = append(stack, "**")
		case utilities.OpConcatN:
			n := op.operand
			l := len(stack) - n
			stack = append(stack[:l], strings.Join(stack[l:], "/"))
		case utilities.OpCapture:
			n := len(stack) - 1
			stack[n] = fmt.Sprintf("{%s=%s}", p.vars[op.operand], stack[n])
		}
	}
	segs := strings.Join(stack, "/")
	if p.verb != "" {
		return fmt.Sprintf("/%s:%s", segs, p.verb)
	}
	return "/" + segs
}

// 以下代码改编自 Go 标准库，遵循其许可证
//
//	Copyright 2009 The Go Authors. All rights reserved.
//	Use of this source code is governed by a BSD-style
//	license that can be found in the LICENSE file.

// ishex 判断字节是否为有效的十六进制字符
func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func isRFC6570Reserved(c byte) bool {
	switch c {
	case '!', '#', '$', '&', '\'', '(', ')', '*',
		'+', ',', '/', ':', ';', '=', '?', '@', '[', ']':
		return true
	default:
		return false
	}
}

// unhex 将十六进制字符转换为数值
func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// shouldUnescapeWithMode 根据模式判断字符是否应 unescape
func shouldUnescapeWithMode(c byte, mode UnescapingMode) bool {
	switch mode {
	case UnescapingModeAllExceptReserved:
		if isRFC6570Reserved(c) {
			return false
		}
	case UnescapingModeAllExceptSlash:
		if c == '/' {
			return false
		}
	case UnescapingModeAllCharacters:
		return true
	}
	return true
}

// unescape 使用指定模式对路径字符串做 unescape
func unescape(s string, mode UnescapingMode, multisegment bool) (string, error) {
	// TODO(v3): remove UnescapingModeLegacy
	if mode == UnescapingModeLegacy {
		return s, nil
	}

	if !multisegment {
		mode = UnescapingModeAllCharacters
	}

	// Count %, check that they're well-formed.
	n := 0
	for i := 0; i < len(s); {
		if s[i] == '%' {
			n++
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				s = s[i:]
				if len(s) > 3 {
					s = s[:3]
				}

				return "", MalformedSequenceError(s)
			}
			i += 3
		} else {
			i++
		}
	}

	if n == 0 {
		return s, nil
	}

	var t strings.Builder
	t.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			c := unhex(s[i+1])<<4 | unhex(s[i+2])
			if shouldUnescapeWithMode(c, mode) {
				t.WriteByte(c)
				i += 2
				continue
			}
			fallthrough
		default:
			t.WriteByte(s[i])
		}
	}

	return t.String(), nil
}
