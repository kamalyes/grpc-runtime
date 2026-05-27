/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec\jsonpb.go
 * @Description: protojson 的 Marshaler 实现，支持 proto 全部特性
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package codec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// 默认 marshal/unmarshal 选项
var (
	defaultMarshalOptions = protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	defaultUnmarshalOptions = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
)

// JSONPb 使用 protojson 的 Marshaler，支持 proto 全部特性
type JSONPb struct {
	protojson.MarshalOptions
	protojson.UnmarshalOptions
}

// ContentType 返回 "application/json"
func (*JSONPb) ContentType(_ interface{}) string {
	return "application/json"
}

// Marshal 将 v 序列化为 JSON
func (j *JSONPb) Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := j.marshalTo(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (j *JSONPb) marshalTo(w io.Writer, v interface{}) error {
	p, ok := v.(proto.Message)
	if !ok {
		buf, err := j.marshalNonProtoField(v)
		if err != nil {
			return err
		}
		if j.Indent != "" {
			b := &bytes.Buffer{}
			if err := json.Indent(b, buf, "", j.Indent); err != nil {
				return err
			}
			buf = b.Bytes()
		}
		_, err = w.Write(buf)
		return err
	}

	b, err := j.MarshalOptions.Marshal(p)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

var protoMessageType = reflect.TypeOf((*proto.Message)(nil)).Elem()

// marshalNonProto 序列化非 proto 消息字段
func (j *JSONPb) marshalNonProtoField(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []byte("null"), nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Slice {
		if rv.IsNil() {
			if j.EmitUnpopulated {
				return []byte("[]"), nil
			}
			return []byte("null"), nil
		}

		if rv.Type().Elem().Implements(protoMessageType) {
			var buf bytes.Buffer
			if err := buf.WriteByte('['); err != nil {
				return nil, err
			}
			for i := 0; i < rv.Len(); i++ {
				if i != 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, err
					}
				}
				if err := j.marshalTo(&buf, rv.Index(i).Interface().(proto.Message)); err != nil {
					return nil, err
				}
			}
			if err := buf.WriteByte(']'); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}

		if rv.Type().Elem().Implements(typeProtoEnum) {
			var buf bytes.Buffer
			if err := buf.WriteByte('['); err != nil {
				return nil, err
			}
			for i := 0; i < rv.Len(); i++ {
				if i != 0 {
					if err := buf.WriteByte(','); err != nil {
						return nil, err
					}
				}
				var err error
				if j.UseEnumNumbers {
					_, err = buf.WriteString(strconv.FormatInt(rv.Index(i).Int(), 10))
				} else {
					_, err = buf.WriteString("\"" + rv.Index(i).Interface().(protoEnum).String() + "\"")
				}
				if err != nil {
					return nil, err
				}
			}
			if err := buf.WriteByte(']'); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}

	if rv.Kind() == reflect.Map {
		m := make(map[string]*json.RawMessage)
		for _, k := range rv.MapKeys() {
			buf, err := j.Marshal(rv.MapIndex(k).Interface())
			if err != nil {
				return nil, err
			}
			m[fmt.Sprintf("%v", k.Interface())] = (*json.RawMessage)(&buf)
		}
		return json.Marshal(m)
	}
	if enum, ok := rv.Interface().(protoEnum); ok && !j.UseEnumNumbers {
		return json.Marshal(enum.String())
	}
	return json.Marshal(rv.Interface())
}

// Unmarshal 从 JSON 反序列化到 v
func (j *JSONPb) Unmarshal(data []byte, v interface{}) error {
	return unmarshalJSONPb(data, j.UnmarshalOptions, v)
}

// NewDecoder 返回 JSON 解码器
func (j *JSONPb) NewDecoder(r io.Reader) Decoder {
	d := json.NewDecoder(r)
	return DecoderWrapper{
		Decoder:          d,
		UnmarshalOptions: j.UnmarshalOptions,
	}
}

// DecoderWrapper 包装 json.Decoder 以支持 proto
type DecoderWrapper struct {
	*json.Decoder
	protojson.UnmarshalOptions
}

// Decode 包装 json.Decoder.Decode 以支持 proto
func (d DecoderWrapper) Decode(v interface{}) error {
	return decodeJSONPb(d.Decoder, d.UnmarshalOptions, v)
}

// NewEncoder 返回 JSON 编码器
func (j *JSONPb) NewEncoder(w io.Writer) Encoder {
	return EncoderFunc(func(v interface{}) error {
		if err := j.marshalTo(w, v); err != nil {
			return err
		}
		_, err := w.Write(j.Delimiter())
		return err
	})
}

// Delimiter 返回换行分隔符
func (j *JSONPb) Delimiter() []byte {
	return []byte("\n")
}

type protoEnum interface {
	fmt.Stringer
	EnumDescriptor() ([]byte, []int)
}

var typeProtoEnum = reflect.TypeOf((*protoEnum)(nil)).Elem()

func unmarshalJSONPb(data []byte, unmarshaler protojson.UnmarshalOptions, v interface{}) error {
	d := json.NewDecoder(bytes.NewReader(data))
	return decodeJSONPb(d, unmarshaler, v)
}

func decodeJSONPb(d *json.Decoder, unmarshaler protojson.UnmarshalOptions, v interface{}) error {
	p, ok := v.(proto.Message)
	if !ok {
		return decodeNonProtoField(d, unmarshaler, v)
	}

	var b json.RawMessage
	if err := d.Decode(&b); err != nil {
		return err
	}

	return unmarshaler.Unmarshal([]byte(b), p)
}

func decodeNonProtoField(d *json.Decoder, unmarshaler protojson.UnmarshalOptions, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%T is not a pointer", v)
	}
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		if rv.Type().ConvertibleTo(protoMessageType) {
			var b json.RawMessage
			if err := d.Decode(&b); err != nil {
				return err
			}
			return unmarshaler.Unmarshal([]byte(b), rv.Interface().(proto.Message))
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Map {
		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rv.Type()))
		}
		conv, ok := convFromType[rv.Type().Key().Kind()]
		if !ok {
			return fmt.Errorf("unsupported type of map field key: %v", rv.Type().Key())
		}

		m := make(map[string]*json.RawMessage)
		if err := d.Decode(&m); err != nil {
			return err
		}
		for k, v := range m {
			result := conv.Call([]reflect.Value{reflect.ValueOf(k)})
			if err := result[1].Interface(); err != nil {
				return err.(error)
			}
			bk := result[0]
			bv := reflect.New(rv.Type().Elem())
			if v == nil {
				null := json.RawMessage("null")
				v = &null
			}
			if err := unmarshalJSONPb([]byte(*v), unmarshaler, bv.Interface()); err != nil {
				return err
			}
			rv.SetMapIndex(bk, bv.Elem())
		}
		return nil
	}
	if rv.Kind() == reflect.Slice {
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			var sl []byte
			if err := d.Decode(&sl); err != nil {
				return err
			}
			if sl != nil {
				rv.SetBytes(sl)
			}
			return nil
		}

		var sl []json.RawMessage
		if err := d.Decode(&sl); err != nil {
			return err
		}
		if sl != nil {
			rv.Set(reflect.MakeSlice(rv.Type(), 0, 0))
		}
		for _, item := range sl {
			bv := reflect.New(rv.Type().Elem())
			if err := unmarshalJSONPb([]byte(item), unmarshaler, bv.Interface()); err != nil {
				return err
			}
			rv.Set(reflect.Append(rv, bv.Elem()))
		}
		return nil
	}
	if _, ok := rv.Interface().(protoEnum); ok {
		var repr interface{}
		if err := d.Decode(&repr); err != nil {
			return err
		}
		switch v := repr.(type) {
		case string:
			return fmt.Errorf("unmarshaling of symbolic enum %q not supported: %T", repr, rv.Interface())
		case float64:
			rv.Set(reflect.ValueOf(int32(v)).Convert(rv.Type()))
			return nil
		default:
			return fmt.Errorf("cannot assign %#v into Go type %T", repr, rv.Interface())
		}
	}
	return d.Decode(v)
}

