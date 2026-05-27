/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-27 22:38:58
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:22:11
 * @FilePath: \apex\grpc-runtime\marshal_httpbodyproto_test.go
 * @Description: HTTP Body协议序列化测试
 */
package runtime

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestHTTPBodyContentType(t *testing.T) {
	m := HTTPBodyMarshaler{
		Marshaler: &JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
		},
	}
	expected := "CustomContentType"
	message := &httpbody.HttpBody{
		ContentType: expected,
	}
	res := m.ContentType(nil)
	assert.Equal(t, "application/json", res)
	res = m.ContentType(message)
	assert.Equal(t, expected, res)
}

func TestHTTPBodyMarshal(t *testing.T) {
	m := HTTPBodyMarshaler{
		Marshaler: &JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
		},
	}
	expected := []byte("Some test")
	message := &httpbody.HttpBody{
		Data: expected,
	}
	res, err := m.Marshal(message)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(res, expected), "got %q, want %q", res, expected)
}
