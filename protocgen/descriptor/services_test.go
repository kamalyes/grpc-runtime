package descriptor

import (
	"testing"

	"github.com/kamalyes/grpc-runtime/httprule"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func compilePath(t *testing.T, path string) httprule.Template {
	parsed, err := httprule.Parse(path)
	if !assert.NoError(t, err) {
		return httprule.Template{}
	}
	return parsed.Compile()
}

func testExtractServices(t *testing.T, input []*descriptorpb.FileDescriptorProto, target string, wantSvcs []*Service) {
	testExtractServicesWithRegistry(t, NewRegistry(), input, target, wantSvcs)
}

func testExtractServicesWithRegistry(t *testing.T, reg *Registry, input []*descriptorpb.FileDescriptorProto, target string, wantSvcs []*Service) {
	for _, file := range input {
		reg.loadFile(file.GetName(), &protogen.File{
			Proto: file,
		})
	}
	err := reg.loadServices(reg.files[target])
	if !assert.NoError(t, err) {
		return
	}

	file := reg.files[target]
	svcs := file.Services
	var i int
	for i = 0; i < len(svcs) && i < len(wantSvcs); i++ {
		svc, wantSvc := svcs[i], wantSvcs[i]
		if !assert.True(t, proto.Equal(svc.ServiceDescriptorProto, wantSvc.ServiceDescriptorProto)) {
			continue
		}
		var j int
		for j = 0; j < len(svc.Methods) && j < len(wantSvc.Methods); j++ {
			meth, wantMeth := svc.Methods[j], wantSvc.Methods[j]
			if !assert.True(t, proto.Equal(meth.MethodDescriptorProto, wantMeth.MethodDescriptorProto)) {
				continue
			}
			assert.Equal(t, wantMeth.RequestType.FQMN(), meth.RequestType.FQMN())
			assert.Equal(t, wantMeth.ResponseType.FQMN(), meth.ResponseType.FQMN())
			var k int
			for k = 0; k < len(meth.Bindings) && k < len(wantMeth.Bindings); k++ {
				binding, wantBinding := meth.Bindings[k], wantMeth.Bindings[k]
				assert.Equal(t, wantBinding.Index, binding.Index)
				assert.Equal(t, wantBinding.PathTmpl, binding.PathTmpl)
				assert.Equal(t, wantBinding.HTTPMethod, binding.HTTPMethod)

				var l int
				for l = 0; l < len(binding.PathParams) && l < len(wantBinding.PathParams); l++ {
					param, wantParam := binding.PathParams[l], wantBinding.PathParams[l]
					if !assert.Equal(t, wantParam.FieldPath.String(), param.FieldPath.String()) {
						continue
					}
					for m := 0; m < len(param.FieldPath) && m < len(wantParam.FieldPath); m++ {
						field, wantField := param.FieldPath[m].Target, wantParam.FieldPath[m].Target
						assert.True(t, proto.Equal(field.FieldDescriptorProto, wantField.FieldDescriptorProto))
					}
				}
				for ; l < len(binding.PathParams); l++ {
					assert.Failf(t, "unexpected path parameter", "svcs[%d].Methods[%d].Bindings[%d].PathParams[%d] = %q", i, j, k, l, binding.PathParams[l].FieldPath.String())
				}
				for ; l < len(wantBinding.PathParams); l++ {
					assert.Failf(t, "missing path parameter", "svcs[%d].Methods[%d].Bindings[%d].PathParams[%d] missing; want %q", i, j, k, l, wantBinding.PathParams[l].FieldPath.String())
				}

				if !assert.Equal(t, wantBinding.Body != nil, binding.Body != nil) {
					continue
				} else if binding.Body != nil {
					assert.Equal(t, wantBinding.Body.FieldPath.String(), binding.Body.FieldPath.String())
				}
			}
			for ; k < len(meth.Bindings); k++ {
				assert.Failf(t, "unexpected binding", "svcs[%d].Methods[%d].Bindings[%d] = %v", i, j, k, meth.Bindings[k])
			}
			for ; k < len(wantMeth.Bindings); k++ {
				assert.Failf(t, "missing binding", "svcs[%d].Methods[%d].Bindings[%d] missing; want %v", i, j, k, wantMeth.Bindings[k])
			}
		}
		for ; j < len(svc.Methods); j++ {
			assert.Failf(t, "unexpected method", "svcs[%d].Methods[%d] = %v", i, j, svc.Methods[j].MethodDescriptorProto)
		}
		for ; j < len(wantSvc.Methods); j++ {
			assert.Failf(t, "missing method", "svcs[%d].Methods[%d] missing; want %v", i, j, wantSvc.Methods[j].MethodDescriptorProto)
		}
	}
	for ; i < len(svcs); i++ {
		assert.Failf(t, "unexpected service", "svcs[%d] = %v", i, svcs[i].ServiceDescriptorProto)
	}
	for ; i < len(wantSvcs); i++ {
		assert.Failf(t, "missing service", "svcs[%d] missing; want %v", i, wantSvcs[i].ServiceDescriptorProto)
	}
}

func crossLinkFixture(f *File) *File {
	for _, m := range f.Messages {
		m.File = f
		for _, f := range m.Fields {
			f.Message = m
		}
	}
	for _, svc := range f.Services {
		svc.File = f
		for _, m := range svc.Methods {
			m.Service = svc
			for _, b := range m.Bindings {
				b.Method = m
				for _, param := range b.PathParams {
					param.Method = m
				}
			}
		}
	}
	for _, e := range f.Enums {
		e.File = f
	}
	return f
}

func TestExtractServicesSimple(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
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
				options <
					[google.api.http] <
						post: "/v1/example/echo"
						body: "*"
					>
				>
			>
		>
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fd.MessageType[0].Field[0],
			},
		},
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg: GoPackage{
			Path: "path/to/example.pb",
			Name: "example_pb",
		},
		Messages: []*Message{msg},
		Services: []*Service{
			{
				ServiceDescriptorProto: fd.Service[0],
				Methods: []*Method{
					{
						MethodDescriptorProto: fd.Service[0].Method[0],
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*Binding{
							{
								PathTmpl:   compilePath(t, "/v1/example/echo"),
								HTTPMethod: "POST",
								Body:       &Body{FieldPath: nil},
							},
						},
					},
				},
			},
		},
	}

	crossLinkFixture(file)
	testExtractServices(t, []*descriptorpb.FileDescriptorProto{&fd}, "path/to/example.proto", file.Services)
}

