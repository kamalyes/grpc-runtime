package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalerForRequest(t *testing.T) {
	ctx := context.Background()
	r, err := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	assert.NoError(t, err, `http.NewRequest("GET", "http://example.com", nil) failed`)

	mux := NewServeMux()

	r.Header.Set("Accept", "application/x-out")
	r.Header.Set("Content-Type", "application/x-in")
	in, out := MarshalerForRequest(mux, r)
	assert.IsType(t, &HTTPBodyMarshaler{}, in)
	assert.IsType(t, &HTTPBodyMarshaler{}, out)

	marshalers := []dummyMarshaler{0, 1, 2}
	specs := []struct {
		opt ServeMuxOption

		wantIn  Marshaler
		wantOut Marshaler
	}{
		{
			opt:     WithMarshalerOption(MIMEWildcard, &marshalers[0]),
			wantIn:  &marshalers[0],
			wantOut: &marshalers[0],
		},
		{
			opt:     WithMarshalerOption("application/x-in", &marshalers[1]),
			wantIn:  &marshalers[1],
			wantOut: &marshalers[1],
		},
		{
			opt:     WithMarshalerOption("application/x-out", &marshalers[2]),
			wantIn:  &marshalers[1],
			wantOut: &marshalers[2],
		},
	}
	for i, spec := range specs {
		var opts []ServeMuxOption
		for _, s := range specs[:i+1] {
			opts = append(opts, s.opt)
		}
		mux = NewServeMux(opts...)

		in, out = MarshalerForRequest(mux, r)
		assert.Same(t, spec.wantIn, in, "spec %d: in", i)
		assert.Same(t, spec.wantOut, out, "spec %d: out", i)
	}

	r.Header.Set("Content-Type", "application/x-in; charset=UTF-8")
	in, out = MarshalerForRequest(mux, r)
	assert.Same(t, &marshalers[1], in)
	assert.Same(t, &marshalers[2], out)

	r.Header.Set("Content-Type", "application/x-another")
	r.Header.Set("Accept", "application/x-another")
	in, out = MarshalerForRequest(mux, r)
	assert.Same(t, &marshalers[0], in)
	assert.Same(t, &marshalers[0], out)
}

type dummyMarshaler int

func (dummyMarshaler) ContentType(_ interface{}) string { return "" }
func (dummyMarshaler) Marshal(interface{}) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (dummyMarshaler) Unmarshal([]byte, interface{}) error {
	return errors.New("not implemented")
}

func (dummyMarshaler) NewDecoder(r io.Reader) Decoder {
	return dummyDecoder{}
}
func (dummyMarshaler) NewEncoder(w io.Writer) Encoder {
	return dummyEncoder{}
}

func (m dummyMarshaler) GoString() string {
	return fmt.Sprintf("dummyMarshaler(%d)", m)
}

type dummyDecoder struct{}

func (dummyDecoder) Decode(interface{}) error {
	return errors.New("not implemented")
}

type dummyEncoder struct{}

func (dummyEncoder) Encode(interface{}) error {
	return errors.New("not implemented")
}
