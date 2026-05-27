package runtime

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kamalyes/grpc-runtime/utilities"
	"github.com/stretchr/testify/assert"
)

const (
	validVersion = 1
	anything     = 0
)

func TestNewPattern(t *testing.T) {
	for _, spec := range []struct {
		ops  []int
		pool []string
		verb string

		stackSizeWant, tailLenWant int
	}{
		{stackSizeWant: 0, tailLenWant: 0},
		{ops: []int{int(utilities.OpNop), anything}, stackSizeWant: 0, tailLenWant: 0},
		{ops: []int{int(utilities.OpPush), anything}, stackSizeWant: 1, tailLenWant: 0},
		{ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"abc"}, stackSizeWant: 1, tailLenWant: 0},
		{ops: []int{int(utilities.OpPushM), anything}, stackSizeWant: 1, tailLenWant: 0},
		{ops: []int{int(utilities.OpPush), anything, int(utilities.OpConcatN), 1}, stackSizeWant: 1, tailLenWant: 0},
		{ops: []int{int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 0}, pool: []string{"abc"}, stackSizeWant: 1, tailLenWant: 0},
		{ops: []int{int(utilities.OpPush), anything, int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPushM), anything, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 2}, pool: []string{"lit1", "lit2", "var1"}, stackSizeWant: 4, tailLenWant: 0},
		{ops: []int{int(utilities.OpPushM), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 2, int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1}, pool: []string{"lit1", "lit2", "var1"}, stackSizeWant: 2, tailLenWant: 2},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPushM), anything, int(utilities.OpLitPush), 2, int(utilities.OpConcatN), 3, int(utilities.OpLitPush), 3, int(utilities.OpCapture), 4}, pool: []string{"lit1", "lit2", "lit3", "lit4", "var1"}, stackSizeWant: 4, tailLenWant: 2},
		{ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"abc"}, verb: "LOCK", stackSizeWant: 1, tailLenWant: 0},
	} {
		pat, err := NewPattern(validVersion, spec.ops, spec.pool, spec.verb)
		if !assert.NoError(t, err, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, spec.verb) {
			continue
		}
		assert.Equal(t, spec.stackSizeWant, pat.stacksize, "pat.stacksize")
		assert.Equal(t, spec.tailLenWant, pat.tailLen, "pat.tailLen")
	}
}

func TestNewPatternWithWrongOp(t *testing.T) {
	for _, spec := range []struct {
		ops  []int
		pool []string
		verb string
	}{
		{ops: []int{-1, anything}},
		{ops: []int{int(utilities.OpEnd), 0}},
		{ops: []int{int(utilities.OpPush)}},
		{ops: []int{int(utilities.OpLitPush), -1}, pool: []string{"abc"}},
		{ops: []int{int(utilities.OpLitPush), 1}, pool: []string{"abc"}},
		{ops: []int{int(utilities.OpConcatN), -1}, pool: []string{"abc"}},
		{ops: []int{int(utilities.OpCapture), -1}, pool: []string{"abc"}},
		{ops: []int{int(utilities.OpCapture), 1}, pool: []string{"abc"}},
		{ops: []int{int(utilities.OpPushM), anything, int(utilities.OpLitPush), 0, int(utilities.OpPushM), anything}, pool: []string{"abc"}},
	} {
		_, err := NewPattern(validVersion, spec.ops, spec.pool, spec.verb)
		assert.ErrorIs(t, err, ErrInvalidPattern, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, spec.verb)
	}
}

func TestNewPatternWithStackUnderflow(t *testing.T) {
	for _, spec := range []struct {
		ops  []int
		pool []string
		verb string
	}{
		{ops: []int{int(utilities.OpConcatN), 1}},
		{ops: []int{int(utilities.OpCapture), 0}, pool: []string{"abc"}},
	} {
		_, err := NewPattern(validVersion, spec.ops, spec.pool, spec.verb)
		assert.ErrorIs(t, err, ErrInvalidPattern, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, spec.verb)
	}
}

