/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec\marshaler.go
 * @Description: 编解码接口定义
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package codec

import "io"

// Marshaler 定义字节序列与 gRPC 载荷之间的转换接口
type Marshaler interface {
	// Marshal 将 v 序列化为字节切片
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal 将字节切片反序列化到 v
	Unmarshal(data []byte, v interface{}) error
	// NewDecoder 返回从 r 读取的解码器
	NewDecoder(r io.Reader) Decoder
	// NewEncoder 返回向 w 写入的编码器
	NewEncoder(w io.Writer) Encoder
	// ContentType 返回此 marshaler 负责的 Content-Type
	ContentType(v interface{}) string
}

// Decoder 字节序列解码器
type Decoder interface {
	Decode(v interface{}) error
}

// Encoder 字节序列编码器
type Encoder interface {
	Encode(v interface{}) error
}

// DecoderFunc 将解码函数适配为 Decoder
type DecoderFunc func(v interface{}) error

// Decode 调用底层函数
func (f DecoderFunc) Decode(v interface{}) error { return f(v) }

// EncoderFunc 将编码函数适配为 Encoder
type EncoderFunc func(v interface{}) error

// Encode 调用底层函数
func (f EncoderFunc) Encode(v interface{}) error { return f(v) }

// Delimited 定义流式分隔符接口
type Delimited interface {
	Delimiter() []byte
}

// StreamContentType 定义流式 Content-Type 接口
type StreamContentType interface {
	StreamContentType(v interface{}) string
}
