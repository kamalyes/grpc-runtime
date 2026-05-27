/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2025-05-28 00:00:00
 * @FilePath: \grpc-runtime\codec.go
 * @Description: 编解码兼容层，委托给 codec 子包
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package runtime

import (
	"net/http"

	"github.com/kamalyes/grpc-runtime/codec"
	"google.golang.org/protobuf/encoding/protojson"
)

// --- 类型别名，委托给 codec 子包 ---

type Marshaler = codec.Marshaler
type Decoder = codec.Decoder
type Encoder = codec.Encoder
type DecoderFunc = codec.DecoderFunc
type EncoderFunc = codec.EncoderFunc
type Delimited = codec.Delimited
type StreamContentType = codec.StreamContentType
type DecoderWrapper = codec.DecoderWrapper
type JSONBuiltin = codec.JSONBuiltin
type JSONPb = codec.JSONPb
type ProtoMarshaller = codec.ProtoMarshaller
type HTTPBodyMarshaler = codec.HTTPBodyMarshaler

// MIMEWildcard 通配 MIME 类型
const MIMEWildcard = codec.MIMEWildcard

// --- mux 依赖的注册表 ---

var defaultMarshaler = &HTTPBodyMarshaler{
	Marshaler: &JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	},
}

// marshalerRegistry 包装 codec.Registry，保持 mux 兼容
type marshalerRegistry struct {
	reg *codec.Registry
}

func makeMarshalerMIMERegistry() marshalerRegistry {
	return marshalerRegistry{reg: codec.NewRegistry()}
}

func (m marshalerRegistry) add(mime string, marshaler Marshaler) error {
	return m.reg.Add(mime, marshaler)
}

// MarshalerForRequest 根据 HTTP 请求查找 inbound/outbound marshaler
func MarshalerForRequest(mux *ServeMux, r *http.Request) (inbound Marshaler, outbound Marshaler) {
	return codec.MarshalerForRequest(mux.marshalers.reg, r)
}

// WithMarshalerOption 返回关联 MIME 类型到 Marshaler 的 ServeMuxOption
func WithMarshalerOption(mime string, marshaler Marshaler) ServeMuxOption {
	return func(mux *ServeMux) {
		if err := mux.marshalers.add(mime, marshaler); err != nil {
			panic(err)
		}
	}
}
