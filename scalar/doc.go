/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\scalar\doc.go
 * @Description: scalar 包提供标量类型转换函数，用于将字符串解析为 proto 字段值
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

// Package scalar 提供标量类型转换函数，将字符串解析为 proto 字段值
// 包括基本类型（int/float/bool/string/bytes）和 well-known 类型（Timestamp/Duration/Wrapper）
// 使用泛型 ParseSlice 统一所有 Slice 转换，消除重复代码
package scalar
