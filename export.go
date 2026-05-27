/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:00:00
 * @FilePath: \grpc-runtime\export.go
 * @Description: 日志接口导出，委托给 logging 子包
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import "github.com/kamalyes/grpc-runtime/logging"

// Logger 最小日志接口，从 logging 模块重新导出
// 旧代码可继续使用根 runtime 包，新代码可直接导入 logging 包
type Logger = logging.Logger

// SetLogger 替换 grpc-runtime 使用的日志器，传入 nil 恢复默认 gRPC 日志器
var SetLogger = logging.SetLogger

// UseGoLogger 配置 grpc-runtime 通过 go-logger 写日志
var UseGoLogger = logging.UseGoLogger

func logDebugf(format string, args ...interface{}) {
	logging.Debugf(format, args...)
}

func logInfof(format string, args ...interface{}) {
	logging.Infof(format, args...)
}

func logWarnf(format string, args ...interface{}) {
	logging.Warnf(format, args...)
}

func logErrorf(format string, args ...interface{}) {
	logging.Errorf(format, args...)
}

func logFatalf(format string, args ...interface{}) {
	logging.Fatalf(format, args...)
}
