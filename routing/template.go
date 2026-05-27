/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:27:38
 * @FilePath: \grpc-runtime\routing\template.go
 * @Description: 路径模板编译，将 template 字符串解析为结构化 segment 数组用于 trie 插入
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package routing

// segKind 路径段类型分类
type segKind int

const (
	segLiteral  segKind = iota // 静态字面量，如 "users"
	segParam                   // 单段参数捕获，如 {user_id}
	segWildcard                // 深度通配符捕获，如 ** 或 {*path}
)

// segment 编译后的路径模板段
type segment struct {
	kind     segKind
	literal  string // 仅 segLiteral 时设置
	paramIdx int    // 在 compiledTemplate.paramNames 中的索引
}

// compiledTemplate 路径模板编译结果
// 替代 opcode VM，用扁平 segment 数组实现更快的匹配
type compiledTemplate struct {
	segments   []segment
	paramNames []string
	verb       string // 非空表示模板含 :verb 后缀
}

// compileTemplate 将路径模板字符串解析为 compiledTemplate
// 模板格式遵循 google.api.http 规范：
//
//	/v1/users/{user_id}       → [literal("v1"), literal("users"), param(0)]
//	/v1/{name=*}             → [literal("v1"), param(0)]
//	/v1/{path=**}            → [literal("v1"), wildcard(0)]
//
// 这是简化编译器，只处理 trie 能精确表达的模板；复杂模板继续交给旧 Pattern MatchFunc 回退
func compileTemplate(template string) *compiledTemplate {
	ct := &compiledTemplate{}
	if template == "" || template == "/" {
		return ct
	}

	// 去掉前导 /
	rest := template
	if rest[0] == '/' {
		rest = rest[1:]
	}

	// 检查 verb 后缀（:verb）
	if idx := findVerbSeparator(rest); idx >= 0 {
		ct.verb = rest[idx+1:]
		rest = rest[:idx]
	}

	// 按 / 分割 segments
	parts := splitPath(rest)
	paramIdx := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		if part == "**" || part == "{**}" || part == "{*path}" {
			ct.segments = append(ct.segments, segment{kind: segWildcard, paramIdx: paramIdx})
			ct.paramNames = append(ct.paramNames, "path")
			paramIdx++
		} else if part[0] == '{' {
			name, expr, ok := parseParamPart(part)
			if !ok {
				return nil
			}
			switch expr {
			case "", "*":
				ct.segments = append(ct.segments, segment{kind: segParam, paramIdx: paramIdx})
			case "**":
				ct.segments = append(ct.segments, segment{kind: segWildcard, paramIdx: paramIdx})
			default:
				// 含 literal 约束的捕获（如 {name=projects/*}）需要旧 Pattern 保证精确语义
				return nil
			}
			ct.paramNames = append(ct.paramNames, name)
			paramIdx++
		} else {
			ct.segments = append(ct.segments, segment{kind: segLiteral, literal: part})
		}
	}

	return ct
}

// match 将请求路径与编译模板匹配，返回捕获的参数
func (ct *compiledTemplate) match(path string, opts MatchOptions) (*Params, error) {
	if path == "" || path[0] != '/' {
		return nil, errNotMatch
	}

	rest := path[1:]

	// 处理 verb
	if ct.verb != "" {
		idx := findVerbInPath(rest)
		if idx < 0 {
			return nil, errNotMatch
		}
		verbPart := rest[idx+1:]
		rest = rest[:idx]
		if verbPart != ct.verb {
			return nil, errNotMatch
		}
	} else if findVerbInPath(rest) >= 0 {
		// template 没有 verb 但 path 有 verb，不匹配
		return nil, errNotMatch
	}

	parts := splitPath(rest)
	params := NewParams(len(ct.paramNames))

	segIdx := 0
	partIdx := 0

	for segIdx < len(ct.segments) {
		seg := ct.segments[segIdx]

		switch seg.kind {
		case segLiteral:
			if partIdx >= len(parts) {
				return nil, errNotMatch
			}
			if parts[partIdx] != seg.literal {
				return nil, errNotMatch
			}
			partIdx++
			segIdx++

		case segParam:
			if partIdx >= len(parts) {
				return nil, errNotMatch
			}
			val, err := unescapeSegment(parts[partIdx], opts)
			if err != nil {
				return nil, err
			}
			params.Add(ct.paramNames[seg.paramIdx], val)
			partIdx++
			segIdx++

		case segWildcard:
			// wildcard 消费所有剩余 segments
			remaining := parts[partIdx:]
			if len(remaining) == 0 {
				// wildcard 至少匹配一个 segment
				return nil, errNotMatch
			}
			val, err := unescapeSegment(joinPath(remaining), opts)
			if err != nil {
				return nil, err
			}
			params.Add(ct.paramNames[seg.paramIdx], val)
			partIdx = len(parts)
			segIdx++

		default:
			return nil, errNotMatch
		}
	}

	// 所有 segments 处理完后，path 也必须消费完
	if partIdx < len(parts) {
		return nil, errNotMatch
	}

	return params, nil
}

// parseParamPart 从 {param} 或 {param=expr} 中提取参数名和表达式
func parseParamPart(part string) (name string, expr string, ok bool) {
	if len(part) < 2 || part[0] != '{' || part[len(part)-1] != '}' {
		return "", "", false
	}
	inner := part[1 : len(part)-1]
	if idx := indexOfEqual(inner); idx >= 0 {
		return inner[:idx], inner[idx+1:], true
	}
	return inner, "", true
}

// indexOfEqual 查找字符串中 = 的位置
func indexOfEqual(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return i
		}
	}
	return -1
}

// findVerbSeparator 在 template 字符串中找 :verb 位置，跳过 {} 内的冒号
func findVerbSeparator(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ':':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// findVerbInPath 在请求路径中找 :verb 位置
func findVerbInPath(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
		if s[i] == '/' {
			break
		}
	}
	return -1
}

// splitPath 按 / 分割路径，忽略空段
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	return parts
}

// joinPath 用 / 连接 segments
func joinPath(parts []string) string {
	result := make([]byte, 0, len(parts)*16)
	for i, p := range parts {
		if i > 0 {
			result = append(result, '/')
		}
		result = append(result, p...)
	}
	return string(result)
}

// unescapeSegment 对单个 path segment 做 unescape
func unescapeSegment(seg string, opts MatchOptions) (string, error) {
	if opts.UnescapingMode == 0 {
		// UnescapingModeLegacy: 不做 unescape
		return seg, nil
	}
	return unescapePath(seg, opts.UnescapingMode)
}
