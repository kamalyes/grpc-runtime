/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:20:32
 * @FilePath: \grpc-runtime\binding\fieldmask.go
 * @Description: FieldMask 处理，从 query 参数中提取 update_mask
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package binding

import (
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// FieldMaskFromRequest 从 HTTP 请求的 query 参数中提取 field mask
// 查找名为 "update_mask" 或 "field_mask" 的参数
func FieldMaskFromRequest(req *http.Request, msg proto.Message) (*fieldmaskpb.FieldMask, error) {
	values := req.URL.Query()
	var paths []string

	for key, vals := range values {
		if key == "update_mask" || key == "field_mask" {
			for _, v := range vals {
				paths = append(paths, strings.Split(v, ",")...)
			}
		}
	}

	if len(paths) == 0 {
		return nil, nil
	}

	// 验证路径是否有效
	if msg != nil {
		ref := msg.ProtoReflect()
		validPaths := make([]string, 0, len(paths))
		for _, path := range paths {
			if isValidPath(ref.Descriptor(), path) {
				validPaths = append(validPaths, path)
			}
		}
		paths = validPaths
	}

	return &fieldmaskpb.FieldMask{Paths: paths}, nil
}

// isValidPath 检查路径是否在消息描述符中有效
func isValidPath(desc protoreflect.MessageDescriptor, path string) bool {
	parts := strings.Split(path, ".")
	current := desc

	for _, part := range parts {
		fd := current.Fields().ByName(protoreflect.Name(part))
		if fd == nil {
			return false
		}
		if fd.Message() != nil {
			current = fd.Message()
		}
	}
	return true
}
