/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:02:56
 * @FilePath: \grpc-runtime\logging\logger_test.go
 * @Description: 测试 logger
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */
package logging

import (
	"fmt"
	"testing"

	gologger "github.com/kamalyes/go-logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureLogger struct {
	debug []string
	info  []string
	warn  []string
	err   []string
	fatal []string
}

func (l *captureLogger) Debugf(format string, args ...interface{}) {
	l.debug = append(l.debug, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Infof(format string, args ...interface{}) {
	l.info = append(l.info, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Warnf(format string, args ...interface{}) {
	l.warn = append(l.warn, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Errorf(format string, args ...interface{}) {
	l.err = append(l.err, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Fatalf(format string, args ...interface{}) {
	l.fatal = append(l.fatal, fmt.Sprintf(format, args...))
}

func TestSetLogger(t *testing.T) {
	original := activeLogger()
	t.Cleanup(func() {
		SetLogger(original)
	})

	logger := &captureLogger{}
	SetLogger(logger)

	Debugf("debug %s", "message")
	Infof("info %s", "message")
	Warnf("warn %s", "message")
	Errorf("error %s", "message")

	assert.Equal(t, []string{"debug message"}, logger.debug)
	assert.Equal(t, []string{"info message"}, logger.info)
	assert.Equal(t, []string{"warn message"}, logger.warn)
	assert.Equal(t, []string{"error message"}, logger.err)
	assert.Empty(t, logger.fatal)
}

func TestSetLoggerNilRestoresDefault(t *testing.T) {
	original := activeLogger()
	t.Cleanup(func() {
		SetLogger(original)
	})

	SetLogger(&captureLogger{})
	SetLogger(nil)

	_, ok := activeLogger().(grpcLogger)
	require.True(t, ok)
}

func TestUseGoLogger(t *testing.T) {
	original := activeLogger()
	t.Cleanup(func() {
		SetLogger(original)
	})

	logger := gologger.NewEmptyLogger()
	UseGoLogger(logger)

	assert.Same(t, logger, activeLogger())
}
