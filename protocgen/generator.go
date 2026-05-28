/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\protocgen\generator.go
 * @Description: protoc 插件生成器接口定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package protocgen

import (
	"github.com/kamalyes/grpc-runtime/protocgen/descriptor"
)

// Generator 定义 protoc 插件生成器的统一接口
// 所有 protoc 插件（如 grpc-gateway、openapiv2）均实现此接口
type Generator interface {
	// Generate 根据目标 proto 文件生成插件响应文件
	Generate(targets []*descriptor.File) ([]*descriptor.ResponseFile, error)
}
