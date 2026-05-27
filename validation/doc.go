/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\validation\doc.go
 * @Description: validation 包提供请求校验的最小接口定义，不依赖具体校验实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

// Package validation 提供请求校验的最小接口定义，runtime 通过此接口注入校验能力
// 具体实现（如 go-argus）由上层注入，避免 runtime 直接依赖特定校验库
package validation
