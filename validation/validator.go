/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\validation\validator.go
 * @Description: 请求校验接口定义和错误格式化
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package validation

// Validator 请求校验最小接口
// runtime 通过此接口注入校验能力，具体实现由上层（如 go-argus）提供
type Validator interface {
	// Struct 对已绑定的请求结构体执行校验
	Struct(v any) error
}

// ErrorFormatter 校验错误格式化函数
// 将校验错误转换为面向客户端的错误消息
type ErrorFormatter func(error) string

// NopValidator 空校验器，不做任何校验
type NopValidator struct{}

// Struct 总是返回 nil
func (NopValidator) Struct(any) error { return nil }