func TestMatch(t *testing.T) {
	for _, spec := range []struct {
		ops      []int
		pool     []string
		verb     string
		match    []string
		notMatch []string
	}{
		{match: []string{""}, notMatch: []string{"example"}},
		{ops: []int{int(utilities.OpNop), anything}, match: []string{""}, notMatch: []string{"example", "path/to/example"}},
		{ops: []int{int(utilities.OpPush), anything}, match: []string{"abc", "def"}, notMatch: []string{"", "abc/def"}},
		{ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"v1"}, match: []string{"v1"}, notMatch: []string{"", "v2"}},
		{ops: []int{int(utilities.OpPushM), anything}, match: []string{"", "abc", "abc/def", "abc/def/ghi"}},
		{ops: []int{int(utilities.OpPushM), anything, int(utilities.OpLitPush), 0}, pool: []string{"tail"}, match: []string{"tail", "abc/tail", "abc/def/tail"}, notMatch: []string{"", "abc", "abc/def", "tail/extra", "abc/tail/extra", "abc/def/tail/extra"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 2}, pool: []string{"v1", "bucket", "name"}, match: []string{"v1/bucket/my-bucket", "v1/bucket/our-bucket"}, notMatch: []string{"", "v1", "v1/bucket", "v2/bucket/my-bucket", "v1/pubsub/my-topic"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPushM), anything, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 2}, pool: []string{"v1", "o", "name"}, match: []string{"v1/o", "v1/o/my-bucket", "v1/o/our-bucket", "v1/o/my-bucket/dir", "v1/o/my-bucket/dir/dir2", "v1/o/my-bucket/dir/dir2/obj"}, notMatch: []string{"", "v1", "v2/o/my-bucket", "v1/b/my-bucket"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPush), anything, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 2, int(utilities.OpLitPush), 3, int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 4}, pool: []string{"v2", "b", "name", "o", "oname"}, match: []string{"v2/b/my-bucket/o/obj", "v2/b/our-bucket/o/obj", "v2/b/my-bucket/o/dir"}, notMatch: []string{"", "v2", "v2/b", "v2/b/my-bucket", "v2/b/my-bucket/o"}},
		{ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"v1"}, verb: "LOCK", match: []string{"v1:LOCK"}, notMatch: []string{"v1", "LOCK"}},
	} {
		pat, err := NewPattern(validVersion, spec.ops, spec.pool, spec.verb)
		if !assert.NoError(t, err, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, spec.verb) {
			continue
		}

		for _, path := range spec.match {
			_, err = pat.Match(segments(path))
			assert.NoError(t, err, "pat.Match(%q); pattern = (%v, %q)", path, spec.ops, spec.pool)
		}

		for _, path := range spec.notMatch {
			_, err = pat.Match(segments(path))
			assert.ErrorIs(t, err, ErrNotMatch, "pat.Match(%q); pattern = (%v, %q)", path, spec.ops, spec.pool)
		}
	}
}

