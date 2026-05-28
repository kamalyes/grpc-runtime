/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:10:15
 * @FilePath: \grpc-runtime\protocgen\doc.go
 * @Description: protocgen 包提供 protoc 插件生成期共享能力
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

// Package protocgen 提供 protoc 插件生成期共享能力
// 供 grpc-gateway 与 openapiv2 插件复用
// 包含 Generator 接口定义，作为所有 protoc 插件生成器的统一抽象
package protocgen
