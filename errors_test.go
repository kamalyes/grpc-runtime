package runtime

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestDefaultHTTPError(t *testing.T) {
	ctx := context.Background()

	statusWithDetails, _ := status.New(codes.FailedPrecondition, "failed precondition").WithDetails(
		&errdetails.PreconditionFailure{},
	)

	for i, spec := range []struct {
		err                  error
		status               int
		msg                  string
		marshaler            Marshaler
		contentType          string
		details              string
		fordwardRespRewriter ForwardResponseRewriter
		extractMessage       func(*testing.T)
	}{
		{
			err:         errors.New("example error"),
			status:      http.StatusInternalServerError,
			marshaler:   &JSONPb{},
			contentType: "application/json",
			msg:         "example error",
		},
		{
			err:         status.Error(codes.NotFound, "no such resource"),
			status:      http.StatusNotFound,
			marshaler:   &JSONPb{},
			contentType: "application/json",
			msg:         "no such resource",
		},
		{
			err:         statusWithDetails.Err(),
			status:      http.StatusBadRequest,
			marshaler:   &JSONPb{},
			contentType: "application/json",
			msg:         "failed precondition",
			details:     "type.googleapis.com/google.rpc.PreconditionFailure",
		},
		{
			err:         errors.New("example error"),
			status:      http.StatusInternalServerError,
			marshaler:   &CustomMarshaler{&JSONPb{}},
			contentType: "Custom-Content-Type",
			msg:         "example error",
		},
		{
			err: &HTTPStatusError{
				HTTPStatus: http.StatusMethodNotAllowed,
				Err:        status.Error(codes.Unimplemented, http.StatusText(http.StatusMethodNotAllowed)),
			},
			status:      http.StatusMethodNotAllowed,
			marshaler:   &JSONPb{},
			contentType: "application/json",
			msg:         "Method Not Allowed",
		},
		{
			err:         status.Error(codes.InvalidArgument, "example error"),
			status:      http.StatusBadRequest,
			marshaler:   &JSONPb{},
			contentType: "application/json",
			msg:         "bad request: example error",
			fordwardRespRewriter: func(ctx context.Context, response proto.Message) (any, error) {
				if s, ok := response.(*statuspb.Status); ok && strings.HasPrefix(s.Message, "example") {
					return &statuspb.Status{
						Code:    s.Code,
						Message: "bad request: " + s.Message,
						Details: s.Details,
					}, nil
				}
				return response, nil
			},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(ctx, "", "", nil)

			opts := []ServeMuxOption{}
			if spec.fordwardRespRewriter != nil {
				opts = append(opts, WithForwardResponseRewriter(spec.fordwardRespRewriter))
			}
			mux := NewServeMux(opts...)

			HTTPError(ctx, mux, spec.marshaler, w, req, spec.err)

			assert.Equal(t, spec.contentType, w.Header().Get("Content-Type"))
			assert.Equal(t, spec.status, w.Code)

			var st statuspb.Status
			err := spec.marshaler.Unmarshal(w.Body.Bytes(), &st)
			assert.NoError(t, err)
			assert.True(t, strings.Contains(st.Message, spec.msg), "st.Message = %q, want to contain %q", st.Message, spec.msg)

			if spec.details != "" {
				assert.Len(t, st.Details, 1)
				assert.Equal(t, spec.details, st.Details[0].TypeUrl)
			}
		})
	}
}

func TestHTTPStreamError(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name             string
		err              error
		expectedStatus   *status.Status
		expectedResponse []byte
	}{
		{
			name:             "Simple error",
			err:              errors.New("simple error"),
			expectedStatus:   status.New(codes.Unknown, "simple error"),
			expectedResponse: []byte(`{"error":{"code":2,"message":"simple error"}}`),
		},
		{
			name:             "Invalid request error",
			err:              status.Error(codes.InvalidArgument, "invalid request"),
			expectedStatus:   status.New(codes.InvalidArgument, "invalid request"),
			expectedResponse: []byte(`{"error":{"code":3,"message":"invalid request"}}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			mux := NewServeMux(WithStreamErrorHandler(
				DefaultStreamErrorHandler,
			))

			marshaler := &JSONPb{}

			HTTPStreamError(ctx, mux, marshaler, w, r, tc.err)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, proto.Equal(status.Convert(tc.err).Proto(), tc.expectedStatus.Proto()), "expected %v, got %v", tc.expectedStatus, status.Convert(tc.err))
			assert.True(t, bytes.Equal(w.Body.Bytes(), tc.expectedResponse), "expected %s, got %s", tc.expectedResponse, w.Body.Bytes())
		})
	}
}
