/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\binding.go
 * @Description: 请求参数绑定兼容层，委托给 binding 子包
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kamalyes/grpc-runtime/utilities"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/durationpb"
	field_mask "google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// --- mux 依赖的 query 解析接口 ---

var valuesKeyRegexp = regexp.MustCompile(`^(.*)\[(.*)\]$`)

var currentQueryParser QueryParameterParser = &DefaultQueryParser{}

// QueryParameterParser 定义 query 参数解析接口
type QueryParameterParser interface {
	Parse(msg proto.Message, values url.Values, filter *utilities.DoubleArray) error
}

// PopulateQueryParameters 解析 query 参数到 msg
func PopulateQueryParameters(msg proto.Message, values url.Values, filter *utilities.DoubleArray) error {
	return currentQueryParser.Parse(msg, values, filter)
}

// DefaultQueryParser 默认 query 参数解析器
type DefaultQueryParser struct{}

func (*DefaultQueryParser) Parse(msg proto.Message, values url.Values, filter *utilities.DoubleArray) error {
	for key, vals := range values {
		if match := valuesKeyRegexp.FindStringSubmatch(key); len(match) == 3 {
			key = match[1]
			vals = append([]string{match[2]}, vals...)
		}

		msgValue := msg.ProtoReflect()
		fieldPath := normalizeFieldPath(msgValue, strings.Split(key, "."))
		if filter.HasCommonPrefix(fieldPath) {
			continue
		}
		if err := populateFieldValueFromPath(msgValue, fieldPath, vals); err != nil {
			return err
		}
	}
	return nil
}

// PopulateFieldFromPath 设置嵌套 Protobuf 结构中的值
func PopulateFieldFromPath(msg proto.Message, fieldPathString string, value string) error {
	fieldPath := strings.Split(fieldPathString, ".")
	return populateFieldValueFromPath(msg.ProtoReflect(), fieldPath, []string{value})
}

func normalizeFieldPath(msgValue protoreflect.Message, fieldPath []string) []string {
	newFieldPath := make([]string, 0, len(fieldPath))
	for i, fieldName := range fieldPath {
		fields := msgValue.Descriptor().Fields()
		fieldDesc := fields.ByTextName(fieldName)
		if fieldDesc == nil {
			fieldDesc = fields.ByJSONName(fieldName)
		}
		if fieldDesc == nil {
			return fieldPath
		}
		newFieldPath = append(newFieldPath, string(fieldDesc.Name()))
		if i == len(fieldPath)-1 {
			break
		}
		if fieldDesc.Message() == nil || fieldDesc.Cardinality() == protoreflect.Repeated {
			return fieldPath
		}
		msgValue = msgValue.Get(fieldDesc).Message()
	}
	return newFieldPath
}

func populateFieldValueFromPath(msgValue protoreflect.Message, fieldPath []string, values []string) error {
	if len(fieldPath) < 1 {
		return errors.New("no field path")
	}
	if len(values) < 1 {
		return errors.New("no value provided")
	}

	var fieldDescriptor protoreflect.FieldDescriptor
	for i, fieldName := range fieldPath {
		fields := msgValue.Descriptor().Fields()
		fieldDescriptor = fields.ByName(protoreflect.Name(fieldName))
		if fieldDescriptor == nil {
			fieldDescriptor = fields.ByJSONName(fieldName)
			if fieldDescriptor == nil {
				logInfof("field not found in %q: %q", msgValue.Descriptor().FullName(), strings.Join(fieldPath, "."))
				return nil
			}
		}

		if of := fieldDescriptor.ContainingOneof(); of != nil && !of.IsSynthetic() {
			if f := msgValue.WhichOneof(of); f != nil {
				if fieldDescriptor.Message() == nil || fieldDescriptor.FullName() != f.FullName() {
					return fmt.Errorf("field already set for oneof %q", of.FullName().Name())
				}
			}
		}

		if i == len(fieldPath)-1 {
			break
		}

		if fieldDescriptor.Message() == nil || fieldDescriptor.Cardinality() == protoreflect.Repeated {
			return fmt.Errorf("invalid path: %q is not a message", fieldName)
		}

		msgValue = msgValue.Mutable(fieldDescriptor).Message()
	}

	switch {
	case fieldDescriptor.IsList():
		return populateRepeatedField(fieldDescriptor, msgValue.Mutable(fieldDescriptor).List(), values)
	case fieldDescriptor.IsMap():
		return populateMapField(fieldDescriptor, msgValue.Mutable(fieldDescriptor).Map(), values)
	}

	if len(values) > 1 {
		return fmt.Errorf("too many values for field %q: %s", fieldDescriptor.FullName().Name(), strings.Join(values, ", "))
	}

	return populateField(fieldDescriptor, msgValue, values[0])
}

