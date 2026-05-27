package runtime

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/kamalyes/grpc-runtime/testpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fakeResponseBodyWrapper struct {
	proto.Message
}

// XXX_ResponseBody returns id of SimpleMessage
func (r fakeResponseBodyWrapper) XXX_ResponseBody() interface{} {
	resp := r.Message.(*testpb.SimpleMessage)
	return resp.Id
}

func TestForwardResponseStream(t *testing.T) {
	type msg struct {
		pb  proto.Message
		err error
	}
	tests := []struct {
		name         string
		msgs         []msg
		statusCode   int
		responseBody bool
	}{
		{name: "encoding", msgs: []msg{{&testpb.SimpleMessage{Id: "One"}, nil}, {&testpb.SimpleMessage{Id: "Two"}, nil}}, statusCode: http.StatusOK},
		{name: "empty", statusCode: http.StatusOK},
		{name: "error", msgs: []msg{{nil, status.Errorf(codes.OutOfRange, "400")}}, statusCode: http.StatusBadRequest},
		{name: "stream_error", msgs: []msg{{&testpb.SimpleMessage{Id: "One"}, nil}, {nil, status.Errorf(codes.OutOfRange, "400")}}, statusCode: http.StatusOK},
		{name: "response body stream case", msgs: []msg{{fakeResponseBodyWrapper{&testpb.SimpleMessage{Id: "One"}}, nil}, {fakeResponseBodyWrapper{&testpb.SimpleMessage{Id: "Two"}}, nil}}, responseBody: true, statusCode: http.StatusOK},
		{name: "response body stream error case", msgs: []msg{{fakeResponseBodyWrapper{&testpb.SimpleMessage{Id: "One"}}, nil}, {nil, status.Errorf(codes.OutOfRange, "400")}}, responseBody: true, statusCode: http.StatusOK},
	}

	newTestRecv := func(t *testing.T, msgs []msg) func() (proto.Message, error) {
		var count int
		return func() (proto.Message, error) {
			if count == len(msgs) {
				return nil, io.EOF
			} else if count > len(msgs) {
				assert.Failf(t, "recv called too many times", "recv() called %d times for %d messages", count, len(msgs))
			}
			count++
			msg := msgs[count-1]
			return msg.pb, msg.err
		}
	}
	ctx := NewServerMetadataContext(context.Background(), ServerMetadata{})
	marshaler := &JSONPb{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recv := newTestRecv(t, tt.msgs)
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			resp := httptest.NewRecorder()

			ForwardResponseStream(ctx, NewServeMux(), marshaler, resp, req, recv)

			w := resp.Result()
			assert.Equal(t, tt.statusCode, w.StatusCode, "StatusCode mismatch")
			assert.Equal(t, "chunked", w.Header.Get("Transfer-Encoding"), "ForwardResponseStream missing header chunked")
			body, err := io.ReadAll(w.Body)
			assert.NoError(t, err, "Failed to read response body")
			w.Body.Close()
			if len(body) > 0 {
				assert.Equal(t, "application/json", w.Header.Get("Content-Type"), "Content-Type mismatch")
			}

			var want []byte
			for i, msg := range tt.msgs {
				if msg.err != nil {
					if i == 0 {
						t.Skip("checking error encodings")
					}
					delimiter := marshaler.Delimiter()
					st := status.Convert(msg.err)
					b, err := marshaler.Marshal(map[string]proto.Message{"error": st.Proto()})
					assert.NoError(t, err, "marshaler.Marshal() failed")
					errBytes := body[len(want):]
					assert.Equal(t, string(b)+string(delimiter), string(errBytes), "ForwardResponseStream error mismatch")
					return
				}

				var b []byte
				if tt.responseBody {
					rb, ok := msg.pb.(fakeResponseBodyWrapper)
					assert.True(t, ok, "stream responseBody failed")
					b, err = marshaler.Marshal(map[string]interface{}{"result": rb.XXX_ResponseBody()})
				} else {
					b, err = marshaler.Marshal(map[string]interface{}{"result": msg.pb})
				}
				assert.NoError(t, err, "marshaler.Marshal() failed")
				want = append(want, b...)
				want = append(want, marshaler.Delimiter()...)
			}

			assert.Equal(t, string(want), string(body), "ForwardResponseStream body mismatch")
		})
	}
}

