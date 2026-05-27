/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec\proto.go
 * @Description: protobuf 二进制 Marshaler 实现
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package codec

import (
	"errors"
	"io"

	"google.golang.org/protobuf/proto"
)

// ProtoMarshaller 使用 protobuf 二进制格式的 Marshaler
type ProtoMarshaller struct{}

// ContentType 返回 "application/octet-stream"
func (*ProtoMarshaller) ContentType(_ interface{}) string {
	return "application/octet-stream"
}

// Marshal 将 proto.Message 序列化为二进制
func (*ProtoMarshaller) Marshal(value interface{}) ([]byte, error) {
	message, ok := value.(proto.Message)
	if !ok {
		return nil, errors.New("unable to marshal non proto field")
	}
	return proto.Marshal(message)
}

// Unmarshal 从二进制反序列化到 proto.Message
func (*ProtoMarshaller) Unmarshal(data []byte, value interface{}) error {
	message, ok := value.(proto.Message)
	if !ok {
		return errors.New("unable to unmarshal non proto field")
	}
	return proto.Unmarshal(data, message)
}

// NewDecoder 返回 protobuf 解码器
func (m *ProtoMarshaller) NewDecoder(reader io.Reader) Decoder {
	return DecoderFunc(func(value interface{}) error {
		buffer, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		return m.Unmarshal(buffer, value)
	})
}

// NewEncoder 返回 protobuf 编码器
func (m *ProtoMarshaller) NewEncoder(writer io.Writer) Encoder {
	return EncoderFunc(func(value interface{}) error {
		buffer, err := m.Marshal(value)
		if err != nil {
			return err
		}
		_, err = writer.Write(buffer)
		return err
	})
}
