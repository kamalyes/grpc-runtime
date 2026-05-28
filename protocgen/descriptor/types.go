/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:22:16
 * @FilePath: \grpc-runtime\protocgen\descriptor\types.go
 * @Description: proto 描述符核心类型定义
 * 包含 File/Message/Enum/Service/Method/Binding/Field/Parameter/Body 等类型
 * 以及 FieldPath 表达式生成逻辑和 proto 类型转换函数映射表
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package descriptor

import (
	"fmt"
	"strings"

	"github.com/kamalyes/grpc-runtime/httprule"
	"github.com/kamalyes/grpc-runtime/protocgen/naming"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ============================================================================ //
// Go 包信息与响应文件
// ============================================================================ //

// GoPackage 表示 Go 包信息
type GoPackage struct {
	// Path 包导入路径
	Path string
	// Name 包名
	Name string
	// Alias 包别名（用于解决包名冲突）
	Alias string
}

// Standard 判断是否为 Go 标准库包
func (p GoPackage) Standard() bool {
	return !strings.Contains(p.Path, ".")
}

// String 返回 Go import 行格式的字符串表示
func (p GoPackage) String() string {
	if p.Alias == "" {
		return fmt.Sprintf("%q", p.Path)
	}
	return fmt.Sprintf("%s %q", p.Alias, p.Path)
}

// ResponseFile 包装 pluginpb.CodeGeneratorResponse_File，附加 Go 包信息
type ResponseFile struct {
	*pluginpb.CodeGeneratorResponse_File
	// GoPkg 生成文件所属的 Go 包
	GoPkg GoPackage
}

// ============================================================================ //
// 文件描述符
// ============================================================================ //

// File 包装 descriptorpb.FileDescriptorProto，提供更丰富的功能
type File struct {
	*descriptorpb.FileDescriptorProto
	// GoPkg 生成 Go 文件的包信息
	GoPkg GoPackage
	// GeneratedFilenamePrefix 生成文件名前缀
	// 例如源文件 "dir/foo.proto" 的前缀为 "dir/foo"
	GeneratedFilenamePrefix string
	// Messages 文件中定义的消息列表
	Messages []*Message
	// Enums 文件中定义的枚举列表
	Enums []*Enum
	// Services 文件中定义的服务列表
	Services []*Service
}

// Pkg 返回包名或别名（如果存在别名则优先使用别名）
func (f *File) Pkg() string {
	pkg := f.GoPkg.Name
	if alias := f.GoPkg.Alias; alias != "" {
		pkg = alias
	}
	return pkg
}

// proto2 判断文件语法是否为 proto2
func (f *File) proto2() bool {
	return f.Syntax == nil || f.GetSyntax() == "proto2"
}

// ============================================================================ //
// 消息描述符
// ============================================================================ //

// Message 描述 protocol buffer 消息类型
type Message struct {
	*descriptorpb.DescriptorProto
	// File 消息所在的文件
	File *File
	// Outers 嵌套类型的外层消息名列表
	Outers []string
	// Fields 消息的字段列表
	Fields []*Field
	// Index 消息在文件中的 proto 路径索引
	Index int
	// ForcePrefixedName 是否强制使用包前缀
	ForcePrefixedName bool
}

// FQMN 返回消息的全限定名（Fully Qualified Message Name）
func (m *Message) FQMN() string {
	components := make([]string, 0, len(m.Outers)+3)
	components = append(components, "")
	if m.File.Package != nil {
		components = append(components, m.File.GetPackage())
	}
	components = append(components, m.Outers...)
	components = append(components, m.GetName())
	return strings.Join(components, ".")
}

// GoType 返回消息的 Go 类型名
// 如果消息不属于当前包则添加包前缀
func (m *Message) GoType(currentPackage string) string {
	var components []string
	components = append(components, m.Outers...)
	components = append(components, m.GetName())

	name := strings.Join(components, "_")
	if !m.ForcePrefixedName && m.File.GoPkg.Path == currentPackage {
		return name
	}
	return fmt.Sprintf("%s.%s", m.File.Pkg(), name)
}

// ============================================================================ //
// 枚举描述符
// ============================================================================ //

// Enum 描述 protocol buffer 枚举类型
type Enum struct {
	*descriptorpb.EnumDescriptorProto
	// File 枚举所在的文件
	File *File
	// Outers 嵌套类型的外层消息名列表
	Outers []string
	// Index 枚举索引值
	Index int
	// ForcePrefixedName 是否强制使用包前缀
	ForcePrefixedName bool
}

// FQEN 返回枚举的全限定名（Fully Qualified Enum Name）
func (e *Enum) FQEN() string {
	components := make([]string, 0, len(e.Outers)+3)
	components = append(components, "")
	if e.File.Package != nil {
		components = append(components, e.File.GetPackage())
	}
	components = append(components, e.Outers...)
	components = append(components, e.GetName())
	return strings.Join(components, ".")
}

// GoType 返回枚举的 Go 类型名
// 如果枚举不属于当前包则添加包前缀
func (e *Enum) GoType(currentPackage string) string {
	var components []string
	components = append(components, e.Outers...)
	components = append(components, e.GetName())

	name := strings.Join(components, "_")
	if !e.ForcePrefixedName && e.File.GoPkg.Path == currentPackage {
		return name
	}
	return fmt.Sprintf("%s.%s", e.File.Pkg(), name)
}

// ============================================================================ //
// 服务描述符
// ============================================================================ //

// Service 包装 descriptorpb.ServiceDescriptorProto
type Service struct {
	*descriptorpb.ServiceDescriptorProto
	// File 服务所在的文件
	File *File
	// Methods 服务中定义的方法列表
	Methods []*Method
	// ForcePrefixedName 是否强制使用包前缀
	ForcePrefixedName bool
}

// FQSN 返回服务的全限定名（Fully Qualified Service Name）
func (s *Service) FQSN() string {
	components := make([]string, 0, 3)
	components = append(components, "")
	if s.File.Package != nil {
		components = append(components, s.File.GetPackage())
	}
	components = append(components, s.GetName())
	return strings.Join(components, ".")
}

// InstanceName 返回服务的实例名，需要时添加包前缀
func (s *Service) InstanceName() string {
	if !s.ForcePrefixedName {
		return s.GetName()
	}
	return fmt.Sprintf("%s.%s", s.File.Pkg(), s.GetName())
}

// ClientConstructorName 返回客户端构造函数名，需要时添加包前缀
func (s *Service) ClientConstructorName() string {
	constructor := "New" + s.GetName() + "Client"
	if !s.ForcePrefixedName {
		return constructor
	}
	return fmt.Sprintf("%s.%s", s.File.Pkg(), constructor)
}

// ============================================================================ //
// 方法描述符
// ============================================================================ //

// Method 包装 descriptorpb.MethodDescriptorProto
type Method struct {
	*descriptorpb.MethodDescriptorProto
	// Service 方法所属的服务
	Service *Service
	// RequestType 请求消息类型
	RequestType *Message
	// ResponseType 响应消息类型
	ResponseType *Message
	// Bindings HTTP 端点绑定列表
	Bindings []*Binding
}

// FQMN 返回方法的全限定名（Fully Qualified Method Name）
func (m *Method) FQMN() string {
	components := make([]string, 0, 2)
	components = append(components, m.Service.FQSN())
	components = append(components, m.GetName())
	return strings.Join(components, ".")
}

// ============================================================================ //
// HTTP 绑定描述符
// ============================================================================ //

// Binding 描述 HTTP 端点与 gRPC 方法的绑定关系
type Binding struct {
	// Method 绑定所属的方法
	Method *Method
	// Index 绑定在方法中的零起始索引
	Index int
	// PathTmpl 路径模板
	PathTmpl httprule.Template
	// HTTPMethod HTTP 方法（GET/POST/PUT/DELETE/PATCH 等）
	HTTPMethod string
	// PathParams 路径参数列表
	PathParams []Parameter
	// Body 请求体绑定描述
	Body *Body
	// ResponseBody 响应体绑定描述
	ResponseBody *Body
}

// ExplicitParams 返回绑定的显式参数列表
// 即 body 字段路径与 path 参数字段路径的并集
func (b *Binding) ExplicitParams() []string {
	var result []string
	if b.Body != nil {
		result = append(result, b.Body.FieldPath.String())
	}
	for _, p := range b.PathParams {
		result = append(result, p.FieldPath.String())
	}
	return result
}

// ============================================================================ //
// 字段描述符
// ============================================================================ //

// Field 包装 descriptorpb.FieldDescriptorProto
type Field struct {
	*descriptorpb.FieldDescriptorProto
	// Message 字段所属的消息
	Message *Message
	// FieldMessage 字段对应的消息类型（仅对消息类型字段有效）
	FieldMessage *Message
	// ForcePrefixedName 是否强制使用包前缀
	ForcePrefixedName bool
}

// FQFN 返回字段的全限定名（Fully Qualified Field Name）
func (f *Field) FQFN() string {
	return strings.Join([]string{f.Message.FQMN(), f.GetName()}, ".")
}

// ============================================================================ //
// 参数与请求体描述符
// ============================================================================ //

// Parameter 描述 HTTP 请求中的参数
type Parameter struct {
	// FieldPath 参数映射到的 proto 字段路径
	FieldPath
	// Target 参数映射到的 proto 字段
	Target *Field
	// Method 使用此参数的方法
	Method *Method
}

// ConvertFuncExpr 返回参数转换函数的 Go 表达式
// 根据字段的 proto 版本和修饰符选择对应的转换函数
func (p Parameter) ConvertFuncExpr() (string, error) {
	tbl := proto3ConvertFuncs
	if !p.IsProto2() && p.IsRepeated() {
		tbl = proto3RepeatedConvertFuncs
	} else if !p.IsProto2() && p.IsOptionalProto3() {
		tbl = proto3OptionalConvertFuncs
	} else if p.IsProto2() && !p.IsRepeated() {
		tbl = proto2ConvertFuncs
	} else if p.IsProto2() && p.IsRepeated() {
		tbl = proto2RepeatedConvertFuncs
	}
	typ := p.Target.GetType()
	conv, ok := tbl[typ]
	if !ok {
		conv, ok = wellKnownTypeConv[p.Target.GetTypeName()]
	}
	if !ok {
		return "", fmt.Errorf("unsupported field type %s of parameter %s in %s.%s", typ, p.FieldPath, p.Method.Service.GetName(), p.Method.GetName())
	}
	return conv, nil
}

// IsEnum 判断参数字段是否为枚举类型
func (p Parameter) IsEnum() bool {
	return p.Target.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
}

// IsRepeated 判断参数字段是否为 repeated
func (p Parameter) IsRepeated() bool {
	return p.Target.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED
}

// IsProto2 判断参数字段是否属于 proto2 语法
func (p Parameter) IsProto2() bool {
	return p.Target.Message.File.proto2()
}

// Body 描述 HTTP 请求/响应体的绑定关系
// 对应 google.api.HttpRule 中的 body 和 response_body 选项
type Body struct {
	// FieldPath body 映射到的 proto 字段路径
	// 为空时表示整个消息就是 body
	FieldPath FieldPath
}

// AssignableExpr 返回 body 的 Go 赋值表达式
func (b Body) AssignableExpr(msgExpr string, currentPackage string) string {
	return b.FieldPath.AssignableExpr(msgExpr, currentPackage)
}

// AssignableExprPrep 返回 body 赋值表达式的准备语句
func (b Body) AssignableExprPrep(msgExpr string, currentPackage string) string {
	return b.FieldPath.AssignableExprPrep(msgExpr, currentPackage)
}

// ============================================================================ //
// 字段路径（FieldPath）
// ============================================================================ //

// FieldPath 表示从请求消息到目标字段的路径
// 每个元素是路径中的一个字段组件
type FieldPath []FieldPathComponent

// String 返回字段路径的字符串表示，以 "." 分隔
func (p FieldPath) String() string {
	if len(p) == 0 {
		return ""
	}
	components := make([]string, 0, len(p))
	for _, c := range p {
		components = append(components, c.Name)
	}
	return strings.Join(components, ".")
}

// IsNestedProto3 判断 FieldPath 是否为嵌套的 Proto3 路径
func (p FieldPath) IsNestedProto3() bool {
	return len(p) > 1 && !p[0].Target.Message.File.proto2()
}

// IsOptionalProto3 判断 FieldPath 是否为 proto3 optional 字段
func (p FieldPath) IsOptionalProto3() bool {
	if len(p) == 0 {
		return false
	}
	return p[0].Target.GetProto3Optional()
}

// AssignableExpr 返回 Go 赋值表达式，用于将值赋给目标字段
// msgExpr 是请求消息的 Go 表达式，currentPackage 是当前包路径
// 对于包含 oneof 的字段路径，需要先调用 AssignableExprPrep 生成准备语句
func (p FieldPath) AssignableExpr(msgExpr string, currentPackage string) string {
	l := len(p)
	if l == 0 {
		return msgExpr
	}

	components := msgExpr
	for i, c := range p {
		if !c.Target.GetProto3Optional() && c.Target.OneofIndex != nil {
			index := c.Target.OneofIndex
			msg := c.Target.Message
			oneOfName := naming.Camel(msg.GetOneofDecl()[*index].GetName())
			oneofFieldName := msg.GoType(currentPackage) + "_" + c.AssignableExpr()

			if c.Target.ForcePrefixedName {
				oneofFieldName = msg.File.Pkg() + "." + msg.GetName() + "_" + c.AssignableExpr()
			}

			components = components + "." + oneOfName + ".(*" + oneofFieldName + ")"
		}

		if i == l-1 {
			components = components + "." + c.AssignableExpr()
			continue
		}
		components = components + "." + c.ValueExpr()
	}
	return components
}

// AssignableExprPrep 返回赋值表达式的准备语句
// 仅在字段路径包含 oneof 时需要，否则返回空字符串
func (p FieldPath) AssignableExprPrep(msgExpr string, currentPackage string) string {
	l := len(p)
	if l == 0 {
		return ""
	}

	var preparations []string
	components := msgExpr
	for i, c := range p {
		if !c.Target.GetProto3Optional() && c.Target.OneofIndex != nil {
			index := c.Target.OneofIndex
			msg := c.Target.Message
			oneOfName := naming.Camel(msg.GetOneofDecl()[*index].GetName())
			oneofFieldName := msg.GoType(currentPackage) + "_" + c.AssignableExpr()

			if c.Target.ForcePrefixedName {
				oneofFieldName = msg.File.Pkg() + "." + msg.GetName() + "_" + c.AssignableExpr()
			}

			components = components + "." + oneOfName
			s := `if %s == nil {
				%s =&%s{}
			} else if _, ok := %s.(*%s); !ok {
				return nil, metadata, status.Errorf(codes.InvalidArgument, "expect type: *%s, but: %%t\n",%s)
			}`

			preparations = append(preparations, fmt.Sprintf(s, components, components, oneofFieldName, components, oneofFieldName, oneofFieldName, components))
			components = components + ".(*" + oneofFieldName + ")"
		}

		if i == l-1 {
			components = components + "." + c.AssignableExpr()
			continue
		}
		components = components + "." + c.ValueExpr()
	}

	return strings.Join(preparations, "\n")
}

// FieldPathComponent 字段路径中的单个组件
type FieldPathComponent struct {
	// Name proto 字段名
	Name string
	// Target 对应的 proto 字段描述
	Target *Field
}

// AssignableExpr 返回此字段的 Go 赋值表达式（CamelCase）
func (c FieldPathComponent) AssignableExpr() string {
	return naming.Camel(c.Name)
}

// ValueExpr 返回此字段的 Go 取值表达式
// proto2 使用 Get 方法，proto3 直接访问字段
func (c FieldPathComponent) ValueExpr() string {
	if c.Target.Message.File.proto2() {
		return fmt.Sprintf("Get%s()", naming.Camel(c.Name))
	}
	return naming.Camel(c.Name)
}

// ============================================================================ //
// proto 类型转换函数映射表
// ============================================================================ //

// proto3ConvertFuncs proto3 标量字段到 runtime 转换函数的映射
var proto3ConvertFuncs = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   "runtime.Float64",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    "runtime.Float32",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    "runtime.Int64",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   "runtime.Uint64",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    "runtime.Int32",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "runtime.Uint64",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "runtime.Uint32",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     "runtime.Bool",
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   "runtime.String",
	descriptorpb.FieldDescriptorProto_TYPE_BYTES:    "runtime.Bytes",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   "runtime.Uint32",
	descriptorpb.FieldDescriptorProto_TYPE_ENUM:     "runtime.Enum",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "runtime.Int32",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "runtime.Int64",
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "runtime.Int32",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "runtime.Int64",
}

