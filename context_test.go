package runtime

import (
	"context"
	"encoding/base64"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

const (
	emptyForwardMetaCount = 1
)

func TestAnnotateContext_WorksWithEmpty(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	expectedHTTPPathPattern := "/v1"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/v1", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.Header.Add("Some-Irrelevant-Header", "some value")
	annotated, err := AnnotateContext(ctx, NewServeMux(), request, expectedRPCName, WithHTTPPathPattern(expectedHTTPPathPattern))
	assert.NoError(t, err, "AnnotateContext failed")

	md, ok := metadata.FromOutgoingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount, "metadata count mismatch")
}

func TestAnnotateContext_ForwardsGrpcMetadata(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	expectedHTTPPathPattern := "/v1"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/v1", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.Header.Add("Some-Irrelevant-Header", "some value")
	request.Header.Add("Grpc-Metadata-FooBar", "Value1")
	request.Header.Add("Grpc-Metadata-Foo-BAZ", "Value2")
	request.Header.Add("Grpc-Metadata-foo-bAz", "Value3")
	request.Header.Add("Authorization", "Token 1234567890")
	annotated, err := AnnotateContext(ctx, NewServeMux(), request, expectedRPCName, WithHTTPPathPattern(expectedHTTPPathPattern))
	assert.NoError(t, err, "AnnotateContext failed")

	md, ok := metadata.FromOutgoingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+4, "metadata count mismatch")
	assert.Equal(t, []string{"Value1"}, md["foobar"])
	assert.Equal(t, []string{"Value2", "Value3"}, md["foo-baz"])
	assert.Equal(t, []string{"Token 1234567890"}, md["grpcgateway-authorization"])
	assert.Equal(t, []string{"Token 1234567890"}, md["authorization"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)

	m, ok = HTTPPathPattern(annotated)
	assert.True(t, ok, "HTTPPathPattern failed")
	assert.Equal(t, expectedHTTPPathPattern, m)
}