// CustomMarshaler 不实现 delimited 接口的自定义序列化器
type CustomMarshaler struct {
	m *JSONPb
}

func (c *CustomMarshaler) Marshal(v interface{}) ([]byte, error)      { return c.m.Marshal(v) }
func (c *CustomMarshaler) Unmarshal(data []byte, v interface{}) error { return c.m.Unmarshal(data, v) }
func (c *CustomMarshaler) NewDecoder(r io.Reader) Decoder             { return c.m.NewDecoder(r) }
func (c *CustomMarshaler) NewEncoder(w io.Writer) Encoder             { return c.m.NewEncoder(w) }
func (c *CustomMarshaler) ContentType(v interface{}) string           { return "Custom-Content-Type" }

// marshalerStreamContentType 实现了自定义 StreamContentType 的 Marshaler
type marshalerStreamContentType struct {
	Marshaler
	CustomStreamContentType string
}

func (m marshalerStreamContentType) StreamContentType(interface{}) string {
	return m.CustomStreamContentType
}

func TestForwardResponseStreamCustomMarshaler(t *testing.T) {
	type msg struct {
		pb  proto.Message
		err error
	}
	marshaler := &CustomMarshaler{&JSONPb{}}

	tests := []struct {
		name            string
		marshaler       Marshaler
		msgs            []msg
		statusCode      int
		wantContentType string
	}{
		{name: "encoding", marshaler: marshaler, msgs: []msg{{&testpb.SimpleMessage{Id: "One"}, nil}, {&testpb.SimpleMessage{Id: "Two"}, nil}}, statusCode: http.StatusOK, wantContentType: "Custom-Content-Type"},
		{name: "empty", marshaler: marshaler, statusCode: http.StatusOK},
		{name: "error", marshaler: marshaler, msgs: []msg{{nil, status.Errorf(codes.OutOfRange, "400")}}, statusCode: http.StatusBadRequest, wantContentType: "Custom-Content-Type"},
		{name: "stream_error", marshaler: marshaler, msgs: []msg{{&testpb.SimpleMessage{Id: "One"}, nil}, {nil, status.Errorf(codes.OutOfRange, "400")}}, statusCode: http.StatusOK, wantContentType: "Custom-Content-Type"},
		{name: "stream_content_type", marshaler: marshalerStreamContentType{Marshaler: marshaler, CustomStreamContentType: "Stream-Content-Type"}, msgs: []msg{{&testpb.SimpleMessage{Id: "One"}, nil}}, statusCode: http.StatusOK, wantContentType: "Stream-Content-Type"},
	}

	newTestRecv := func(t *testing.T, msgs []msg) func() (proto.Message, error) {
		var count int
		return func() (proto.Message, error) {
			if count == len(msgs) {
				return nil, io.EOF
			} else if count > len(msgs) {
				assert.Failf(t, "recv called too many times", "recv() called %d times for %d messages", count, len(msgs))
			}
			count++
			msg := msgs[count-1]
			return msg.pb, msg.err
		}
	}
	ctx := NewServerMetadataContext(context.Background(), ServerMetadata{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recv := newTestRecv(t, tt.msgs)
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			resp := httptest.NewRecorder()

			ForwardResponseStream(ctx, NewServeMux(), tt.marshaler, resp, req, recv)

			w := resp.Result()
			assert.Equal(t, tt.statusCode, w.StatusCode, "StatusCode mismatch")
			assert.Equal(t, "chunked", w.Header.Get("Transfer-Encoding"), "ForwardResponseStream missing header chunked")
			body, err := io.ReadAll(w.Body)
			assert.NoError(t, err, "Failed to read response body")
			w.Body.Close()
			assert.Equal(t, tt.wantContentType, w.Header.Get("Content-Type"), "Content-Type mismatch")

			var want []byte
			for _, msg := range tt.msgs {
				if msg.err != nil {
					t.Skip("checking error encodings")
				}
				b, err := tt.marshaler.Marshal(map[string]proto.Message{"result": msg.pb})
				assert.NoError(t, err, "marshaler.Marshal() failed")
				want = append(want, b...)
				want = append(want, "\n"...)
			}

			assert.Equal(t, string(want), string(body), "ForwardResponseStream body mismatch")
		})
	}
}

