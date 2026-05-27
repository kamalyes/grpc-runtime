package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/kamalyes/grpc-runtime/utilities"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestMuxServeHTTP(t *testing.T) {
	type stubPattern struct {
		method string
		ops    []int
		pool   []string
		verb   string
	}
	for i, spec := range []struct {
		patterns                  []stubPattern
		reqMethod                 string
		reqPath                   string
		headers                   map[string]string
		respStatus                int
		respContent               string
		disablePathLengthFallback bool
		unescapingMode            UnescapingMode
	}{
		{reqMethod: "GET", reqPath: "/", respStatus: http.StatusNotFound},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "GET", reqPath: "/foo", respStatus: http.StatusOK, respContent: "GET /foo"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "GET", reqPath: "/bar", respStatus: http.StatusNotFound},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpPush), 0}}, {method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "GET", reqPath: "/foo", respStatus: http.StatusOK, respContent: "GET /foo"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", respStatus: http.StatusOK, respContent: "POST /foo"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "DELETE", reqPath: "/foo", respStatus: http.StatusNotImplemented},
		{patterns: []stubPattern{{method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}, verb: "archive"}}, reqMethod: "DELETE", reqPath: "/foo/bar:archive", respStatus: http.StatusNotImplemented},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, respStatus: http.StatusOK, respContent: "GET /foo"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, respStatus: http.StatusNotImplemented, disablePathLengthFallback: true},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, respStatus: http.StatusOK, respContent: "POST /foo", disablePathLengthFallback: true},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded", "X-HTTP-Method-Override": "GET"}, respStatus: http.StatusOK, respContent: "GET /foo"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, respStatus: http.StatusOK, respContent: "GET /foo"},
		{patterns: []stubPattern{{method: "DELETE", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}, {method: "PUT", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}, {method: "PATCH", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, respStatus: http.StatusNotImplemented},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "/foo", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusNotImplemented},
		{patterns: []stubPattern{{method: "POST", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}, verb: "bar"}}, reqMethod: "POST", reqPath: "/foo:bar", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, respContent: "POST /foo:bar"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}}, {method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}, verb: "verb"}}, reqMethod: "GET", reqPath: "/foo/bar:verb", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, respContent: "GET /foo/{id=*}:verb"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}}}, reqMethod: "GET", reqPath: "/foo/bar", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, respContent: "GET /foo/{id=*}"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}}}, reqMethod: "GET", reqPath: "/foo/bar:123", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, respContent: "GET /foo/{id=*}"},
		{patterns: []stubPattern{{method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}, verb: "verb"}}, reqMethod: "POST", reqPath: "/foo/bar:verb", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, respContent: "POST /foo/{id=*}:verb"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"foo"}}}, reqMethod: "POST", reqPath: "foo", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusBadRequest},
		{patterns: []stubPattern{{method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}, verb: "verb:subverb"}}, reqMethod: "POST", reqPath: "/foo/bar:verb:subverb", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, respContent: "POST /foo/{id=*}:verb:subverb"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 1, int(utilities.OpCapture), 1, int(utilities.OpLitPush), 2}, pool: []string{"foo", "id", "bar"}}}, reqMethod: "POST", reqPath: "/foo/404%2fwith%2Fspace/bar", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusNotFound, unescapingMode: UnescapingModeLegacy},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1, int(utilities.OpLitPush), 2}, pool: []string{"foo", "id", "bar"}}}, reqMethod: "GET", reqPath: "/foo/success%2fwith%2Fspace/bar", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, unescapingMode: UnescapingModeAllExceptReserved, respContent: "GET /foo/{id=*}/bar"},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1, int(utilities.OpLitPush), 2}, pool: []string{"foo", "id", "bar"}}}, reqMethod: "GET", reqPath: "/foo/success%2fwith%2Fspace/bar", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusNotFound, unescapingMode: UnescapingModeAllCharacters},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1, int(utilities.OpLitPush), 2}, pool: []string{"foo", "id", "bar"}}}, reqMethod: "GET", reqPath: "/foo/success%2fwith%2Fspace/bar", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusNotFound, unescapingMode: UnescapingModeLegacy},
		{patterns: []stubPattern{{method: "GET", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPushM), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id", "bar"}}}, reqMethod: "GET", reqPath: "/foo/success%2fwith%2Fspace", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, unescapingMode: UnescapingModeAllExceptReserved, respContent: "GET /foo/{id=**}"},
		{patterns: []stubPattern{{method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpLitPush), 2, int(utilities.OpPush), 0, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 3}, pool: []string{"api", "v1", "organizations", "name"}, verb: "action"}}, reqMethod: "POST", reqPath: "/api/v1/" + url.QueryEscape("organizations/foo") + ":action", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, unescapingMode: UnescapingModeAllCharacters, respContent: "POST /api/v1/{name=organizations/*}:action"},
		{patterns: []stubPattern{{method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpLitPush), 2}, pool: []string{"api", "v1", "organizations"}, verb: "verb"}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpLitPush), 2}, pool: []string{"api", "v1", "organizations"}}, {method: "POST", ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpLitPush), 2}, pool: []string{"api", "v1", "dummies"}, verb: "verb"}}, reqMethod: "POST", reqPath: "/api/v1/organizations:verb", headers: map[string]string{"Content-Type": "application/json"}, respStatus: http.StatusOK, unescapingMode: UnescapingModeAllCharacters, respContent: "POST /api/v1/organizations:verb"},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var opts []ServeMuxOption
			opts = append(opts, WithUnescapingMode(spec.unescapingMode))
			if spec.disablePathLengthFallback {
				opts = append(opts, WithDisablePathLengthFallback())
			}
			mux := NewServeMux(opts...)
			for _, p := range spec.patterns {
				func(p stubPattern) {
					pat, err := NewPattern(1, p.ops, p.pool, p.verb)
					assert.NoError(t, err, "NewPattern(1, %#v, %#v, %q)", p.ops, p.pool, p.verb)
					mux.Handle(p.method, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
						_, _ = fmt.Fprintf(w, "%s %s", p.method, pat.String())
					})
				}(p)
			}

			reqUrl := fmt.Sprintf("https://host.example%s", spec.reqPath)
			ctx := context.Background()
			r, err := http.NewRequestWithContext(ctx, spec.reqMethod, reqUrl, bytes.NewReader(nil))
			assert.NoError(t, err, "http.NewRequestWithContext(%q, %q, nil)", spec.reqMethod, reqUrl)
			for name, value := range spec.headers {
				r.Header.Set(name, value)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)

			assert.Equal(t, spec.respStatus, w.Code, "w.Code; patterns=%v; req=%v", spec.patterns, r)
			if spec.respContent != "" {
				assert.Equal(t, spec.respContent, w.Body.String(), "w.Body; patterns=%v; req=%v", spec.patterns, r)
			}
		})
	}
}