func populateField(fieldDescriptor protoreflect.FieldDescriptor, msgValue protoreflect.Message, value string) error {
	v, err := parseField(fieldDescriptor, value)
	if err != nil {
		return fmt.Errorf("parsing field %q: %w", fieldDescriptor.FullName().Name(), err)
	}
	msgValue.Set(fieldDescriptor, v)
	return nil
}

func populateRepeatedField(fieldDescriptor protoreflect.FieldDescriptor, list protoreflect.List, values []string) error {
	for _, value := range values {
		v, err := parseField(fieldDescriptor, value)
		if err != nil {
			return fmt.Errorf("parsing list %q: %w", fieldDescriptor.FullName().Name(), err)
		}
		list.Append(v)
	}
	return nil
}

func populateMapField(fieldDescriptor protoreflect.FieldDescriptor, mp protoreflect.Map, values []string) error {
	if len(values) != 2 {
		return fmt.Errorf("more than one value provided for key %q in map %q", values[0], fieldDescriptor.FullName())
	}
	key, err := parseField(fieldDescriptor.MapKey(), values[0])
	if err != nil {
		return fmt.Errorf("parsing map key %q: %w", fieldDescriptor.FullName().Name(), err)
	}
	value, err := parseField(fieldDescriptor.MapValue(), values[1])
	if err != nil {
		return fmt.Errorf("parsing map value %q: %w", fieldDescriptor.FullName().Name(), err)
	}
	mp.Set(key.MapKey(), value)
	return nil
}