// convFromType map key 类型到转换函数的映射
var convFromType = map[reflect.Kind]reflect.Value{
	reflect.String:  reflect.ValueOf(scalarString),
	reflect.Bool:    reflect.ValueOf(scalarBool),
	reflect.Float64: reflect.ValueOf(scalarFloat64),
	reflect.Float32: reflect.ValueOf(scalarFloat32),
	reflect.Int64:   reflect.ValueOf(scalarInt64),
	reflect.Int32:   reflect.ValueOf(scalarInt32),
	reflect.Uint64:  reflect.ValueOf(scalarUint64),
	reflect.Uint32:  reflect.ValueOf(scalarUint32),
	reflect.Slice:   reflect.ValueOf(scalarBytes),
}

// 这些函数保持与根包 convert.go 的兼容签名
func scalarString(val string) (string, error)   { return val, nil }
func scalarBool(val string) (bool, error)       { return strconv.ParseBool(val) }
func scalarFloat64(val string) (float64, error) { return strconv.ParseFloat(val, 64) }
func scalarFloat32(val string) (float32, error) {
	f, err := strconv.ParseFloat(val, 32)
	return float32(f), err
}
func scalarInt64(val string) (int64, error) { return strconv.ParseInt(val, 0, 64) }
func scalarInt32(val string) (int32, error) {
	i, err := strconv.ParseInt(val, 0, 32)
	return int32(i), err
}
func scalarUint64(val string) (uint64, error) { return strconv.ParseUint(val, 0, 64) }
func scalarUint32(val string) (uint32, error) {
	i, err := strconv.ParseUint(val, 0, 32)
	return uint32(i), err
}
func scalarBytes(val string) ([]byte, error) { return []byte(val), nil }
