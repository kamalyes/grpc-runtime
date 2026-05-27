/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:02:33
 * @FilePath: \grpc-runtime\logging\logger.go
 * @Description: 日志接口和实现，支持 gRPC 日志和 go-logger 两种后端
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package logging

import (
	"sync/atomic"

	gologger "github.com/kamalyes/go-logger"
	"google.golang.org/grpc/grpclog"
)

// Logger grpc-runtime 使用的最小日志接口
// 镜像 go-logger 的 printf 风格方法，保持与具体日志实现解耦
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type loggerHolder struct {
	logger Logger
}

type grpcLogger struct{}

var runtimeLogger atomic.Pointer[loggerHolder]

func init() { SetLogger(grpcLogger{}) }

// SetLogger 替换 grpc-runtime 使用的日志器，传入 nil 恢复默认 gRPC 日志器
func SetLogger(logger Logger) {
	if logger == nil {
		logger = grpcLogger{}
	}
	runtimeLogger.Store(&loggerHolder{logger: logger})
}

// UseGoLogger 配置 grpc-runtime 通过 go-logger 写日志
func UseGoLogger(logger gologger.ILogger) {
	SetLogger(logger)
}

func activeLogger() Logger {
	holder := runtimeLogger.Load()
	if holder == nil || holder.logger == nil {
		return grpcLogger{}
	}
	return holder.logger
}

// Debugf 写入 debug 级别日志
func Debugf(format string, args ...interface{}) {
	activeLogger().Debugf(format, args...)
}

// Infof 写入 info 级别日志
func Infof(format string, args ...interface{}) {
	activeLogger().Infof(format, args...)
}

// Warnf 写入 warn 级别日志
func Warnf(format string, args ...interface{}) {
	activeLogger().Warnf(format, args...)
}

// Errorf 写入 error 级别日志
func Errorf(format string, args ...interface{}) {
	activeLogger().Errorf(format, args...)
}

// Fatalf 写入 fatal 级别日志
func Fatalf(format string, args ...interface{}) {
	activeLogger().Fatalf(format, args...)
}

func (grpcLogger) Debugf(format string, args ...interface{}) {
	if grpclog.V(1) {
		grpclog.Infof(format, args...)
	}
}

func (grpcLogger) Infof(format string, args ...interface{}) {
	grpclog.Infof(format, args...)
}

func (grpcLogger) Warnf(format string, args ...interface{}) {
	grpclog.Warningf(format, args...)
}

func (grpcLogger) Errorf(format string, args ...interface{}) {
	grpclog.Errorf(format, args...)
}

func (grpcLogger) Fatalf(format string, args ...interface{}) {
	grpclog.Fatalf(format, args...)
}