// proto3OptionalConvertFuncs proto3 optional 字段到 runtime 指针转换函数的映射
// 在 proto3 转换函数名后追加 "P" 表示返回指针类型
var proto3OptionalConvertFuncs = func() map[descriptorpb.FieldDescriptorProto_Type]string {
	result := make(map[descriptorpb.FieldDescriptorProto_Type]string, len(proto3ConvertFuncs))
	for typ, converter := range proto3ConvertFuncs {
		result[typ] = converter + "P"
	}
	return result
}()

// proto3RepeatedConvertFuncs proto3 repeated 字段到 runtime 切片转换函数的映射
var proto3RepeatedConvertFuncs = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   "runtime.Float64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    "runtime.Float32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    "runtime.Int64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   "runtime.Uint64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    "runtime.Int32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "runtime.Uint64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "runtime.Uint32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     "runtime.BoolSlice",
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   "runtime.StringSlice",
	descriptorpb.FieldDescriptorProto_TYPE_BYTES:    "runtime.BytesSlice",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   "runtime.Uint32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_ENUM:     "runtime.EnumSlice",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "runtime.Int32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "runtime.Int64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "runtime.Int32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "runtime.Int64Slice",
}

// proto2ConvertFuncs proto2 标量字段到 runtime 指针转换函数的映射
var proto2ConvertFuncs = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   "runtime.Float64P",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    "runtime.Float32P",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    "runtime.Int64P",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   "runtime.Uint64P",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    "runtime.Int32P",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "runtime.Uint64P",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "runtime.Uint32P",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     "runtime.BoolP",
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   "runtime.StringP",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   "runtime.Uint32P",
	descriptorpb.FieldDescriptorProto_TYPE_ENUM:     "runtime.EnumP",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "runtime.Int32P",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "runtime.Int64P",
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "runtime.Int32P",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "runtime.Int64P",
}