func TestAnnotateContext_ForwardGrpcBinaryMetadata(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	binData := []byte("\x00test-binary-data")
	request.Header.Add("Grpc-Metadata-Test-Bin", base64.StdEncoding.EncodeToString(binData))

	annotated, err := AnnotateContext(ctx, NewServeMux(), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateContext failed")

	md, ok := metadata.FromOutgoingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+1, "metadata count mismatch")
	assert.Equal(t, []string{string(binData)}, md["test-bin"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateContext_AddsXForwardedHeaders(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://bar.foo.example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.RemoteAddr = "192.0.2.100:12345"

	serveMux := NewServeMux(WithIncomingHeaderMatcher(func(key string) (string, bool) {
		return key, true
	}))

	annotated, err := AnnotateContext(ctx, serveMux, request, expectedRPCName)
	assert.NoError(t, err, "AnnotateContext failed")

	md, ok := metadata.FromOutgoingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+1, "metadata count mismatch")
	assert.Equal(t, []string{"bar.foo.example.com"}, md["x-forwarded-host"])
	assert.Equal(t, []string{"192.0.2.100"}, md["x-forwarded-for"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateContext_AppendsToExistingXForwardedHeaders(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://bar.foo.example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.Header.Add("X-Forwarded-Host", "qux.example.com")
	request.Header.Add("X-Forwarded-For", "192.0.2.100")
	request.Header.Add("X-Forwarded-For", "192.0.2.101, 192.0.2.102")
	request.RemoteAddr = "192.0.2.200:12345"

	serveMux := NewServeMux(WithIncomingHeaderMatcher(func(key string) (string, bool) {
		return key, true
	}))

	annotated, err := AnnotateContext(ctx, serveMux, request, expectedRPCName)
	assert.NoError(t, err, "AnnotateContext failed")

	md, ok := metadata.FromOutgoingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+1, "metadata count mismatch")
	assert.Equal(t, []string{"qux.example.com"}, md["x-forwarded-host"])
	assert.Equal(t, []string{"192.0.2.100, 192.0.2.101, 192.0.2.102, 192.0.2.200"}, md["x-forwarded-for"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateContext_SupportsTimeouts(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	annotated, err := AnnotateContext(ctx, NewServeMux(), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateContext failed")
	_, ok := annotated.Deadline()
	assert.False(t, ok, "no deadline by default")

	const acceptableError = 50 * time.Millisecond
	DefaultContextTimeout = 10 * time.Second
	annotated, err = AnnotateContext(ctx, NewServeMux(), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateContext failed")
	deadline, ok := annotated.Deadline()
	assert.True(t, ok, "expected deadline")
	assert.WithinDuration(t, time.Now().Add(DefaultContextTimeout), deadline, acceptableError)

	for _, spec := range []struct {
		timeout string
		want    time.Duration
	}{
		{timeout: "17H", want: 17 * time.Hour},
		{timeout: "19M", want: 19 * time.Minute},
		{timeout: "23S", want: 23 * time.Second},
		{timeout: "1009m", want: 1009 * time.Millisecond},
		{timeout: "1000003u", want: 1000003 * time.Microsecond},
		{timeout: "100000007n", want: 100000007 * time.Nanosecond},
	} {
		request.Header.Set("Grpc-Timeout", spec.timeout)
		annotated, err = AnnotateContext(ctx, NewServeMux(), request, expectedRPCName)
		assert.NoError(t, err, "AnnotateContext failed; timeout=%q", spec.timeout)
		deadline, ok := annotated.Deadline()
		assert.True(t, ok, "expected deadline; timeout=%q", spec.timeout)
		assert.WithinDuration(t, time.Now().Add(spec.want), deadline, acceptableError)
		m, ok := RPCMethod(annotated)
		assert.True(t, ok, "RPCMethod failed")
		assert.Equal(t, expectedRPCName, m)
	}
}

func TestAnnotateContext_SupportsCustomAnnotators(t *testing.T) {
	ctx := context.Background()
	md1 := func(context.Context, *http.Request) metadata.MD { return metadata.New(map[string]string{"foo": "bar"}) }
	md2 := func(context.Context, *http.Request) metadata.MD { return metadata.New(map[string]string{"baz": "qux"}) }
	expected := metadata.New(map[string]string{"foo": "bar", "baz": "qux"})
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	annotated, err := AnnotateContext(ctx, NewServeMux(WithMetadata(md1), WithMetadata(md2)), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateContext failed")

	actual, _ := metadata.FromOutgoingContext(annotated)
	for key, e := range expected {
		a, ok := actual[key]
		assert.True(t, ok, "missing key %s", key)
		assert.Equal(t, e, a, "metadata.MD[%s] mismatch", key)
	}

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateIncomingContext_WorksWithEmpty(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	expectedHTTPPathPattern := "/v1"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/v1", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.Header.Add("Some-Irrelevant-Header", "some value")
	annotated, err := AnnotateIncomingContext(ctx, NewServeMux(), request, expectedRPCName, WithHTTPPathPattern(expectedHTTPPathPattern))
	assert.NoError(t, err, "AnnotateIncomingContext failed")

	md, ok := metadata.FromIncomingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount, "metadata count mismatch")

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateIncomingContext_ForwardsGrpcMetadata(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	expectedHTTPPathPattern := "/v1"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com/v1", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.Header.Add("Some-Irrelevant-Header", "some value")
	request.Header.Add("Grpc-Metadata-FooBar", "Value1")
	request.Header.Add("Grpc-Metadata-Foo-BAZ", "Value2")
	request.Header.Add("Grpc-Metadata-foo-bAz", "Value3")
	request.Header.Add("Authorization", "Token 1234567890")
	annotated, err := AnnotateIncomingContext(ctx, NewServeMux(), request, expectedRPCName, WithHTTPPathPattern(expectedHTTPPathPattern))
	assert.NoError(t, err, "AnnotateIncomingContext failed")

	md, ok := metadata.FromIncomingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+4, "metadata count mismatch")
	assert.Equal(t, []string{"Value1"}, md["foobar"])
	assert.Equal(t, []string{"Value2", "Value3"}, md["foo-baz"])
	assert.Equal(t, []string{"Token 1234567890"}, md["grpcgateway-authorization"])
	assert.Equal(t, []string{"Token 1234567890"}, md["authorization"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)

	m, ok = HTTPPathPattern(annotated)
	assert.True(t, ok, "HTTPPathPattern failed")
	assert.Equal(t, expectedHTTPPathPattern, m)
}

func TestAnnotateIncomingContext_ForwardGrpcBinaryMetadata(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://www.example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	binData := []byte("\x00test-binary-data")
	request.Header.Add("Grpc-Metadata-Test-Bin", base64.StdEncoding.EncodeToString(binData))

	annotated, err := AnnotateIncomingContext(ctx, NewServeMux(), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateIncomingContext failed")

	md, ok := metadata.FromIncomingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+1, "metadata count mismatch")
	assert.Equal(t, []string{string(binData)}, md["test-bin"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateIncomingContext_AddsXForwardedHeaders(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://bar.foo.example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.RemoteAddr = "192.0.2.100:12345"

	serveMux := NewServeMux(WithIncomingHeaderMatcher(func(key string) (string, bool) {
		return key, true
	}))

	annotated, err := AnnotateIncomingContext(ctx, serveMux, request, expectedRPCName)
	assert.NoError(t, err, "AnnotateIncomingContext failed")

	md, ok := metadata.FromIncomingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+1, "metadata count mismatch")
	assert.Equal(t, []string{"bar.foo.example.com"}, md["x-forwarded-host"])
	assert.Equal(t, []string{"192.0.2.100"}, md["x-forwarded-for"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateIncomingContext_AppendsToExistingXForwardedHeaders(t *testing.T) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://bar.foo.example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	request.Header.Add("X-Forwarded-Host", "qux.example.com")
	request.Header.Add("X-Forwarded-For", "192.0.2.100")
	request.Header.Add("X-Forwarded-For", "192.0.2.101, 192.0.2.102")
	request.RemoteAddr = "192.0.2.200:12345"

	serveMux := NewServeMux(WithIncomingHeaderMatcher(func(key string) (string, bool) {
		return key, true
	}))

	annotated, err := AnnotateIncomingContext(ctx, serveMux, request, expectedRPCName)
	assert.NoError(t, err, "AnnotateIncomingContext failed")

	md, ok := metadata.FromIncomingContext(annotated)
	assert.True(t, ok, "expected metadata in context")
	assert.Len(t, md, emptyForwardMetaCount+1, "metadata count mismatch")
	assert.Equal(t, []string{"qux.example.com"}, md["x-forwarded-host"])
	assert.Equal(t, []string{"192.0.2.100, 192.0.2.101, 192.0.2.102, 192.0.2.200"}, md["x-forwarded-for"])

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

func TestAnnotateIncomingContext_SupportsTimeouts(t *testing.T) {
	// 重置DefaultContextTimeout，因为TestAnnotateContext_SupportsTimeouts会修改它
	DefaultContextTimeout = 0 * time.Second
	expectedRPCName := "/example.Example/Example"
	ctx := context.Background()
	request, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	annotated, err := AnnotateIncomingContext(ctx, NewServeMux(), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateIncomingContext failed")
	_, ok := annotated.Deadline()
	assert.False(t, ok, "no deadline by default")

	const acceptableError = 50 * time.Millisecond
	DefaultContextTimeout = 10 * time.Second
	annotated, err = AnnotateIncomingContext(ctx, NewServeMux(), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateIncomingContext failed")
	deadline, ok := annotated.Deadline()
	assert.True(t, ok, "expected deadline")
	assert.WithinDuration(t, time.Now().Add(DefaultContextTimeout), deadline, acceptableError)

	for _, spec := range []struct {
		timeout string
		want    time.Duration
	}{
		{timeout: "17H", want: 17 * time.Hour},
		{timeout: "19M", want: 19 * time.Minute},
		{timeout: "23S", want: 23 * time.Second},
		{timeout: "1009m", want: 1009 * time.Millisecond},
		{timeout: "1000003u", want: 1000003 * time.Microsecond},
		{timeout: "100000007n", want: 100000007 * time.Nanosecond},
	} {
		request.Header.Set("Grpc-Timeout", spec.timeout)
		annotated, err = AnnotateIncomingContext(ctx, NewServeMux(), request, expectedRPCName)
		assert.NoError(t, err, "AnnotateIncomingContext failed; timeout=%q", spec.timeout)
		deadline, ok := annotated.Deadline()
		assert.True(t, ok, "expected deadline; timeout=%q", spec.timeout)
		assert.WithinDuration(t, time.Now().Add(spec.want), deadline, acceptableError)
		m, ok := RPCMethod(annotated)
		assert.True(t, ok, "RPCMethod failed")
		assert.Equal(t, expectedRPCName, m)
	}
}

func TestAnnotateIncomingContext_SupportsCustomAnnotators(t *testing.T) {
	ctx := context.Background()
	md1 := func(context.Context, *http.Request) metadata.MD { return metadata.New(map[string]string{"foo": "bar"}) }
	md2 := func(context.Context, *http.Request) metadata.MD { return metadata.New(map[string]string{"baz": "qux"}) }
	expected := metadata.New(map[string]string{"foo": "bar", "baz": "qux"})
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	assert.NoError(t, err, "http.NewRequestWithContext failed")

	annotated, err := AnnotateIncomingContext(ctx, NewServeMux(WithMetadata(md1), WithMetadata(md2)), request, expectedRPCName)
	assert.NoError(t, err, "AnnotateIncomingContext failed")

	actual, _ := metadata.FromIncomingContext(annotated)
	for key, e := range expected {
		a, ok := actual[key]
		assert.True(t, ok, "missing key %s", key)
		assert.Equal(t, e, a, "metadata.MD[%s] mismatch", key)
	}

	m, ok := RPCMethod(annotated)
	assert.True(t, ok, "RPCMethod failed")
	assert.Equal(t, expectedRPCName, m)
}

// avoid compiler optimising benchmark away
var benchResult = reflect.DeepEqual(nil, nil)

func BenchmarkAnnotateContext(b *testing.B) {
	ctx := context.Background()
	expectedRPCName := "/example.Example/Example"
	request, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	if err != nil {
		b.Fatal(err)
	}
	request.Header.Add("Grpc-Metadata-FooBar", "Value1")
	request.Header.Add("Grpc-Metadata-Foo-BAZ", "Value2")
	request.Header.Add("Authorization", "Token 1234567890")

	for i := 0; i < b.N; i++ {
		_, err := AnnotateContext(ctx, NewServeMux(), request, expectedRPCName)
		if err != nil {
			b.Fatal(err)
		}
	}
}
