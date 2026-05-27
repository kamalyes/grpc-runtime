/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\validation\validator_test.go
 * @Description: 请求校验接口单元测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNopValidator(t *testing.T) {
	v := NopValidator{}
	assert.NoError(t, v.Struct("anything"))
	assert.NoError(t, v.Struct(nil))
	assert.NoError(t, v.Struct(42))
}

func TestValidatorInterface(t *testing.T) {
	// 验证自定义实现满足 Validator 接口
	var v Validator = testValidator{err: errors.New("validation failed")}
	assert.Error(t, v.Struct("test"))

	v = testValidator{err: nil}
	assert.NoError(t, v.Struct("test"))
}

func TestErrorFormatter(t *testing.T) {
	var fn ErrorFormatter = func(err error) string {
		return "formatted: " + err.Error()
	}
	result := fn(errors.New("bad input"))
	assert.Equal(t, "formatted: bad input", result)
}

type testValidator struct {
	err error
}

func (v testValidator) Struct(any) error { return v.err }