func parseField(fieldDescriptor protoreflect.FieldDescriptor, value string) (protoreflect.Value, error) {
	switch fieldDescriptor.Kind() {
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(v), nil
	case protoreflect.EnumKind:
		enum, err := protoregistry.GlobalTypes.FindEnumByName(fieldDescriptor.Enum().FullName())
		if err != nil {
			if errors.Is(err, protoregistry.NotFound) {
				return protoreflect.Value{}, fmt.Errorf("enum %q is not registered", fieldDescriptor.Enum().FullName())
			}
			return protoreflect.Value{}, fmt.Errorf("failed to look up enum: %w", err)
		}
		v := enum.Descriptor().Values().ByName(protoreflect.Name(value))
		if v == nil {
			i, err := strconv.Atoi(value)
			if err != nil {
				return protoreflect.Value{}, fmt.Errorf("%q is not a valid value", value)
			}
			if v = enum.Descriptor().Values().ByNumber(protoreflect.EnumNumber(i)); v == nil {
				return protoreflect.Value{}, fmt.Errorf("%q is not a valid value", value)
			}
		}
		return protoreflect.ValueOfEnum(v.Number()), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(value), nil
	case protoreflect.BytesKind:
		v, err := Bytes(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBytes(v), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return parseMessage(fieldDescriptor.Message(), value)
	default:
		panic(fmt.Sprintf("unknown field kind: %v", fieldDescriptor.Kind()))
	}
}

func parseMessage(msgDescriptor protoreflect.MessageDescriptor, value string) (protoreflect.Value, error) {
	var msg proto.Message
	switch msgDescriptor.FullName() {
	case "google.protobuf.Timestamp":
		t, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		timestamp := timestamppb.New(t)
		if ok := timestamp.IsValid(); !ok {
			return protoreflect.Value{}, fmt.Errorf("%s before 0001-01-01", value)
		}
		msg = timestamp
	case "google.protobuf.Duration":
		d, err := time.ParseDuration(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = durationpb.New(d)
	case "google.protobuf.DoubleValue":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.Double(v)
	case "google.protobuf.FloatValue":
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.Float(float32(v))
	case "google.protobuf.Int64Value":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.Int64(v)
	case "google.protobuf.Int32Value":
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.Int32(int32(v))
	case "google.protobuf.UInt64Value":
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.UInt64(v)
	case "google.protobuf.UInt32Value":
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.UInt32(uint32(v))
	case "google.protobuf.BoolValue":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.Bool(v)
	case "google.protobuf.StringValue":
		msg = wrapperspb.String(value)
	case "google.protobuf.BytesValue":
		v, err := Bytes(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = wrapperspb.Bytes(v)
	case "google.protobuf.FieldMask":
		fm := &field_mask.FieldMask{}
		fm.Paths = append(fm.Paths, strings.Split(value, ",")...)
		msg = fm
	case "google.protobuf.Value":
		var v structpb.Value
		if err := protojson.Unmarshal([]byte(value), &v); err != nil {
			return protoreflect.Value{}, err
		}
		msg = &v
	case "google.protobuf.Struct":
		var v structpb.Struct
		if err := protojson.Unmarshal([]byte(value), &v); err != nil {
			return protoreflect.Value{}, err
		}
		msg = &v
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported message type: %q", string(msgDescriptor.FullName()))
	}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// --- FieldMask ---

func getFieldByName(fields protoreflect.FieldDescriptors, name string) protoreflect.FieldDescriptor {
	fd := fields.ByName(protoreflect.Name(name))
	if fd != nil {
		return fd
	}
	return fields.ByJSONName(name)
}

// FieldMaskFromRequestBody 从 JSON body 创建 FieldMask
func FieldMaskFromRequestBody(r io.Reader, msg proto.Message) (*field_mask.FieldMask, error) {
	fm := &field_mask.FieldMask{}
	var root interface{}

	if err := json.NewDecoder(r).Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return fm, nil
		}
		return nil, err
	}

	queue := []fieldMaskPathItem{{node: root, msg: msg.ProtoReflect()}}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		m, ok := item.node.(map[string]interface{})
		switch {
		case ok && len(m) > 0:
			for k, v := range m {
				if item.msg == nil {
					return nil, errors.New("JSON structure did not match request type")
				}

				fd := getFieldByName(item.msg.Descriptor().Fields(), k)
				if fd == nil {
					return nil, fmt.Errorf("could not find field %q in %q", k, item.msg.Descriptor().FullName())
				}

				if isDynamicProtoMessage(fd.Message()) {
					for _, p := range buildPathsBlindly(string(fd.FullName().Name()), v) {
						newPath := p
						if item.path != "" {
							newPath = item.path + "." + newPath
						}
						queue = append(queue, fieldMaskPathItem{path: newPath})
					}
					continue
				}

				if isProtobufAnyMessage(fd.Message()) && !fd.IsList() {
					_, hasTypeField := v.(map[string]interface{})["@type"]
					if hasTypeField {
						queue = append(queue, fieldMaskPathItem{path: k})
						continue
					} else {
						return nil, fmt.Errorf("could not find field @type in %q in message %q", k, item.msg.Descriptor().FullName())
					}
				}

				child := fieldMaskPathItem{node: v}
				if item.path == "" {
					child.path = string(fd.FullName().Name())
				} else {
					child.path = item.path + "." + string(fd.FullName().Name())
				}

				switch {
				case fd.IsList(), fd.IsMap():
					fm.Paths = append(fm.Paths, child.path)
				case fd.Message() != nil:
					child.msg = item.msg.Get(fd).Message()
					fallthrough
				default:
					queue = append(queue, child)
				}
			}
		case ok && len(m) == 0:
			fallthrough
		case len(item.path) > 0:
			fm.Paths = append(fm.Paths, item.path)
		}
	}

	sort.Strings(fm.Paths)
	return fm, nil
}

func isProtobufAnyMessage(md protoreflect.MessageDescriptor) bool {
	return md != nil && (md.FullName() == "google.protobuf.Any")
}

func isDynamicProtoMessage(md protoreflect.MessageDescriptor) bool {
	return md != nil && (md.FullName() == "google.protobuf.Struct" || md.FullName() == "google.protobuf.Value")
}

func buildPathsBlindly(name string, in interface{}) []string {
	m, ok := in.(map[string]interface{})
	if !ok {
		return []string{name}
	}

	var paths []string
	queue := []fieldMaskPathItem{{path: name, node: m}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		m, ok := cur.node.(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range m {
			if mi, ok := v.(map[string]interface{}); ok {
				queue = append(queue, fieldMaskPathItem{path: cur.path + "." + k, node: mi})
			} else {
				curPath := cur.path + "." + k
				paths = append(paths, curPath)
			}
		}
	}
	return paths
}

type fieldMaskPathItem struct {
	path string
	node interface{}
	msg  protoreflect.Message
}
