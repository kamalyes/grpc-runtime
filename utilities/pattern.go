/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:00:00
 * @FilePath: \grpc-runtime\utilities\pattern.go
 * @Description: 路径模式操作码定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package utilities

// OpCode 编译后路径模式的操作码
type OpCode int

// 操作码常量
const (
	// OpNop 无操作
	OpNop = OpCode(iota)
	// OpPush 将组件压入栈
	OpPush
	// OpLitPush 如果匹配字面量则将组件压入栈
	OpLitPush
	// OpPushM 拼接剩余组件并压入栈
	OpPushM
	// OpConcatN 从栈弹出 N 项，拼接后压回栈
	OpConcatN
	// OpCapture 从栈弹出一项并绑定到变量
	OpCapture
	// OpEnd 最小无效操作码
	OpEnd
)
