/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 02:50:11
 * @FilePath: \grpc-runtime\response\forward_test.go
 * @Description: 响应转发测试
 *
 * Copyright (c) 2025 by kamalyes, All Rights Reserved.
 */

package response

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kamalyes/grpc-runtime/codec"
	"github.com/stretchr/testify/assert"
)

type mockMarshaler struct {
	contentType string
	data        []byte
	err         error
	delimiter   []byte
}

func (m *mockMarshaler) Marshal(v interface{}) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}
func (m *mockMarshaler) Unmarshal(data []byte, v interface{}) error { return nil }
func (m *mockMarshaler) NewDecoder(r io.Reader) codec.Decoder       { return nil }
func (m *mockMarshaler) NewEncoder(w io.Writer) codec.Encoder       { return nil }
func (m *mockMarshaler) ContentType(v interface{}) string           { return m.contentType }

// flushRecorder 包装 httptest.ResponseRecorder 并实现 http.Flusher
type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {}

func TestForwardResponseMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := &mockMarshaler{contentType: "application/json", data: []byte(`{"hello":"world"}`)}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		err := ForwardResponseMessage(m, w, r, "test")
		assert.NoError(t, err)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, `{"hello":"world"}`, w.Body.String())
	})

	t.Run("marshal_error", func(t *testing.T) {
		m := &mockMarshaler{contentType: "application/json", err: errors.New("marshal failed")}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		err := ForwardResponseMessage(m, w, r, "test")
		assert.Error(t, err)
	})
}

type mockDelimitedMarshaler struct {
	mockMarshaler
}

func (m *mockDelimitedMarshaler) Delimiter() []byte { return []byte("\n") }

func TestForwardResponseStream(t *testing.T) {
	t.Run("delimited_success", func(t *testing.T) {
		m := &mockDelimitedMarshaler{mockMarshaler: mockMarshaler{contentType: "application/json", data: []byte(`{"msg":1}`)}}
		w := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		callCount := 0
		recv := func() (interface{}, error) {
			callCount++
			if callCount > 2 {
				return nil, io.EOF
			}
			return "msg", nil
		}
		err := ForwardResponseStream(m, w, r, recv)
		assert.NoError(t, err)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, "chunked", w.Header().Get("Transfer-Encoding"))
	})

	t.Run("delimited_recv_error", func(t *testing.T) {
		m := &mockDelimitedMarshaler{mockMarshaler: mockMarshaler{contentType: "application/json", data: []byte(`{}`)}}
		w := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		recv := func() (interface{}, error) {
			return nil, errors.New("stream error")
		}
		err := ForwardResponseStream(m, w, r, recv)
		assert.Error(t, err)
	})

	t.Run("delimited_marshal_error", func(t *testing.T) {
		m := &mockDelimitedMarshaler{mockMarshaler: mockMarshaler{contentType: "application/json", err: errors.New("marshal failed")}}
		w := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		recv := func() (interface{}, error) { return "msg", nil }
		err := ForwardResponseStream(m, w, r, recv)
		assert.Error(t, err)
	})

	t.Run("non_delimited_success", func(t *testing.T) {
		m := &mockMarshaler{contentType: "application/json", data: []byte(`{"msg":1}`)}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		recv := func() (interface{}, error) { return "msg", nil }
		err := ForwardResponseStream(m, w, r, recv)
		assert.NoError(t, err)
	})

	t.Run("non_delimited_recv_error", func(t *testing.T) {
		m := &mockMarshaler{contentType: "application/json", data: []byte(`{}`)}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		recv := func() (interface{}, error) { return nil, errors.New("recv error") }
		err := ForwardResponseStream(m, w, r, recv)
		assert.Error(t, err)
	})
}