func TestExtractServicesWithoutAnnotation(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
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
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fd.MessageType[0].Field[0],
			},
		},
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg: GoPackage{
			Path: "path/to/example.pb",
			Name: "example_pb",
		},
		Messages: []*Message{msg},
		Services: []*Service{
			{
				ServiceDescriptorProto: fd.Service[0],
				Methods: []*Method{
					{
						MethodDescriptorProto: fd.Service[0].Method[0],
						RequestType:           msg,
						ResponseType:          msg,
					},
				},
			},
		},
	}

	crossLinkFixture(file)
	testExtractServices(t, []*descriptorpb.FileDescriptorProto{&fd}, "path/to/example.proto", file.Services)
}

func TestExtractServicesGenerateUnboundMethods(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
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
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fd.MessageType[0].Field[0],
			},
		},
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg: GoPackage{
			Path: "path/to/example.pb",
			Name: "example_pb",
		},
		Messages: []*Message{msg},
		Services: []*Service{
			{
				ServiceDescriptorProto: fd.Service[0],
				Methods: []*Method{
					{
						MethodDescriptorProto: fd.Service[0].Method[0],
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*Binding{
							{
								PathTmpl:   compilePath(t, "/example.ExampleService/Echo"),
								HTTPMethod: "POST",
								Body:       &Body{FieldPath: nil},
							},
						},
					},
				},
			},
		},
	}

	crossLinkFixture(file)
	reg := NewRegistry()
	reg.SetGenerateUnboundMethods(true)
	testExtractServicesWithRegistry(t, reg, []*descriptorpb.FileDescriptorProto{&fd}, "path/to/example.proto", file.Services)
}