func TestForwardResponseMessage(t *testing.T) {
	msg := &testpb.SimpleMessage{Id: "One"}
	tests := []struct {
		name              string
		marshaler         Marshaler
		contentType       string
		frw               ForwardResponseRewriter
		getWantedResponse func(msg any) ([]byte, error)
	}{
		{name: "standard marshaler", marshaler: &JSONPb{}, contentType: "application/json"},
		{name: "httpbody marshaler", marshaler: &HTTPBodyMarshaler{&JSONPb{}}, contentType: "application/json"},
		{name: "custom marshaler", marshaler: &CustomMarshaler{&JSONPb{}}, contentType: "Custom-Content-Type"},
		{
			name:        "custom forward response rewriter",
			marshaler:   &JSONPb{},
			contentType: "application/json",
			frw: func(ctx context.Context, response proto.Message) (any, error) {
				return map[string]any{"ok": true, "data": response}, nil
			},
			getWantedResponse: func(msg any) ([]byte, error) {
				return new(JSONPb).Marshal(map[string]any{"ok": true, "data": msg})
			},
		},
	}

	ctx := NewServerMetadataContext(context.Background(), ServerMetadata{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			resp := httptest.NewRecorder()

			opts := []ServeMuxOption{}
			if tt.frw != nil {
				opts = append(opts, WithForwardResponseRewriter(tt.frw))
			}

			ForwardResponseMessage(ctx, NewServeMux(opts...), tt.marshaler, resp, req, msg)

			w := resp.Result()
			assert.Equal(t, http.StatusOK, w.StatusCode, "StatusCode mismatch")
			assert.Equal(t, tt.contentType, w.Header.Get("Content-Type"), "Content-Type mismatch")
			body, err := io.ReadAll(w.Body)
			assert.NoError(t, err, "Failed to read response body")
			w.Body.Close()

			if tt.getWantedResponse == nil {
				tt.getWantedResponse = tt.marshaler.Marshal
			}
			want, err := tt.getWantedResponse(msg)
			assert.NoError(t, err, "marshaler.Marshal() failed")
			assert.Equal(t, string(want), string(body), "ForwardResponseMessage body mismatch")
		})
	}
}

