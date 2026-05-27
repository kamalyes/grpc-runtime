/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2025-05-25 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 01:19:06
 * @FilePath: \grpc-runtime\response\forward.go
 * @Description: 响应转发，将 gRPC 响应写入 HTTP response writer
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package response

import (
	"io"
	"net/http"

	"github.com/kamalyes/grpc-runtime/codec"
)

// ForwardResponseMessage 将 proto 消息序列化后写入 HTTP 响应
func ForwardResponseMessage(marshaler codec.Marshaler, w http.ResponseWriter, req *http.Request, msg interface{}) error {
	contentType := marshaler.ContentType(msg)
	w.Header().Set("Content-Type", contentType)
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ForwardResponseStream 流式转发 gRPC 响应
func ForwardResponseStream(marshaler codec.Marshaler, w http.ResponseWriter, req *http.Request, recv func() (interface{}, error)) error {
	if d, ok := marshaler.(codec.Delimited); ok {
		w.Header().Set("Content-Type", marshaler.ContentType(nil))
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher := w.(http.Flusher)
		for {
			msg, err := recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			buf, err := marshaler.Marshal(msg)
			if err != nil {
				return err
			}
			if _, err := w.Write(buf); err != nil {
				return err
			}
			if _, err := w.Write(d.Delimiter()); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
	msg, err := recv()
	if err != nil {
		return err
	}
	return ForwardResponseMessage(marshaler, w, req, msg)
}
