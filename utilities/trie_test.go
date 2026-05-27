package utilities_test

import (
	"testing"

	"github.com/kamalyes/grpc-runtime/utilities"
	"github.com/stretchr/testify/assert"
)

func TestMaxCommonPrefix(t *testing.T) {
	for _, spec := range []struct {
		da     utilities.DoubleArray
		tokens []string
		want   bool
	}{
		{da: utilities.DoubleArray{}, tokens: nil, want: false},
		{da: utilities.DoubleArray{}, tokens: []string{"foo"}, want: false},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0}, Base: []int{1, 1, 0}, Check: []int{0, 1, 2}}, tokens: nil, want: false},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0}, Base: []int{1, 1, 0}, Check: []int{0, 1, 2}}, tokens: []string{"foo"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0}, Base: []int{1, 1, 0}, Check: []int{0, 1, 2}}, tokens: []string{"bar"}, want: false},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 1, 2, 0, 0}, Check: []int{0, 1, 1, 2, 3}}, tokens: []string{"foo"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 1, 2, 0, 0}, Check: []int{0, 1, 1, 2, 3}}, tokens: []string{"bar"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 1, 2, 0, 0}, Check: []int{0, 1, 1, 2, 3}}, tokens: []string{"something-else"}, want: false},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 1, 2, 0, 0}, Check: []int{0, 1, 1, 2, 3}}, tokens: []string{"foo", "bar"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 3, 1, 0, 4, 0, 0}, Check: []int{0, 1, 1, 3, 2, 2, 5}}, tokens: []string{"foo"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 3, 1, 0, 4, 0, 0}, Check: []int{0, 1, 1, 3, 2, 2, 5}}, tokens: []string{"foo", "bar"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 3, 1, 0, 4, 0, 0}, Check: []int{0, 1, 1, 3, 2, 2, 5}}, tokens: []string{"bar"}, want: true},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 3, 1, 0, 4, 0, 0}, Check: []int{0, 1, 1, 3, 2, 2, 5}}, tokens: []string{"something-else"}, want: false},
		{da: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 3, 1, 0, 4, 0, 0}, Check: []int{0, 1, 1, 3, 2, 2, 5}}, tokens: []string{"foo", "bar", "baz"}, want: true},
	} {
		got := spec.da.HasCommonPrefix(spec.tokens)
		assert.Equal(t, spec.want, got, "%#v.HasCommonPrefix(%v)", spec.da, spec.tokens)
	}
}

func TestAdd(t *testing.T) {
	for _, spec := range []struct {
		tokens [][]string
		want   utilities.DoubleArray
	}{
		{want: utilities.DoubleArray{Encoding: make(map[string]int)}},
		{tokens: [][]string{{"foo"}}, want: utilities.DoubleArray{Encoding: map[string]int{"foo": 0}, Base: []int{1, 1, 0}, Check: []int{0, 1, 2}}},
		{tokens: [][]string{{"foo"}, {"bar"}}, want: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1}, Base: []int{1, 1, 2, 0, 0}, Check: []int{0, 1, 1, 2, 3}}},
		{tokens: [][]string{{"foo", "bar"}, {"foo", "baz"}}, want: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1, "baz": 2}, Base: []int{1, 1, 1, 2, 0, 0}, Check: []int{0, 1, 2, 2, 3, 4}}},
		{tokens: [][]string{{"foo", "bar"}, {"foo", "baz"}, {"qux"}}, want: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1, "baz": 2, "qux": 3}, Base: []int{1, 1, 1, 2, 3, 0, 0, 0}, Check: []int{0, 1, 2, 2, 1, 3, 4, 5}}},
		{tokens: [][]string{{"foo", "bar"}, {"foo", "baz", "bar"}, {"qux", "foo"}}, want: utilities.DoubleArray{Encoding: map[string]int{"foo": 0, "bar": 1, "baz": 2, "qux": 3}, Base: []int{1, 1, 1, 5, 8, 0, 3, 0, 5, 0}, Check: []int{0, 1, 2, 2, 1, 3, 4, 7, 5, 9}}},
	} {
		da := utilities.NewDoubleArray(spec.tokens)
		assert.Equal(t, spec.want.Encoding, da.Encoding, "da.Encoding; tokens = %#v", spec.tokens)
		assert.True(t, compareArray(da.Base, spec.want.Base), "da.Base = %v; want %v; tokens = %#v", da.Base, spec.want.Base, spec.tokens)
		assert.True(t, compareArray(da.Check, spec.want.Check), "da.Check = %v; want %v; tokens = %#v", da.Check, spec.want.Check, spec.tokens)
	}
}

func compareArray(got, want []int) bool {
	var i int
	for i = 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			return false
		}
	}
	if i < len(want) {
		return false
	}
	for ; i < len(got); i++ {
		if got[i] != 0 {
			return false
		}
	}
	return true
}
