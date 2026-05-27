package utilities_test

import (
	"flag"
	"testing"

	"github.com/kamalyes/grpc-runtime/utilities"
	"github.com/stretchr/testify/assert"
)

func TestStringArrayFlag(t *testing.T) {
	for _, tt := range []struct {
		name  string
		flags []string
		want  string
	}{
		{name: "No Value", flags: []string{}, want: ""},
		{name: "Single Value", flags: []string{"--my_flag=1"}, want: "1"},
		{name: "Repeated Value", flags: []string{"--my_flag=1", "--my_flag=2"}, want: "1,2"},
		{name: "Repeated Same Value", flags: []string{"--my_flag=1", "--my_flag=1"}, want: "1,1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			flagSet := flag.NewFlagSet("test", flag.PanicOnError)
			result := utilities.StringArrayFlag(flagSet, "my_flag", "repeated flag")
			assert.NoError(t, flagSet.Parse(tt.flags), "flagSet.Parse()")
			assert.Equal(t, tt.want, result.String(), "StringArrayFlag()")
		})
	}
}
