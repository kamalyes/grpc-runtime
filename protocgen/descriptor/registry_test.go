package descriptor

import (
	"testing"

	"github.com/kamalyes/grpc-runtime/protocgen/descriptor/openapiconfig"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	testProtoPackage  = "grpc.runtime.testpb"
	testGoPackage     = "github.com/kamalyes/grpc-runtime/testpb"
	testGoPackageName = "testpb"
)

func newGeneratorFromSources(req *pluginpb.CodeGeneratorRequest, sources ...string) (*protogen.Plugin, error) {
	for _, src := range sources {
		var fd descriptorpb.FileDescriptorProto
		if err := prototext.Unmarshal([]byte(src), &fd); err != nil {
			return nil, err
		}
		req.FileToGenerate = append(req.FileToGenerate, fd.GetName())
		req.ProtoFile = append(req.ProtoFile, &fd)
	}
	return protogen.Options{}.New(req)
}

func loadFileWithCodeGeneratorRequest(t *testing.T, reg *Registry, req *pluginpb.CodeGeneratorRequest, sources ...string) []*descriptorpb.FileDescriptorProto {
	t.Helper()
	plugin, err := newGeneratorFromSources(req, sources...)
	if !assert.NoError(t, err) {
		return nil
	}
	err = reg.LoadFromPlugin(plugin)
	if !assert.NoError(t, err) {
		return nil
	}
	return plugin.Request.ProtoFile
}

func loadFile(t *testing.T, reg *Registry, src string) *descriptorpb.FileDescriptorProto {
	t.Helper()
	fds := loadFileWithCodeGeneratorRequest(t, reg, &pluginpb.CodeGeneratorRequest{}, src)
	if !assert.NotEmpty(t, fds) {
		return nil
	}
	return fds[0]
}

func assertLoadedFile(t *testing.T, reg *Registry, name string) *File {
	t.Helper()
	file := reg.files[name]
	if !assert.NotNil(t, file) {
		return nil
	}
	return file
}

