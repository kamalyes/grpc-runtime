/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:26:15
 * @FilePath: \grpc-runtime\protocgen\plugin\request.go
 * @Description: protoc 插件请求解析，从输入流读取并反序列化 CodeGeneratorRequest
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package plugin

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

// ParseRequest 从输入流解析 protoc CodeGeneratorRequest
func ParseRequest(r io.Reader) (*pluginpb.CodeGeneratorRequest, error) {
	input, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read code generator request: %w", err)
	}
	req := new(pluginpb.CodeGeneratorRequest)
	if err := proto.Unmarshal(input, req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal code generator request: %w", err)
	}
	return req, nil
}