func TestMatchWithBinding(t *testing.T) {
	for _, spec := range []struct {
		ops  []int
		pool []string
		path string
		verb string
		mode UnescapingMode
		want map[string]string
	}{
		{want: map[string]string{}},
		{ops: []int{int(utilities.OpNop), anything}, want: map[string]string{}},
		{ops: []int{int(utilities.OpPush), anything}, path: "abc", want: map[string]string{}},
		{ops: []int{int(utilities.OpPush), anything}, verb: "LOCK", path: "abc:LOCK", want: map[string]string{}},
		{ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"endpoint"}, path: "endpoint", want: map[string]string{}},
		{ops: []int{int(utilities.OpPushM), anything}, path: "abc/def/ghi", want: map[string]string{}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 2}, pool: []string{"v1", "bucket", "name"}, path: "v1/bucket/my-bucket", want: map[string]string{"name": "my-bucket"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 2}, pool: []string{"v1", "bucket", "name"}, verb: "LOCK", path: "v1/bucket/my-bucket:LOCK", want: map[string]string{"name": "my-bucket"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPushM), anything, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 2}, pool: []string{"v1", "o", "name"}, path: "v1/o/my-bucket/dir/dir2/obj", want: map[string]string{"name": "o/my-bucket/dir/dir2/obj"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPushM), anything, int(utilities.OpLitPush), 2, int(utilities.OpConcatN), 3, int(utilities.OpCapture), 4, int(utilities.OpLitPush), 3}, pool: []string{"v1", "o", ".ext", "tail", "name"}, path: "v1/o/my-bucket/dir/dir2/obj/.ext/tail", want: map[string]string{"name": "o/my-bucket/dir/dir2/obj/.ext"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPush), anything, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 2, int(utilities.OpLitPush), 3, int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 4}, pool: []string{"v2", "b", "name", "o", "oname"}, path: "v2/b/my-bucket/o/obj", want: map[string]string{"name": "b/my-bucket", "oname": "obj"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPush), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1, int(utilities.OpLitPush), 2}, pool: []string{"foo", "id", "bar"}, path: "foo/part1%2Fpart2/bar", mode: UnescapingModeAllExceptReserved, want: map[string]string{"id": "part1/part2"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPushM), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}, path: "foo/test%2Fbar", mode: UnescapingModeAllExceptReserved, want: map[string]string{"id": "test%2Fbar"}},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpPushM), 0, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 1}, pool: []string{"foo", "id"}, path: "foo/test%2Fbar", mode: UnescapingModeAllCharacters, want: map[string]string{"id": "test/bar"}},
	} {
		pat, err := NewPattern(validVersion, spec.ops, spec.pool, spec.verb)
		if !assert.NoError(t, err, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, spec.verb) {
			continue
		}

		components, verb := segments(spec.path)
		got, err := pat.MatchAndEscape(components, verb, spec.mode)
		assert.NoError(t, err, "pat.Match(%q); pattern = (%v, %q)", spec.path, spec.ops, spec.pool)
		assert.True(t, reflect.DeepEqual(got, spec.want), "pat.Match(%q) = %q, want %q; pattern = (%v, %q)", spec.path, got, spec.want, spec.ops, spec.pool)
	}
}

func segments(path string) (components []string, verb string) {
	if path == "" {
		return nil, ""
	}
	components = strings.Split(path, "/")
	l := len(components)
	c := components[l-1]
	if idx := strings.LastIndex(c, ":"); idx >= 0 {
		components[l-1], verb = c[:idx], c[idx+1:]
	}
	return components, verb
}

func TestPatternString(t *testing.T) {
	for _, spec := range []struct {
		ops  []int
		pool []string
		want string
	}{
		{want: "/"},
		{ops: []int{int(utilities.OpNop), anything}, want: "/"},
		{ops: []int{int(utilities.OpPush), anything}, want: "/*"},
		{ops: []int{int(utilities.OpLitPush), 0}, pool: []string{"endpoint"}, want: "/endpoint"},
		{ops: []int{int(utilities.OpPushM), anything}, want: "/**"},
		{ops: []int{int(utilities.OpPush), anything, int(utilities.OpConcatN), 1}, want: "/*"},
		{ops: []int{int(utilities.OpPush), anything, int(utilities.OpConcatN), 1, int(utilities.OpCapture), 0}, pool: []string{"name"}, want: "/{name=*}"},
		{ops: []int{int(utilities.OpLitPush), 0, int(utilities.OpLitPush), 1, int(utilities.OpPush), anything, int(utilities.OpConcatN), 2, int(utilities.OpCapture), 2, int(utilities.OpLitPush), 3, int(utilities.OpPushM), anything, int(utilities.OpLitPush), 4, int(utilities.OpConcatN), 3, int(utilities.OpCapture), 6, int(utilities.OpLitPush), 5}, pool: []string{"v1", "buckets", "bucket_name", "objects", ".ext", "tail", "name"}, want: "/v1/{bucket_name=buckets/*}/{name=objects/**/.ext}/tail"},
	} {
		p, err := NewPattern(validVersion, spec.ops, spec.pool, "")
		if !assert.NoError(t, err, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, "") {
			continue
		}
		assert.Equal(t, spec.want, p.String())

		verb := "LOCK"
		p, err = NewPattern(validVersion, spec.ops, spec.pool, verb)
		if !assert.NoError(t, err, "NewPattern(%d, %v, %q, %q)", validVersion, spec.ops, spec.pool, verb) {
			continue
		}
		assert.Equal(t, fmt.Sprintf("%s:%s", spec.want, verb), p.String())
	}
}