func TestServeHTTP_WithMethodOverrideAndFormParsing(t *testing.T) {
	r := httptest.NewRequest("POST", "/foo", strings.NewReader("bar=hoge"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-HTTP-Method-Override", "GET")
	w := httptest.NewRecorder()

	NewServeMux().ServeHTTP(w, r)

	assert.Equal(t, "hoge", r.FormValue("bar"), "form is not parsed")
}

func TestServeMux_StaticPathFastPath(t *testing.T) {
	mux := NewServeMux()
	pat := MustPattern(NewPattern(1, []int{
		int(utilities.OpLitPush), 0,
		int(utilities.OpLitPush), 1,
	}, []string{"v1", "health"}, ""))

	mux.Handle(http.MethodGet, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		assert.Empty(t, pathParams, "pathParams")
		_, _ = w.Write([]byte("ok"))
	})

	_, ok := mux.staticHandlers.Lookup(http.MethodGet, "/v1/health")
	assert.True(t, ok, "static handler was not indexed")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/health", nil))

	assert.Equal(t, http.StatusOK, w.Code, "status")
	assert.Equal(t, "ok", w.Body.String(), "body")
}

func TestServeMux_DynamicPathNotStaticIndexed(t *testing.T) {
	mux := NewServeMux()
	pat := MustPattern(NewPattern(1, []int{
		int(utilities.OpLitPush), 0,
		int(utilities.OpPush), 0,
		int(utilities.OpCapture), 1,
	}, []string{"v1", "name"}, ""))

	mux.Handle(http.MethodGet, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		_, _ = w.Write([]byte(pathParams["name"]))
	})

	assert.Equal(t, 0, mux.staticHandlers.Len(http.MethodGet), "static handlers len")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/alice", nil))

	assert.Equal(t, "alice", w.Body.String(), "body")
}

