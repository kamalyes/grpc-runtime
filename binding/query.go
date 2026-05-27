/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:31:12
 * @FilePath: \grpc-runtime\binding\query.go
 * @Description: query 参数解析器，将 URL query 参数绑定到 proto 消息
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package binding

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kamalyes/grpc-runtime/scalar"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// QueryParser 将 URL query 参数绑定到 proto 消息
type QueryParser struct {
	// FieldNameMapper 字段名映射函数，默认使用 proto 字段名
	FieldNameMapper func(protoreflect.FieldDescriptor) string
}

// NewQueryParser 创建 QueryParser
func NewQueryParser() *QueryParser {
	return &QueryParser{
		FieldNameMapper: func(fd protoreflect.FieldDescriptor) string {
			return string(fd.Name())
		},
	}
}

// Parse 将 URL query 参数绑定到 msg
func (p *QueryParser) Parse(req *http.Request, msg proto.Message) error {
	values := req.URL.Query()
	return p.parseValues(values, msg)
}

// ParseMap 将 map 形式的参数绑定到 msg
func (p *QueryParser) ParseValues(values map[string][]string, msg proto.Message) error {
	return p.parseValues(values, msg)
}

func (p *QueryParser) parseValues(values map[string][]string, msg proto.Message) error {
	if len(values) == 0 {
		return nil
	}

	ref := msg.ProtoReflect()
	fields := ref.Descriptor().Fields()

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		fieldName := p.FieldNameMapper(fd)

		vals, ok := values[fieldName]
		if !ok || len(vals) == 0 {
			continue
		}

		if err := setField(ref, fd, vals); err != nil {
			return fmt.Errorf("field %q: %w", fieldName, err)
		}
	}

	return nil
}

// setField 根据字段类型设置值
func setField(ref protoreflect.Message, fd protoreflect.FieldDescriptor, vals []string) error {
	val := vals[0]

	if fd.IsMap() {
		return setMapField(ref, fd, vals)
	}

	if fd.IsList() {
		return setListField(ref, fd, vals)
	}

	v, err := scalarValue(fd, val)
	if err != nil {
		return err
	}

	ref.Set(fd, v)
	return nil
}

// setListField 设置列表字段
func setListField(ref protoreflect.Message, fd protoreflect.FieldDescriptor, vals []string) error {
	list := ref.Mutable(fd).List()
	for _, val := range vals {
		v, err := scalarValue(fd, val)
		if err != nil {
			return err
		}
		list.Append(v)
	}
	return nil
}

// setMapField 设置 map 字段
func setMapField(ref protoreflect.Message, fd protoreflect.FieldDescriptor, vals []string) error {
	m := ref.Mutable(fd).Map()
	keyFd := fd.MapKey()
	valFd := fd.MapValue()

	for _, val := range vals {
		parts := strings.SplitN(val, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("map value %q must be in key:value format", val)
		}
		kv, err := scalarValue(keyFd, parts[0])
		if err != nil {
			return err
		}
		vv, err := scalarValue(valFd, parts[1])
		if err != nil {
			return err
		}
		m.Set(kv.MapKey(), vv)
	}
	return nil
}

// scalarValue 将字符串转为 protoreflect.Value
func scalarValue(fd protoreflect.FieldDescriptor, val string) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(val), nil
	case protoreflect.BoolKind:
		b, err := scalar.Bool(val)
		return protoreflect.ValueOfBool(b), err
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		i, err := scalar.Int32(val)
		return protoreflect.ValueOfInt32(i), err
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		i, err := scalar.Int64(val)
		return protoreflect.ValueOfInt64(i), err
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		i, err := scalar.Uint32(val)
		return protoreflect.ValueOfUint32(i), err
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		i, err := scalar.Uint64(val)
		return protoreflect.ValueOfUint64(i), err
	case protoreflect.FloatKind:
		f, err := scalar.Float32(val)
		return protoreflect.ValueOfFloat32(f), err
	case protoreflect.DoubleKind:
		f, err := scalar.Float64(val)
		return protoreflect.ValueOfFloat64(f), err
	case protoreflect.BytesKind:
		b, err := scalar.Bytes(val)
		return protoreflect.ValueOfBytes(b), err
	case protoreflect.EnumKind:
		i, err := scalar.Int32(val)
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(i)), err
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported field kind: %v", fd.Kind())
	}
}