func TestLoadFile(t *testing.T) {
	reg := NewRegistry()
	fd := loadFile(t, reg, `
		name: 'example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
		message_type <
			name: 'ExampleMessage'
			field <
				name: 'str'
				label: LABEL_OPTIONAL
				type: TYPE_STRING
				number: 1
			>
		>
	`)

	file := assertLoadedFile(t, reg, "example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: testGoPackage, Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)

	msg, err := reg.LookupMsg("", "."+testProtoPackage+".ExampleMessage")
	if !assert.NoError(t, err) {
		return
	}
	assert.Same(t, fd.MessageType[0], msg.DescriptorProto)
	assert.Same(t, file, msg.File)
	assert.Nil(t, msg.Outers)
	if assert.Len(t, msg.Fields, 1) {
		assert.Same(t, fd.MessageType[0].Field[0], msg.Fields[0].FieldDescriptorProto)
		assert.Same(t, msg, msg.Fields[0].Message)
	}

	if assert.Len(t, file.Messages, 1) {
		assert.Same(t, msg, file.Messages[0])
	}
}

func TestLoadFileNestedPackage(t *testing.T) {
	reg := NewRegistry()
	loadFile(t, reg, `
		name: 'example.proto'
		package: 'grpc.runtime.testpb.nested'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb/nested' >
	`)

	file := assertLoadedFile(t, reg, "example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: testGoPackage + "/nested", Name: "nested"}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestLoadFileWithDir(t *testing.T) {
	reg := NewRegistry()
	loadFile(t, reg, `
		name: 'path/to/example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
	`)

	file := assertLoadedFile(t, reg, "path/to/example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: testGoPackage, Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestLoadFileWithoutPackage(t *testing.T) {
	reg := NewRegistry()
	loadFile(t, reg, `
		name: 'path/to/example_file.proto'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
	`)

	file := assertLoadedFile(t, reg, "path/to/example_file.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: testGoPackage, Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestLoadFileWithMapping(t *testing.T) {
	reg := NewRegistry()
	loadFileWithCodeGeneratorRequest(t, reg, &pluginpb.CodeGeneratorRequest{
		Parameter: proto.String("Mpath/to/example.proto=example.com/proj/example/proto"),
	}, `
		name: 'path/to/example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
	`)

	file := assertLoadedFile(t, reg, "path/to/example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: "example.com/proj/example/proto", Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestLoadFileWithPackageNameCollision(t *testing.T) {
	reg := NewRegistry()
	loadFile(t, reg, `
		name: 'path/to/another.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
	`)
	loadFile(t, reg, `
		name: 'path/to/example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb/alternate;testpb' >
	`)
	if !assert.NoError(t, reg.ReserveGoPackageAlias("ioutil", "io/ioutil")) {
		return
	}
	loadFile(t, reg, `
		name: 'path/to/ioutil.proto'
		package: 'ioutil'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb/ioutil;ioutil' >
	`)

	file := assertLoadedFile(t, reg, "path/to/another.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: testGoPackage, Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)

	file = assertLoadedFile(t, reg, "path/to/example.proto")
	if file == nil {
		return
	}
	wantPkg = GoPackage{Path: testGoPackage + "/alternate", Name: testGoPackageName, Alias: "testpb_0"}
	assert.Equal(t, wantPkg, file.GoPkg)

	file = assertLoadedFile(t, reg, "path/to/ioutil.proto")
	if file == nil {
		return
	}
	wantPkg = GoPackage{Path: testGoPackage + "/ioutil", Name: "ioutil", Alias: "ioutil_0"}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestLoadFileWithIdenticalGoPkg(t *testing.T) {
	reg := NewRegistry()
	loadFileWithCodeGeneratorRequest(t, reg, &pluginpb.CodeGeneratorRequest{
		Parameter: proto.String("Mpath/to/another.proto=example.com/example,Mpath/to/example.proto=example.com/example"),
	}, `
		name: 'path/to/another.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
	`, `
		name: 'path/to/example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
	`)

	file := assertLoadedFile(t, reg, "path/to/example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: "example.com/example", Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)

	file = assertLoadedFile(t, reg, "path/to/another.proto")
	if file == nil {
		return
	}
	wantPkg = GoPackage{Path: "example.com/example", Name: testGoPackageName}
	assert.Equal(t, wantPkg, file.GoPkg)
}

// TestLookupMsgWithoutPackage tests a case when there is no "package" directive.
// In Go, it is required to have a generated package so we rely on
// google.golang.org/protobuf/compiler/protogen to provide it.
func TestLookupMsgWithoutPackage(t *testing.T) {
	reg := NewRegistry()
	fd := loadFile(t, reg, `
		name: 'example.proto'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
		message_type <
			name: 'ExampleMessage'
			field <
				name: 'str'
				label: LABEL_OPTIONAL
				type: TYPE_STRING
				number: 1
			>
		>
	`)

	msg, err := reg.LookupMsg("", ".ExampleMessage")
	if !assert.NoError(t, err) {
		return
	}
	assert.Same(t, fd.MessageType[0], msg.DescriptorProto)
}

func TestLookupMsgWithNestedPackage(t *testing.T) {
	reg := NewRegistry()
	fd := loadFile(t, reg, `
		name: 'example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
		message_type <
			name: 'ExampleMessage'
			field <
				name: 'str'
				label: LABEL_OPTIONAL
				type: TYPE_STRING
				number: 1
			>
		>
	`)

	for _, name := range []string{
		"grpc.runtime.testpb.ExampleMessage",
		"runtime.testpb.ExampleMessage",
		"testpb.ExampleMessage",
		"ExampleMessage",
	} {
		msg, err := reg.LookupMsg(testProtoPackage, name)
		if !assert.NoError(t, err) {
			return
		}
		assert.Same(t, fd.MessageType[0], msg.DescriptorProto)
	}

	for _, loc := range []string{
		"." + testProtoPackage,
		testProtoPackage,
		".grpc.runtime",
		"grpc.runtime",
		".grpc",
		"grpc",
		".",
		"",
		"somewhere.else",
	} {
		name := testProtoPackage + ".ExampleMessage"
		msg, err := reg.LookupMsg(loc, name)
		if !assert.NoError(t, err) {
			return
		}
		assert.Same(t, fd.MessageType[0], msg.DescriptorProto)
	}

	for _, loc := range []string{
		"." + testProtoPackage,
		testProtoPackage,
		".grpc.runtime",
		"grpc.runtime",
		".grpc",
		"grpc",
	} {
		name := "runtime.testpb.ExampleMessage"
		msg, err := reg.LookupMsg(loc, name)
		if !assert.NoError(t, err) {
			return
		}
		assert.Same(t, fd.MessageType[0], msg.DescriptorProto)
	}
}

func TestLoadWithInconsistentTargetPackage(t *testing.T) {
	for _, spec := range []struct {
		req        string
		consistent bool
	}{
		// root package, explicit go package
		{
			req: `
				file_to_generate: 'a.proto'
				file_to_generate: 'b.proto'
				proto_file <
					name: 'a.proto'
					options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
					message_type < name: 'A' >
					service <
						name: "AService"
						method <
							name: "Meth"
							input_type: "A"
							output_type: "A"
							options <
								[google.api.http] < post: "/v1/a" body: "*" >
							>
						>
					>
				>
				proto_file <
					name: 'b.proto'
					options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
					message_type < name: 'B' >
					service <
						name: "BService"
						method <
							name: "Meth"
							input_type: "B"
							output_type: "B"
							options <
								[google.api.http] < post: "/v1/b" body: "*" >
							>
						>
					>
				>
			`,
			consistent: true,
		},
		// named package, explicit go package
		{
			req: `
				file_to_generate: 'a.proto'
				file_to_generate: 'b.proto'
				proto_file <
					name: 'a.proto'
					package: 'grpc.runtime.testpb'
					options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
					message_type < name: 'A' >
					service <
						name: "AService"
						method <
							name: "Meth"
							input_type: "A"
							output_type: "A"
							options <
								[google.api.http] < post: "/v1/a" body: "*" >
							>
						>
					>
				>
				proto_file <
					name: 'b.proto'
					package: 'grpc.runtime.testpb'
					options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
					message_type < name: 'B' >
					service <
						name: "BService"
						method <
							name: "Meth"
							input_type: "B"
							output_type: "B"
							options <
								[google.api.http] < post: "/v1/b" body: "*" >
							>
						>
					>
				>
			`,
			consistent: true,
		},
	} {
		var req pluginpb.CodeGeneratorRequest
		if !assert.NoError(t, prototext.Unmarshal([]byte(spec.req), &req)) {
			return
		}
		_, err := newGeneratorFromSources(&req)
		if spec.consistent {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestLoadOverriddenPackageName(t *testing.T) {
	reg := NewRegistry()
	loadFile(t, reg, `
		name: 'example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'example.com/xyz;pb' >
	`)
	file := assertLoadedFile(t, reg, "example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: "example.com/xyz", Name: "pb"}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestLoadWithStandalone(t *testing.T) {
	reg := NewRegistry()
	reg.SetStandalone(true)
	loadFile(t, reg, `
		name: 'example.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'example.com/xyz;pb' >
	`)
	file := assertLoadedFile(t, reg, "example.proto")
	if file == nil {
		return
	}
	wantPkg := GoPackage{Path: "example.com/xyz", Name: "pb", Alias: "extPb"}
	assert.Equal(t, wantPkg, file.GoPkg)
}

func TestUnboundExternalHTTPRules(t *testing.T) {
	reg := NewRegistry()
	methodName := "." + testProtoPackage + ".ExampleService.Echo"
	reg.AddExternalHTTPRule(methodName, nil)
	assertStringSlice(t, "unbound external HTTP rules", reg.UnboundExternalHTTPRules(), []string{methodName})
	loadFile(t, reg, `
		name: "path/to/example.proto",
		package: "grpc.runtime.testpb"
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
		message_type <
			name: "StringMessage"
			field <
				name: "string"
				number: 1
				label: LABEL_OPTIONAL
				type: TYPE_STRING
			>
		>
		service <
			name: "ExampleService"
			method <
				name: "Echo"
				input_type: "StringMessage"
				output_type: "StringMessage"
			>
		>
	`)
	assertStringSlice(t, "unbound external HTTP rules", reg.UnboundExternalHTTPRules(), []string{})
}

func TestRegisterOpenAPIOptions(t *testing.T) {
	codeReqText := `file_to_generate: 'a.proto'
	proto_file <
		name: 'a.proto'
		package: 'grpc.runtime.testpb'
		options < go_package: 'github.com/kamalyes/grpc-runtime/testpb' >
		message_type <
			name: 'ExampleMessage'
			field <
				name: 'str'
				label: LABEL_OPTIONAL
				type: TYPE_STRING
				number: 1
			>
		>
		service <
			name: "AService"
			method <
				name: "Meth"
				input_type: "ExampleMessage"
				output_type: "ExampleMessage"
				options <
					[google.api.http] < post: "/v1/a" body: "*" >
				>
			>
		>
	>
	`
	var codeReq pluginpb.CodeGeneratorRequest
	if !assert.NoError(t, prototext.Unmarshal([]byte(codeReqText), &codeReq)) {
		return
	}

	for _, tcase := range []struct {
		options   *openapiconfig.OpenAPIOptions
		shouldErr bool
		desc      string
	}{
		{
			desc: "handle nil options",
		},
		{
			desc: "successfully add options if referenced entity exists",
			options: &openapiconfig.OpenAPIOptions{
				File: []*openapiconfig.OpenAPIFileOption{
					{
						File: "a.proto",
					},
				},
				Method: []*openapiconfig.OpenAPIMethodOption{
					{
						Method: "grpc.runtime.testpb.AService.Meth",
					},
				},
				Message: []*openapiconfig.OpenAPIMessageOption{
					{
						Message: "grpc.runtime.testpb.ExampleMessage",
					},
				},
				Service: []*openapiconfig.OpenAPIServiceOption{
					{
						Service: "grpc.runtime.testpb.AService",
					},
				},
				Field: []*openapiconfig.OpenAPIFieldOption{
					{
						Field: "grpc.runtime.testpb.ExampleMessage.str",
					},
				},
			},
		},
		{
			desc: "reject fully qualified names with leading \".\"",
			options: &openapiconfig.OpenAPIOptions{
				File: []*openapiconfig.OpenAPIFileOption{
					{
						File: "a.proto",
					},
				},
				Method: []*openapiconfig.OpenAPIMethodOption{
					{
						Method: ".grpc.runtime.testpb.AService.Meth",
					},
				},
				Message: []*openapiconfig.OpenAPIMessageOption{
					{
						Message: ".grpc.runtime.testpb.ExampleMessage",
					},
				},
				Service: []*openapiconfig.OpenAPIServiceOption{
					{
						Service: ".grpc.runtime.testpb.AService",
					},
				},
				Field: []*openapiconfig.OpenAPIFieldOption{
					{
						Field: ".grpc.runtime.testpb.ExampleMessage.str",
					},
				},
			},
			shouldErr: true,
		},
		{
			desc: "error if file does not exist",
			options: &openapiconfig.OpenAPIOptions{
				File: []*openapiconfig.OpenAPIFileOption{
					{
						File: "b.proto",
					},
				},
			},
			shouldErr: true,
		},
		{
			desc: "error if method does not exist",
			options: &openapiconfig.OpenAPIOptions{
				Method: []*openapiconfig.OpenAPIMethodOption{
					{
						Method: "grpc.runtime.testpb.AService.Meth2",
					},
				},
			},
			shouldErr: true,
		},
		{
			desc: "error if message does not exist",
			options: &openapiconfig.OpenAPIOptions{
				Message: []*openapiconfig.OpenAPIMessageOption{
					{
						Message: "grpc.runtime.testpb.NonexistentMessage",
					},
				},
			},
			shouldErr: true,
		},
		{
			desc: "error if service does not exist",
			options: &openapiconfig.OpenAPIOptions{
				Service: []*openapiconfig.OpenAPIServiceOption{
					{
						Service: "grpc.runtime.testpb.AService1",
					},
				},
			},
			shouldErr: true,
		},
		{
			desc: "error if field does not exist",
			options: &openapiconfig.OpenAPIOptions{
				Field: []*openapiconfig.OpenAPIFieldOption{
					{
						Field: "grpc.runtime.testpb.ExampleMessage.str1",
					},
				},
			},
			shouldErr: true,
		},
	} {
		t.Run(tcase.desc, func(t *testing.T) {
			reg := NewRegistry()
			loadFileWithCodeGeneratorRequest(t, reg, &codeReq)
			err := reg.RegisterOpenAPIOptions(tcase.options)
			if tcase.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func assertStringSlice(t *testing.T, message string, got, want []string) {
	if !assert.Len(t, got, len(want), message) {
		return
	}
	for i := range want {
		assert.Equal(t, want[i], got[i], message)
	}
}