func TestExtractServicesCrossPackage(t *testing.T) {
	srcs := []string{
		`
			name: "path/to/example.proto",
			package: "example"
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
					name: "ToString"
					input_type: ".another.example.BoolMessage"
					output_type: "StringMessage"
					options <
						[google.api.http] <
							post: "/v1/example/to_s"
							body: "*"
						>
					>
				>
			>
		`, `
			name: "path/to/another/example.proto",
			package: "another.example"
			message_type <
				name: "BoolMessage"
				field <
					name: "bool"
					number: 1
					label: LABEL_OPTIONAL
					type: TYPE_BOOL
				>
			>
		`,
	}
	var fds []*descriptorpb.FileDescriptorProto
	for _, src := range srcs {
		var fd descriptorpb.FileDescriptorProto
		if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
			return
		}
		fds = append(fds, &fd)
	}
	stringMsg := &Message{
		DescriptorProto: fds[0].MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fds[0].MessageType[0].Field[0],
			},
		},
	}
	boolMsg := &Message{
		DescriptorProto: fds[1].MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fds[1].MessageType[0].Field[0],
			},
		},
	}
	files := []*File{
		{
			FileDescriptorProto: fds[0],
			GoPkg: GoPackage{
				Path: "path/to/example.pb",
				Name: "example_pb",
			},
			Messages: []*Message{stringMsg},
			Services: []*Service{
				{
					ServiceDescriptorProto: fds[0].Service[0],
					Methods: []*Method{
						{
							MethodDescriptorProto: fds[0].Service[0].Method[0],
							RequestType:           boolMsg,
							ResponseType:          stringMsg,
							Bindings: []*Binding{
								{
									PathTmpl:   compilePath(t, "/v1/example/to_s"),
									HTTPMethod: "POST",
									Body:       &Body{FieldPath: nil},
								},
							},
						},
					},
				},
			},
		},
		{
			FileDescriptorProto: fds[1],
			GoPkg: GoPackage{
				Path: "path/to/another/example.pb",
				Name: "example_pb",
			},
			Messages: []*Message{boolMsg},
		},
	}

	for _, file := range files {
		crossLinkFixture(file)
	}
	testExtractServices(t, fds, "path/to/example.proto", files[0].Services)
}

