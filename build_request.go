/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\build_request.go
 * @Description: 请求构建 pipeline，统一处理 body/path/query/fieldmask/validate
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kamalyes/grpc-runtime/utilities"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// BuildRequest 执行完整的请求构建 pipeline。
// 顺序：decode body -> apply path params -> apply query params -> apply field mask -> validate。
func BuildRequest(ctx context.Context, mux *ServeMux, r *http.Request, msg proto.Message, pathParams map[string]string, body BodyBinding, queryFilter QueryParamFilter) error {
	if err := decodeBody(mux, r, msg, body); err != nil {
		return err
	}
	if len(pathParams) > 0 {
		if err := applyPathParams(msg, pathParams); err != nil {
			return err
		}
	}
	if queryFilter != nil || r.URL.RawQuery != "" {
		if err := PopulateQueryParameters(msg, r.URL.Query(), queryFilter); err != nil {
			return err
		}
	}
	if r.URL.RawQuery != "" {
		if err := applyFieldMask(r, msg); err != nil {
			return err
		}
	}
	return ValidateRequest(ctx, mux, r, msg)
}

func decodeBody(mux *ServeMux, r *http.Request, msg proto.Message, body BodyBinding) error {
	if !body.HasBody || r.Body == nil {
		return nil
	}
	defer r.Body.Close()

	inbound, _ := MarshalerForRequest(mux, r)
	if body.FieldPath == "" {
		err := inbound.NewDecoder(r.Body).Decode(msg)
		if err == io.EOF {
			return nil
		}
		return err
	}

	target, err := mutableFieldMessage(msg.ProtoReflect(), strings.Split(body.FieldPath, "."))
	if err != nil {
		return err
	}
	err = inbound.NewDecoder(r.Body).Decode(target)
	if err == io.EOF {
		return nil
	}
	return err
}

func mutableFieldMessage(msg protoreflect.Message, fieldPath []string) (proto.Message, error) {
	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty body field path")
	}
	for i, name := range fieldPath {
		fd := fieldByName(msg.Descriptor().Fields(), name)
		if fd == nil {
			return nil, fmt.Errorf("body field %q not found in %q", name, msg.Descriptor().FullName())
		}
		if fd.Message() == nil || fd.IsList() || fd.IsMap() {
			return nil, fmt.Errorf("body field %q is not a singular message", name)
		}
		child := msg.Mutable(fd).Message()
		if i == len(fieldPath)-1 {
			return child.Interface(), nil
		}
		msg = child
	}
	return nil, fmt.Errorf("empty body field path")
}

func fieldByName(fields protoreflect.FieldDescriptors, name string) protoreflect.FieldDescriptor {
	fd := fields.ByTextName(name)
	if fd != nil {
		return fd
	}
	fd = fields.ByJSONName(name)
	if fd != nil {
		return fd
	}
	return fields.ByName(protoreflect.Name(name))
}

func applyPathParams(msg proto.Message, pathParams map[string]string) error {
	values := make(map[string][]string, len(pathParams))
	for k, v := range pathParams {
		values[k] = []string{v}
	}
	return PopulateQueryParameters(msg, values, utilities.NewDoubleArray(nil))
}

func applyFieldMask(_ *http.Request, _ proto.Message) error {
	return nil
}