func BenchmarkServeMux_StaticPath(b *testing.B) {
	mux := NewServeMux()
	pat := MustPattern(NewPattern(1, []int{
		int(utilities.OpLitPush), 0,
		int(utilities.OpLitPush), 1,
		int(utilities.OpLitPush), 2,
	}, []string{"api", "v1", "health"}, ""))
	mux.Handle(http.MethodGet, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkServeMux_DynamicPath(b *testing.B) {
	mux := NewServeMux()
	pat := MustPattern(NewPattern(1, []int{
		int(utilities.OpLitPush), 0,
		int(utilities.OpPush), 0,
		int(utilities.OpCapture), 1,
	}, []string{"api", "id"}, ""))
	mux.Handle(http.MethodGet, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/123", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}
}

var defaultHeaderMatcherTests = []struct {
	name     string
	in       string
	outValue string
	outValid bool
}{
	{"permanent HTTP header should return prefixed", "Accept", "grpcgateway-Accept", true},
	{"key prefixed with MetadataHeaderPrefix should return without the prefix", "Grpc-Metadata-Custom-Header", "Custom-Header", true},
	{"non-permanent HTTP header key without prefix should not return", "Custom-Header", "", false},
}

func TestDefaultHeaderMatcher(t *testing.T) {
	for _, tt := range defaultHeaderMatcherTests {
		t.Run(tt.name, func(t *testing.T) {
			out, valid := DefaultHeaderMatcher(tt.in)
			assert.Equal(t, tt.outValue, out, "value")
			assert.Equal(t, tt.outValid, valid, "valid")
		})
	}
}

var defaultRouteMatcherTests = []struct {
	name   string
	method string
	path   string
	valid  bool
}{
	{"Test route /", "GET", "/", true},
	{"Simple Endpoint", "GET", "/v1/{bucket}/do:action", true},
	{"Complex Endpoint", "POST", "/v1/b/{bucket_name=buckets/*}/o/{name}", true},
	{"Wildcard Endpoint", "GET", "/v1/endpoint/*", true},
	{"Invalid Endpoint", "POST", "v1/b/:name/do", false},
}

func TestServeMux_HandlePath(t *testing.T) {
	mux := NewServeMux()
	testFn := func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	}
	for _, tt := range defaultRouteMatcherTests {
		t.Run(tt.name, func(t *testing.T) {
			err := mux.HandlePath(tt.method, tt.path, testFn)
			if tt.valid {
				assert.NoError(t, err, "route %v with method %v and path %v", tt.name, tt.method, tt.path)
			} else {
				assert.Error(t, err, "route %v with method %v and path %v should be invalid", tt.name, tt.method, tt.path)
			}
		})
	}
}

var healthCheckTests = []struct {
	name           string
	code           codes.Code
	status         grpc_health_v1.HealthCheckResponse_ServingStatus
	httpStatusCode int
}{
	{"Test grpc error code", codes.NotFound, grpc_health_v1.HealthCheckResponse_UNKNOWN, http.StatusNotFound},
	{"Test HealthCheckResponse_SERVING", codes.OK, grpc_health_v1.HealthCheckResponse_SERVING, http.StatusOK},
	{"Test HealthCheckResponse_NOT_SERVING", codes.OK, grpc_health_v1.HealthCheckResponse_NOT_SERVING, http.StatusServiceUnavailable},
	{"Test HealthCheckResponse_UNKNOWN", codes.OK, grpc_health_v1.HealthCheckResponse_UNKNOWN, http.StatusServiceUnavailable},
	{"Test HealthCheckResponse_SERVICE_UNKNOWN", codes.OK, grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN, http.StatusNotFound},
}

func TestWithHealthzEndpoint_codes(t *testing.T) {
	for _, tt := range healthCheckTests {
		t.Run(tt.name, func(t *testing.T) {
			mux := NewServeMux(WithHealthzEndpoint(&dummyHealthCheckClient{status: tt.status, code: tt.code}))

			r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, r)

			assert.Equal(t, tt.httpStatusCode, rr.Code, "result http status code for grpc code %q and status %q", tt.code, tt.status)
		})
	}
}

func TestWithHealthEndpointAt_consistentWithHealthz(t *testing.T) {
	const endpointPath = "/healthz"

	r := httptest.NewRequest(http.MethodGet, endpointPath, nil)

	for _, tt := range healthCheckTests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			client := &dummyHealthCheckClient{
				status: tt.status,
				code:   tt.code,
			}

			w := httptest.NewRecorder()

			NewServeMux(
				WithHealthEndpointAt(client, endpointPath),
			).ServeHTTP(w, r)

			refW := httptest.NewRecorder()

			NewServeMux(
				WithHealthzEndpoint(client),
			).ServeHTTP(refW, r)

			assert.Equal(t, refW.Code, w.Code, "result http status code for grpc code %q and status %q", tt.code, tt.status)
		})
	}
}