func TestOutgoingHeaderMatcher(t *testing.T) {
	t.Parallel()
	msg := &testpb.SimpleMessage{Id: "foo"}
	for _, tc := range []struct {
		name    string
		md      ServerMetadata
		headers http.Header
		matcher HeaderMatcherFunc
	}{
		{
			name:    "default matcher",
			md:      ServerMetadata{HeaderMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			headers: http.Header{"Content-Type": []string{"application/json"}, "Grpc-Metadata-Foo": []string{"bar"}, "Grpc-Metadata-Baz": []string{"qux"}},
		},
		{
			name:    "custom matcher",
			md:      ServerMetadata{HeaderMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			headers: http.Header{"Content-Type": []string{"application/json"}, "Custom-Foo": []string{"bar"}},
			matcher: func(key string) (string, bool) {
				if key == "foo" {
					return "custom-foo", true
				}
				return "", false
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewServerMetadataContext(context.Background(), tc.md)

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			resp := httptest.NewRecorder()

			mux := NewServeMux(WithOutgoingHeaderMatcher(tc.matcher))
			ForwardResponseMessage(ctx, mux, &JSONPb{}, resp, req, msg)

			w := resp.Result()
			defer w.Body.Close()
			assert.Equal(t, http.StatusOK, w.StatusCode, "StatusCode mismatch")
			assert.Equal(t, tc.headers, w.Header, "Header mismatch")
		})
	}
}

func TestOutgoingHeaderMatcherWithContentLength(t *testing.T) {
	t.Parallel()
	msg := &testpb.SimpleMessage{Id: "foo"}
	for _, tc := range []struct {
		name    string
		md      ServerMetadata
		headers http.Header
		matcher HeaderMatcherFunc
	}{
		{
			name:    "default matcher",
			md:      ServerMetadata{HeaderMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			headers: http.Header{"Content-Length": []string{"12"}, "Content-Type": []string{"application/json"}, "Grpc-Metadata-Foo": []string{"bar"}, "Grpc-Metadata-Baz": []string{"qux"}},
		},
		{
			name:    "custom matcher",
			md:      ServerMetadata{HeaderMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			headers: http.Header{"Content-Length": []string{"12"}, "Content-Type": []string{"application/json"}, "Custom-Foo": []string{"bar"}},
			matcher: func(key string) (string, bool) {
				if key == "foo" {
					return "custom-foo", true
				}
				return "", false
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewServerMetadataContext(context.Background(), tc.md)

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			resp := httptest.NewRecorder()

			mux := NewServeMux(WithOutgoingHeaderMatcher(tc.matcher), WithWriteContentLength())
			ForwardResponseMessage(ctx, mux, &JSONPb{}, resp, req, msg)

			w := resp.Result()
			defer w.Body.Close()
			assert.Equal(t, http.StatusOK, w.StatusCode, "StatusCode mismatch")
			assert.Equal(t, tc.headers, w.Header, "Header mismatch")
		})
	}
}

func TestOutgoingTrailerMatcher(t *testing.T) {
	t.Parallel()
	msg := &testpb.SimpleMessage{Id: "foo"}
	for _, tc := range []struct {
		name    string
		md      ServerMetadata
		caller  http.Header
		headers http.Header
		trailer http.Header
		matcher HeaderMatcherFunc
	}{
		{
			name:    "default matcher, caller accepts",
			md:      ServerMetadata{TrailerMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			caller:  http.Header{"Te": []string{"trailers"}},
			headers: http.Header{"Transfer-Encoding": []string{"chunked"}, "Content-Type": []string{"application/json"}, "Trailer": []string{"Grpc-Trailer-Baz", "Grpc-Trailer-Foo"}},
			trailer: http.Header{"Grpc-Trailer-Foo": []string{"bar"}, "Grpc-Trailer-Baz": []string{"qux"}},
		},
		{
			name:    "default matcher, caller rejects",
			md:      ServerMetadata{TrailerMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			headers: http.Header{"Content-Length": []string{"12"}, "Content-Type": []string{"application/json"}},
		},
		{
			name:    "custom matcher",
			md:      ServerMetadata{TrailerMD: metadata.Pairs("foo", "bar", "baz", "qux")},
			caller:  http.Header{"Te": []string{"trailers"}},
			headers: http.Header{"Transfer-Encoding": []string{"chunked"}, "Content-Type": []string{"application/json"}, "Trailer": []string{"Custom-Trailer-Foo"}},
			trailer: http.Header{"Custom-Trailer-Foo": []string{"bar"}},
			matcher: func(key string) (string, bool) {
				if key == "foo" {
					return "custom-trailer-foo", true
				}
				return "", false
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := NewServerMetadataContext(context.Background(), tc.md)

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			req.Header = tc.caller
			resp := httptest.NewRecorder()

			mux := NewServeMux(WithOutgoingTrailerMatcher(tc.matcher), WithWriteContentLength())
			ForwardResponseMessage(ctx, mux, &JSONPb{}, resp, req, msg)

			w := resp.Result()
			_, _ = io.Copy(io.Discard, w.Body)
			defer w.Body.Close()
			assert.Equal(t, http.StatusOK, w.StatusCode, "StatusCode mismatch")

			sort.Strings(w.Header["Trailer"])
			assert.Equal(t, tc.headers, w.Header, "Header mismatch")
			assert.Equal(t, tc.trailer, w.Trailer, "Trailer mismatch")
		})
	}
}