func TestExtractServicesWithBodyPath(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
		message_type <
			name: "OuterMessage"
			nested_type <
				name: "StringMessage"
				field <
					name: "string"
					number: 1
					label: LABEL_OPTIONAL
					type: TYPE_STRING
				>
			>
			field <
				name: "nested"
				number: 1
				label: LABEL_OPTIONAL
				type: TYPE_MESSAGE
				type_name: "StringMessage"
			>
		>
		service <
			name: "ExampleService"
			method <
				name: "Echo"
				input_type: "OuterMessage"
				output_type: "OuterMessage"
				options <
					[google.api.http] <
						post: "/v1/example/echo"
						body: "nested"
					>
				>
			>
		>
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fd.MessageType[0].Field[0],
			},
		},
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg: GoPackage{
			Path: "path/to/example.pb",
			Name: "example_pb",
		},
		Messages: []*Message{msg},
		Services: []*Service{
			{
				ServiceDescriptorProto: fd.Service[0],
				Methods: []*Method{
					{
						MethodDescriptorProto: fd.Service[0].Method[0],
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*Binding{
							{
								PathTmpl:   compilePath(t, "/v1/example/echo"),
								HTTPMethod: "POST",
								Body: &Body{
									FieldPath: FieldPath{
										{
											Name:   "nested",
											Target: msg.Fields[0],
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	crossLinkFixture(file)
	testExtractServices(t, []*descriptorpb.FileDescriptorProto{&fd}, "path/to/example.proto", file.Services)
}

func TestExtractServicesWithPathParam(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
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
				options <
					[google.api.http] <
						get: "/v1/example/echo/{string=*}"
					>
				>
			>
		>
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fd.MessageType[0].Field[0],
			},
		},
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg: GoPackage{
			Path: "path/to/example.pb",
			Name: "example_pb",
		},
		Messages: []*Message{msg},
		Services: []*Service{
			{
				ServiceDescriptorProto: fd.Service[0],
				Methods: []*Method{
					{
						MethodDescriptorProto: fd.Service[0].Method[0],
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*Binding{
							{
								PathTmpl:   compilePath(t, "/v1/example/echo/{string=*}"),
								HTTPMethod: "GET",
								PathParams: []Parameter{
									{
										FieldPath: FieldPath{
											{
												Name:   "string",
												Target: msg.Fields[0],
											},
										},
										Target: msg.Fields[0],
									},
								},
							},
						},
					},
				},
			},
		},
	}

	crossLinkFixture(file)
	testExtractServices(t, []*descriptorpb.FileDescriptorProto{&fd}, "path/to/example.proto", file.Services)
}

func TestExtractServicesWithAdditionalBinding(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
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
				options <
					[google.api.http] <
						post: "/v1/example/echo"
						body: "*"
						additional_bindings <
							get: "/v1/example/echo/{string}"
						>
						additional_bindings <
							post: "/v2/example/echo"
							body: "string"
						>
					>
				>
			>
		>
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	msg := &Message{
		DescriptorProto: fd.MessageType[0],
		Fields: []*Field{
			{
				FieldDescriptorProto: fd.MessageType[0].Field[0],
			},
		},
	}
	file := &File{
		FileDescriptorProto: &fd,
		GoPkg: GoPackage{
			Path: "path/to/example.pb",
			Name: "example_pb",
		},
		Messages: []*Message{msg},
		Services: []*Service{
			{
				ServiceDescriptorProto: fd.Service[0],
				Methods: []*Method{
					{
						MethodDescriptorProto: fd.Service[0].Method[0],
						RequestType:           msg,
						ResponseType:          msg,
						Bindings: []*Binding{
							{
								Index:      0,
								PathTmpl:   compilePath(t, "/v1/example/echo"),
								HTTPMethod: "POST",
								Body:       &Body{FieldPath: nil},
							},
							{
								Index:      1,
								PathTmpl:   compilePath(t, "/v1/example/echo/{string}"),
								HTTPMethod: "GET",
								PathParams: []Parameter{
									{
										FieldPath: FieldPath{
											{
												Name:   "string",
												Target: msg.Fields[0],
											},
										},
										Target: msg.Fields[0],
									},
								},
								Body: nil,
							},
							{
								Index:      2,
								PathTmpl:   compilePath(t, "/v2/example/echo"),
								HTTPMethod: "POST",
								Body: &Body{
									FieldPath: FieldPath{
										FieldPathComponent{
											Name:   "string",
											Target: msg.Fields[0],
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	crossLinkFixture(file)
	testExtractServices(t, []*descriptorpb.FileDescriptorProto{&fd}, "path/to/example.proto", file.Services)
}

func TestExtractServicesWithError(t *testing.T) {
	for _, spec := range []struct {
		target string
		srcs   []string
	}{
		{
			target: "path/to/example.proto",
			srcs: []string{
				// message not found
				`
					name: "path/to/example.proto",
					package: "example"
					service <
						name: "ExampleService"
						method <
							name: "Echo"
							input_type: "StringMessage"
							output_type: "StringMessage"
							options <
								[google.api.http] <
									post: "/v1/example/echo"
									body: "*"
								>
							>
						>
					>
				`,
			},
		},
		// body field path not resolved
		{
			target: "path/to/example.proto",
			srcs: []string{`
						name: "path/to/example.proto",
						package: "example"
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
								options <
									[google.api.http] <
										post: "/v1/example/echo"
										body: "bool"
									>
								>
							>
						>`,
			},
		},
		// param field path not resolved
		{
			target: "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
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
							options <
								[google.api.http] <
									post: "/v1/example/echo/{bool=*}"
								>
							>
						>
					>
				`,
			},
		},
		// non aggregate type on field path
		{
			target: "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
					message_type <
						name: "OuterMessage"
						field <
							name: "mid"
							number: 1
							label: LABEL_OPTIONAL
							type: TYPE_STRING
						>
						field <
							name: "bool"
							number: 2
							label: LABEL_OPTIONAL
							type: TYPE_BOOL
						>
					>
					service <
						name: "ExampleService"
						method <
							name: "Echo"
							input_type: "OuterMessage"
							output_type: "OuterMessage"
							options <
								[google.api.http] <
									post: "/v1/example/echo/{mid.bool=*}"
								>
							>
						>
					>
				`,
			},
		},
		// path param in client streaming
		{
			target: "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
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
							options <
								[google.api.http] <
									post: "/v1/example/echo/{bool=*}"
								>
							>
							client_streaming: true
						>
					>
				`,
			},
		},
		// body for GET
		{
			target: "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
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
							options <
								[google.api.http] <
									get: "/v1/example/echo"
									body: "string"
								>
							>
						>
					>
				`,
			},
		},
		// body for DELETE
		{
			target: "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
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
							name: "RemoveResource"
							input_type: "StringMessage"
							output_type: "StringMessage"
							options <
								[google.api.http] <
									delete: "/v1/example/resource"
									body: "string"
								>
							>
						>
					>
				`,
			},
		},
		// no pattern specified
		{
			target: "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
					service <
						name: "ExampleService"
						method <
							name: "RemoveResource"
							input_type: "StringMessage"
							output_type: "StringMessage"
							options <
								[google.api.http] <
									body: "string"
								>
							>
						>
					>
				`,
			},
		},
		// unsupported path parameter type
		{
			target: "path/to/example.proto",
			srcs: []string{`
					name: "path/to/example.proto",
					package: "example"
					message_type <
						name: "OuterMessage"
						nested_type <
							name: "StringMessage"
							field <
								name: "value"
								number: 1
								label: LABEL_OPTIONAL
								type: TYPE_STRING
							>
						>
						field <
							name: "string"
							number: 1
							label: LABEL_OPTIONAL
							type: TYPE_MESSAGE
							type_name: "StringMessage"
						>
					>
					service <
						name: "ExampleService"
						method <
							name: "Echo"
							input_type: "OuterMessage"
							output_type: "OuterMessage"
							options <
								[google.api.http] <
									get: "/v1/example/echo/{string=*}"
								>
							>
						>
					>
				`,
			},
		},
	} {
		reg := NewRegistry()

		for _, src := range spec.srcs {
			var fd descriptorpb.FileDescriptorProto
			if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
				return
			}
			reg.loadFile(spec.target, &protogen.File{
				Proto: &fd,
			})
		}
		err := reg.loadServices(reg.files[spec.target])
		assert.Error(t, err)
		t.Log(err)
	}
}

func TestResolveFieldPath(t *testing.T) {
	for _, spec := range []struct {
		src     string
		path    string
		wantErr bool
	}{
		{
			src: `
				name: 'example.proto'
				package: 'example'
				message_type <
					name: 'ExampleMessage'
					field <
						name: 'string'
						type: TYPE_STRING
						label: LABEL_OPTIONAL
						number: 1
					>
				>
			`,
			path:    "string",
			wantErr: false,
		},
		// no such field
		{
			src: `
				name: 'example.proto'
				package: 'example'
				message_type <
					name: 'ExampleMessage'
					field <
						name: 'string'
						type: TYPE_STRING
						label: LABEL_OPTIONAL
						number: 1
					>
				>
			`,
			path:    "something_else",
			wantErr: true,
		},
		// repeated field
		{
			src: `
				name: 'example.proto'
				package: 'example'
				message_type <
					name: 'ExampleMessage'
					field <
						name: 'string'
						type: TYPE_STRING
						label: LABEL_REPEATED
						number: 1
					>
				>
			`,
			path:    "string",
			wantErr: false,
		},
		// nested field
		{
			src: `
				name: 'example.proto'
				package: 'example'
				message_type <
					name: 'ExampleMessage'
					field <
						name: 'nested'
						type: TYPE_MESSAGE
						type_name: 'AnotherMessage'
						label: LABEL_OPTIONAL
						number: 1
					>
					field <
						name: 'terminal'
						type: TYPE_BOOL
						label: LABEL_OPTIONAL
						number: 2
					>
				>
				message_type <
					name: 'AnotherMessage'
					field <
						name: 'nested2'
						type: TYPE_MESSAGE
						type_name: 'ExampleMessage'
						label: LABEL_OPTIONAL
						number: 1
					>
				>
			`,
			path:    "nested.nested2.nested.nested2.nested.nested2.terminal",
			wantErr: false,
		},
		// non aggregate field on the path
		{
			src: `
				name: 'example.proto'
				package: 'example'
				message_type <
					name: 'ExampleMessage'
					field <
						name: 'nested'
						type: TYPE_MESSAGE
						type_name: 'AnotherMessage'
						label: LABEL_OPTIONAL
						number: 1
					>
					field <
						name: 'terminal'
						type: TYPE_BOOL
						label: LABEL_OPTIONAL
						number: 2
					>
				>
				message_type <
					name: 'AnotherMessage'
					field <
						name: 'nested2'
						type: TYPE_MESSAGE
						type_name: 'ExampleMessage'
						label: LABEL_OPTIONAL
						number: 1
					>
				>
			`,
			path:    "nested.terminal.nested2",
			wantErr: true,
		},
		// repeated field
		{
			src: `
				name: 'example.proto'
				package: 'example'
				message_type <
					name: 'ExampleMessage'
					field <
						name: 'nested'
						type: TYPE_MESSAGE
						type_name: 'AnotherMessage'
						label: LABEL_OPTIONAL
						number: 1
					>
					field <
						name: 'terminal'
						type: TYPE_BOOL
						label: LABEL_OPTIONAL
						number: 2
					>
				>
				message_type <
					name: 'AnotherMessage'
					field <
						name: 'nested2'
						type: TYPE_MESSAGE
						type_name: 'ExampleMessage'
						label: LABEL_REPEATED
						number: 1
					>
				>
			`,
			path:    "nested.nested2.terminal",
			wantErr: false,
		},
	} {
		var file descriptorpb.FileDescriptorProto
		if !assert.NoError(t, prototext.Unmarshal([]byte(spec.src), &file)) {
			return
		}
		reg := NewRegistry()
		reg.loadFile(file.GetName(), &protogen.File{
			Proto: &file,
		})
		f, err := reg.LookupFile(file.GetName())
		if !assert.NoError(t, err) {
			return
		}
		_, err = reg.resolveFieldPath(f.Messages[0], spec.path, false)
		if spec.wantErr {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
	}
}

func TestExtractServicesWithDeleteBody(t *testing.T) {
	for _, spec := range []struct {
		allowDeleteBody bool
		expectErr       bool
		target          string
		srcs            []string
	}{
		// body for DELETE, but registry configured to allow it
		{
			allowDeleteBody: true,
			expectErr:       false,
			target:          "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
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
							name: "RemoveResource"
							input_type: "StringMessage"
							output_type: "StringMessage"
							options <
								[google.api.http] <
									delete: "/v1/example/resource"
									body: "string"
								>
							>
						>
					>
				`,
			},
		},
		// body for DELETE, registry configured not to allow it
		{
			allowDeleteBody: false,
			expectErr:       true,
			target:          "path/to/example.proto",
			srcs: []string{
				`
					name: "path/to/example.proto",
					package: "example"
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
							name: "RemoveResource"
							input_type: "StringMessage"
							output_type: "StringMessage"
							options <
								[google.api.http] <
									delete: "/v1/example/resource"
									body: "string"
								>
							>
						>
					>
				`,
			},
		},
	} {
		reg := NewRegistry()
		reg.SetAllowDeleteBody(spec.allowDeleteBody)

		for _, src := range spec.srcs {
			var fd descriptorpb.FileDescriptorProto
			if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
				return
			}
			reg.loadFile(fd.GetName(), &protogen.File{
				Proto: &fd,
			})
		}
		err := reg.loadServices(reg.files[spec.target])
		if spec.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		t.Log(err)
	}
}

func TestCauseErrorWithPathParam(t *testing.T) {
	src := `
		name: "path/to/example.proto",
		package: "example"
		message_type <
			name: "TypeMessage"
			field <
					name: "message"
					type: TYPE_MESSAGE
					type_name: 'ExampleMessage'
					number: 1,
					label: LABEL_OPTIONAL
				>
		>
		service <
			name: "ExampleService"
			method <
				name: "Echo"
				input_type: "TypeMessage"
				output_type: "TypeMessage"
				options <
					[google.api.http] <
						get: "/v1/example/echo/{message=*}"
					>
				>
			>
		>
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	target := "path/to/example.proto"
	reg := NewRegistry()
	reg.loadFile(fd.GetName(), &protogen.File{
		Proto: &fd,
	})
	// switch this field to see the error
	wantErr := true
	err := reg.loadServices(reg.files[target])
	if wantErr {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func TestOptionalProto3URLPathMappingError(t *testing.T) {
	src := `
		name: "path/to/example.proto"
		package: "example"
		message_type <
			name: "StringMessage"
			field <
				name: "field1"
				number: 1
				type: TYPE_STRING
				proto3_optional: true
			>
		>
		service <
			name: "ExampleService"
			method <
				name: "Echo"
				input_type: "StringMessage"
				output_type: "StringMessage"
				options <
					[google.api.http] <
						get: "/v1/example/echo/{field1=*}"
					>
				>
			>
		>
	`
	var fd descriptorpb.FileDescriptorProto
	if !assert.NoError(t, prototext.Unmarshal([]byte(src), &fd)) {
		return
	}
	target := "path/to/example.proto"
	reg := NewRegistry()
	reg.loadFile(fd.GetName(), &protogen.File{
		Proto: &fd,
	})
	wantErrMsg := "field not allowed in field path: field1 in field1"
	err := reg.loadServices(reg.files[target])
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), wantErrMsg)
	}
}
