package descriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGoPackageStandard(t *testing.T) {
	for _, spec := range []struct {
		pkg  GoPackage
		want bool
	}{
		{pkg: GoPackage{Path: "fmt", Name: "fmt"}, want: true},
		{pkg: GoPackage{Path: "encoding/json", Name: "json"}, want: true},
		{pkg: GoPackage{Path: "google.golang.org/protobuf/encoding/protojson", Name: "jsonpb"}, want: false},
		{pkg: GoPackage{Path: "golang.org/x/net/context", Name: "context"}, want: false},
		{pkg: GoPackage{Path: "github.com/kamalyes/grpc-runtime", Name: "main"}, want: false},
		{pkg: GoPackage{Path: "github.com/google/googleapis/google/api/http.pb", Name: "http_pb", Alias: "htpb"}, want: false},
	} {
		assert.Equal(t, spec.want, spec.pkg.Standard())
	}
}

func TestGoPackageString(t *testing.T) {
	for _, spec := range []struct {
		pkg  GoPackage
		want string
	}{
		{pkg: GoPackage{Path: "fmt", Name: "fmt"}, want: `"fmt"`},
		{pkg: GoPackage{Path: "encoding/json", Name: "json"}, want: `"encoding/json"`},
		{pkg: GoPackage{Path: "google.golang.org/protobuf/encoding/protojson", Name: "jsonpb"}, want: `"google.golang.org/protobuf/encoding/protojson"`},
		{pkg: GoPackage{Path: "golang.org/x/net/context", Name: "context"}, want: `"golang.org/x/net/context"`},
		{pkg: GoPackage{Path: "github.com/kamalyes/grpc-runtime", Name: "main"}, want: `"github.com/kamalyes/grpc-runtime"`},
		{pkg: GoPackage{Path: "github.com/google/googleapis/google/api/http.pb", Name: "http_pb", Alias: "htpb"}, want: `htpb "github.com/google/googleapis/google/api/http.pb"`},
	} {
		assert.Equal(t, spec.want, spec.pkg.String())
	}
}

func TestFieldPath(t *testing.T) {
	var fds []*descriptorpb.FileDescriptorProto
	for _, src := range []string{
		`
		name: 'example.proto'
		package: 'example'
		message_type <
			name: 'Nest'
			field <
				name: 'nest2_field'
				label: LABEL_OPTIONAL
				type: TYPE_MESSAGE
				type_name: 'Nest2'
				number: 1
			>
			field <
				name: 'terminal_field'
				label: LABEL_OPTIONAL
				type: TYPE_STRING
				number: 2
			>
		>
		syntax: "proto3"
		`, `
		name: 'another.proto'
		package: 'example'
		message_type <
			name: 'Nest2'
			field <
				name: 'nest_field'
				label: LABEL_OPTIONAL
				type: TYPE_MESSAGE
				type_name: 'Nest'
				number: 1
			>
			field <
				name: 'terminal_field'
				label: LABEL_OPTIONAL
				type: TYPE_STRING
				number: 2
			>
		>
		syntax: "proto2"
		`,
	} {
		var fd descriptorpb.FileDescriptorProto
		if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
			return
		}
		fds = append(fds, &fd)
	}
	nest1 := &Message{
		DescriptorProto: fds[0].MessageType[0],
		Fields: []*Field{
			{FieldDescriptorProto: fds[0].MessageType[0].Field[0]},
			{FieldDescriptorProto: fds[0].MessageType[0].Field[1]},
		},
	}
	nest2 := &Message{
		DescriptorProto: fds[1].MessageType[0],
		Fields: []*Field{
			{FieldDescriptorProto: fds[1].MessageType[0].Field[0]},
			{FieldDescriptorProto: fds[1].MessageType[0].Field[1]},
		},
	}
	file1 := &File{
		FileDescriptorProto: fds[0],
		GoPkg:               GoPackage{Path: "example", Name: "example"},
		Messages:            []*Message{nest1},
	}
	file2 := &File{
		FileDescriptorProto: fds[1],
		GoPkg:               GoPackage{Path: "example", Name: "example"},
		Messages:            []*Message{nest2},
	}
	crossLinkFixture(file1)
	crossLinkFixture(file2)

	c1 := FieldPathComponent{
		Name:   "nest_field",
		Target: nest2.Fields[0],
	}
	assert.Equal(t, "GetNestField()", c1.ValueExpr())
	assert.Equal(t, "NestField", c1.AssignableExpr())

	c2 := FieldPathComponent{
		Name:   "nest2_field",
		Target: nest1.Fields[0],
	}
	assert.Equal(t, "Nest2Field", c2.ValueExpr())
	assert.Equal(t, "Nest2Field", c2.ValueExpr())

	fp := FieldPath{
		c1, c2, c1, FieldPathComponent{
			Name:   "terminal_field",
			Target: nest1.Fields[1],
		},
	}
	assert.Equal(t, "resp.GetNestField().Nest2Field.GetNestField().TerminalField", fp.AssignableExpr("resp", "example"))

	fp2 := FieldPath{
		c2, c1, c2, FieldPathComponent{
			Name:   "terminal_field",
			Target: nest2.Fields[1],
		},
	}
	assert.Equal(t, "resp.Nest2Field.GetNestField().Nest2Field.TerminalField", fp2.AssignableExpr("resp", "example"))

	var fpEmpty FieldPath
	assert.Equal(t, "resp", fpEmpty.AssignableExpr("resp", "example"))
}

func TestGoType(t *testing.T) {
	src := `
		name: 'example.proto'
		package: 'example'
		message_type <
			name: 'Message'
			field <
				name: 'field'
				type: TYPE_STRING
				number: 1
			>
		>,
		enum_type <
			name: 'EnumName'
		>,
	`

	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}

	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{FieldDescriptorProto: fd.MessageType[0].Field[0]},
		},
	}
	enum := &Enum{
		EnumDescriptorProto: fd.EnumType[0],
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg:               GoPackage{Path: "example", Name: "example"},
		Messages:            []*Message{msg},
		Enums:               []*Enum{enum},
	}
	crossLinkFixture(file)

	assert.Equal(t, "Message", msg.GoType("example"))
	assert.Equal(t, "example.Message", msg.GoType("extPackage"))
	msg.ForcePrefixedName = true
	assert.Equal(t, "example.Message", msg.GoType("example"))

	assert.Equal(t, "EnumName", enum.GoType("example"))
	assert.Equal(t, "example.EnumName", enum.GoType("extPackage"))
	enum.ForcePrefixedName = true
	assert.Equal(t, "example.EnumName", enum.GoType("example"))

}