func TestWithHealthzEndpoint_serviceParam(t *testing.T) {
	service := "test"

	// trigger error to output service in body
	dummyClient := dummyHealthCheckClient{status: grpc_health_v1.HealthCheckResponse_UNKNOWN, code: codes.Unknown}
	mux := NewServeMux(WithHealthzEndpoint(&dummyClient))

	r := httptest.NewRequest(http.MethodGet, "/healthz?service="+service, nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, r)

	assert.True(t, strings.Contains(rr.Body.String(), service), "service query parameter should be translated to HealthCheckRequest: expected %s to contain %s", rr.Body.String(), service)
}

func TestWithHealthzEndpoint_header(t *testing.T) {
	for _, tt := range healthCheckTests {
		t.Run(tt.name, func(t *testing.T) {
			mux := NewServeMux(WithHealthzEndpoint(&dummyHealthCheckClient{status: tt.status, code: tt.code}))

			r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, r)

			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "result http header Content-Type for grpc code %q and status %q", tt.code, tt.status)
		})
	}
}

var _ grpc_health_v1.HealthClient = (*dummyHealthCheckClient)(nil)

type dummyHealthCheckClient struct {
	status grpc_health_v1.HealthCheckResponse_ServingStatus
	code   codes.Code
}

func (g *dummyHealthCheckClient) Check(ctx context.Context, r *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	if g.code != codes.OK {
		return nil, status.Error(g.code, r.GetService())
	}

	return &grpc_health_v1.HealthCheckResponse{Status: g.status}, nil
}

func (g *dummyHealthCheckClient) Watch(ctx context.Context, r *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (grpc_health_v1.Health_WatchClient, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
func (g *dummyHealthCheckClient) List(ctx context.Context, r *grpc_health_v1.HealthListRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func TestServeMux_HandleMiddlewares(t *testing.T) {
	var mws []int
	mux := NewServeMux(WithMiddlewares(
		func(next HandlerFunc) HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
				mws = append(mws, 1)
				next(w, r, pathParams)
			}
		},
		func(next HandlerFunc) HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
				mws = append(mws, 2)
				next(w, r, pathParams)
			}
		},
	))
	err := mux.HandlePath("GET", "/test", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		assert.NotEmpty(t, mws, "middlewares not called")
		assert.Equal(t, 1, mws[0], "first middleware is not called first")
		assert.Equal(t, 2, mws[1], "second middleware is not called the second")
	})
	assert.NoError(t, err, "route test with method GET and path /test")

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code, "request not processed")
}

func TestServeMux_InjectPattern(t *testing.T) {
	mux := NewServeMux()
	err := mux.HandlePath("GET", "/test", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		p, ok := HTTPPattern(r.Context())
		assert.True(t, ok, "pattern is not injected")
		assert.Equal(t, "/test", p.String(), "pattern not /test")
	})
	assert.NoError(t, err, "route test with method GET and path /test")

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code, "request not processed")
}
