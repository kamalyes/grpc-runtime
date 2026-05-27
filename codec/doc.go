/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec\doc.go
 * @Description: codec 包提供编解码接口和实现，支持 JSON/Proto/HTTPBody 等格式
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

// Package codec 提供编解码接口和实现，支持 JSON/Proto/HTTPBody 等格式
// 接口定义在 marshaler.go，实现分别在各文件中
// Registry 提供 MIME 类型到 Marshaler 的并发安全映射
package codec
