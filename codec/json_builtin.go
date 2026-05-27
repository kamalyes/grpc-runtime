/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec\json_builtin.go
 * @Description: 标准 encoding/json 的 Marshaler 实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package codec

import (
	"encoding/json"
	"io"
)

// JSONBuiltin 使用标准 encoding/json 的 Marshaler
// 不支持 proto 高级特性（map/oneof 等），但简单消息性能更好
type JSONBuiltin struct{}

// ContentType 返回 "application/json"
func (*JSONBuiltin) ContentType(_ interface{}) string {
	return "application/json"
}

// Marshal 将 v 序列化为 JSON
func (j *JSONBuiltin) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalIndent 带缩进的 JSON 序列化
func (j *JSONBuiltin) MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// Unmarshal 从 JSON 反序列化到 v
func (j *JSONBuiltin) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// NewDecoder 返回 JSON 解码器
func (j *JSONBuiltin) NewDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}

// NewEncoder 返回 JSON 编码器
func (j *JSONBuiltin) NewEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

// Delimiter 返回换行分隔符
func (j *JSONBuiltin) Delimiter() []byte {
	return []byte("\n")
}
