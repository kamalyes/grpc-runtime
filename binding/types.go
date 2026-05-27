/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\binding\types.go
 * @Description: 生成器使用的绑定类型定义，隐藏 DoubleArray 等旧实现细节
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package binding

import (
	"strings"

	"github.com/kamalyes/grpc-runtime/utilities"
)

// BodyBinding 描述 HTTP body 与 proto request 字段的绑定关系
// 新生成代码使用此类型替代旧生成代码中直接拼 body 字段路径的方式
type BodyBinding struct {
	// FieldPath body 对应的 proto 字段路径，如 "user" 或空字符串表示整个消息就是 body
	FieldPath string
	// HasBody 是否有 HTTP body
	HasBody bool
}

// NoBody 返回无 HTTP body 的绑定描述
func NoBody() BodyBinding {
	return BodyBinding{}
}

// Body 返回指定字段路径的 HTTP body 绑定描述
func Body(fieldPath string) BodyBinding {
	return BodyBinding{FieldPath: fieldPath, HasBody: true}
}

// QueryFilter 类型别名，隐藏 utilities.DoubleArray 实现细节
// 新生成代码使用此类型替代旧生成代码中直接构造 DoubleArray 的方式
type QueryFilter = *utilities.DoubleArray

// NewQueryFilter 创建 query 过滤器
// fields 参数为 proto 字段路径列表，如 "user_id"、"name.nested"
// 内部将字段路径按 "." 分割后构建 DoubleArray
func NewQueryFilter(fields ...string) QueryFilter {
	seqs := make([][]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		seqs = append(seqs, strings.Split(field, "."))
	}
	return utilities.NewDoubleArray(seqs)
}
