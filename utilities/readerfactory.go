/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:00:00
 * @FilePath: \grpc-runtime\utilities\readerfactory.go
 * @Description: IO 读取器工厂，支持从同一数据源重复创建读取器
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package utilities

import (
	"bytes"
	"io"
)

// IOReaderFactory 接收 io.Reader 并返回一个工厂函数
// 每次调用工厂函数都返回一个从数据起始位置开始的新的读取器
func IOReaderFactory(r io.Reader) (func() io.Reader, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return func() io.Reader {
		return bytes.NewReader(b)
	}, nil
}