// proto2RepeatedConvertFuncs proto2 repeated 字段到 runtime 切片转换函数的映射
var proto2RepeatedConvertFuncs = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   "runtime.Float64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    "runtime.Float32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    "runtime.Int64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   "runtime.Uint64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    "runtime.Int32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "runtime.Uint64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "runtime.Uint32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     "runtime.BoolSlice",
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   "runtime.StringSlice",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   "runtime.Uint32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_ENUM:     "runtime.EnumSlice",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "runtime.Int32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "runtime.Int64Slice",
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "runtime.Int32Slice",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "runtime.Int64Slice",
}

// wellKnownTypeConv Well-Known Type 到 runtime 转换函数的映射
var wellKnownTypeConv = map[string]string{
	".google.protobuf.Timestamp":   "runtime.Timestamp",
	".google.protobuf.Duration":    "runtime.Duration",
	".google.protobuf.StringValue": "runtime.StringValue",
	".google.protobuf.FloatValue":  "runtime.FloatValue",
	".google.protobuf.DoubleValue": "runtime.DoubleValue",
	".google.protobuf.BoolValue":   "runtime.BoolValue",
	".google.protobuf.BytesValue":  "runtime.BytesValue",
	".google.protobuf.Int32Value":  "runtime.Int32Value",
	".google.protobuf.UInt32Value": "runtime.UInt32Value",
	".google.protobuf.Int64Value":  "runtime.Int64Value",
	".google.protobuf.UInt64Value": "runtime.UInt64Value",
}

// IsWellKnownType 判断给定全限定类型名是否为 Well-Known Type
func IsWellKnownType(typeName string) bool {
	_, ok := wellKnownTypeConv[typeName]
	return ok
}
