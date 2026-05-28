/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:26:36
 * @FilePath: \grpc-runtime\protocgen\descriptor\registry.go
 * @Description: proto 描述符注册表，管理消息/枚举/服务/方法的注册与查找
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package descriptor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kamalyes/grpc-runtime/protocgen/descriptor/openapiconfig"
	"github.com/kamalyes/grpc-runtime/protocgen/openapiv2/options"
	"github.com/kamalyes/grpc-runtime/protocgen/plugin"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// repeatedFieldSeparator repeated 路径参数的分隔符配置
type repeatedFieldSeparator struct {
	name string
	sep  rune
}

// annotationIdentifier HTTP 注解标识，用于检测重复注解
type annotationIdentifier struct {
	method       string
	pathTemplate string
	service      *Service
}

// Registry proto 描述符注册表
// 从 pluginpb.CodeGeneratorRequest 中提取并管理消息、枚举、服务、方法等描述符信息
type Registry struct {
	// msgs 全限定消息名到描述符的映射
	msgs map[string]*Message
	// enums 全限定枚举名到描述符的映射
	enums map[string]*Enum
	// files 文件路径到描述符的映射
	files map[string]*File
	// meths 全限定方法名到描述符的映射
	meths map[string]*Method

	// prefix Go 包路径前缀
	prefix string
	// pkgMap 用户指定的文件路径到 proto 包名的映射
	pkgMap map[string]string
	// pkgAliases 已占用的包别名到包路径的映射
	pkgAliases map[string]string

	// allowDeleteBody 是否允许 HTTP DELETE 方法携带请求体
	allowDeleteBody bool
	// externalHTTPRules 外部 HTTP 规则映射（全限定方法名 → HttpRule 列表）
	externalHTTPRules map[string][]*annotations.HttpRule

	// allowMerge 是否合并多个 proto 文件为一个 OpenAPI 文件
	allowMerge bool
	// mergeFileName 合并后的 OpenAPI 文件名
	mergeFileName string
	// includePackageInTags 是否在 Tags 中包含 proto 包名
	includePackageInTags bool

	// repeatedPathParamSeparator repeated 路径参数的分隔符
	repeatedPathParamSeparator repeatedFieldSeparator
	// useJSONNamesForFields 是否使用 JSON 标签名作为 OpenAPI 字段名
	useJSONNamesForFields bool
	// useProto3FieldSemantics 是否使用 proto3 字段语义生成 OpenAPI 定义
	useProto3FieldSemantics bool

	// openAPINamingStrategy OpenAPI 命名策略（legacy/simple/fqn）
	openAPINamingStrategy string
	// visibilityRestrictionSelectors 可见性限制选择器
	visibilityRestrictionSelectors map[string]bool

	// useGoTemplate 是否在 proto 注释中使用 Go 模板
	useGoTemplate bool
	// goTemplateArgs Go 模板参数
	goTemplateArgs map[string]string
	// ignoreComments 是否忽略所有 proto 注释
	ignoreComments bool
	// removeInternalComments 是否移除内部注释（-- ... --）
	removeInternalComments bool

	// enumsAsInts 是否将枚举渲染为整数
	enumsAsInts bool
	// omitEnumDefaultValue 是否省略枚举默认值
	omitEnumDefaultValue bool
	// disableDefaultErrors 是否禁用默认错误类型生成
	disableDefaultErrors bool
	// simpleOperationIDs 是否使用简化的操作 ID（去除服务前缀）
	simpleOperationIDs bool

	// standalone 是否独立模式（强制包前缀）
	standalone bool
	// warnOnUnboundMethods 是否对未绑定 HTTP 规则的方法发出警告
	warnOnUnboundMethods bool
	// generateUnboundMethods 是否为未绑定 HTTP 规则的方法生成代理
	generateUnboundMethods bool
	// proto3OptionalNullable 是否将 proto3 optional 字段标记为 x-nullable
	proto3OptionalNullable bool

	// fileOptions 文件级 OpenAPI 选项
	fileOptions map[string]*options.Swagger
	// methodOptions 方法级 OpenAPI 选项
	methodOptions map[string]*options.Operation
	// messageOptions 消息级 OpenAPI 选项
	messageOptions map[string]*options.Schema
	// serviceOptions 服务级 OpenAPI 选项
	serviceOptions map[string]*options.Tag
	// fieldOptions 字段级 OpenAPI 选项
	fieldOptions map[string]*options.JSONSchema

	// omitPackageDoc 是否省略生成代码的包注释
	omitPackageDoc bool
	// recursiveDepth 字段参数的最大递归深度
	recursiveDepth int
	// annotationMap 已注册的 HTTP 注解（用于检测重复）
	annotationMap map[annotationIdentifier]struct{}

	// disableServiceTags 是否禁用服务标签生成
	disableServiceTags bool
	// disableDefaultResponses 是否禁用默认响应生成
	disableDefaultResponses bool
	// useAllOfForRefs 是否使用 allOf 作为 $ref 的容器
	useAllOfForRefs bool
	// allowPatchFeature 是否允许 PATCH 特性（使用 FieldMask）
	allowPatchFeature bool
	// preserveRPCOrder 是否保持 proto 文件中的 RPC 方法顺序
	preserveRPCOrder bool
	// enableRpcDeprecation 是否处理 gRPC 方法的 deprecated 选项
	enableRpcDeprecation bool
	// expandSlashedPathPatterns 是否展开包含子路径的路径参数模式
	expandSlashedPathPatterns bool
	// generateXGoType 是否生成 x-go-type 注解
	generateXGoType bool
}

// NewRegistry 创建新的注册表实例
func NewRegistry() *Registry {
	return &Registry{
		msgs:                           make(map[string]*Message),
		enums:                          make(map[string]*Enum),
		meths:                          make(map[string]*Method),
		files:                          make(map[string]*File),
		pkgMap:                         make(map[string]string),
		pkgAliases:                     make(map[string]string),
		externalHTTPRules:              make(map[string][]*annotations.HttpRule),
		openAPINamingStrategy:          "legacy",
		visibilityRestrictionSelectors: make(map[string]bool),
		repeatedPathParamSeparator: repeatedFieldSeparator{
			name: "csv",
			sep:  ',',
		},
		fileOptions:    make(map[string]*options.Swagger),
		methodOptions:  make(map[string]*options.Operation),
		messageOptions: make(map[string]*options.Schema),
		serviceOptions: make(map[string]*options.Tag),
		fieldOptions:   make(map[string]*options.JSONSchema),
		annotationMap:  make(map[annotationIdentifier]struct{}),
		recursiveDepth: 1000,
	}
}

// Load 从 CodeGeneratorRequest 加载所有描述符定义
func (r *Registry) Load(req *pluginpb.CodeGeneratorRequest) error {
	gen, err := protogen.Options{}.New(req)
	if err != nil {
		return err
	}
	plugin.SetSupportedFeaturesOnPlugin(gen)
	return r.load(gen)
}

// LoadFromPlugin 从已初始化的 protogen.Plugin 加载描述符
func (r *Registry) LoadFromPlugin(gen *protogen.Plugin) error {
	return r.load(gen)
}

// load 内部加载逻辑：先加载所有文件的消息和枚举，再加载服务
func (r *Registry) load(gen *protogen.Plugin) error {
	filePaths := make([]string, 0, len(gen.FilesByPath))
	for filePath := range gen.FilesByPath {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	for _, filePath := range filePaths {
		r.loadFile(filePath, gen.FilesByPath[filePath])
	}

	for _, filePath := range filePaths {
		if !gen.FilesByPath[filePath].Generate {
			continue
		}
		file := r.files[filePath]
		if err := r.loadServices(file); err != nil {
			return err
		}
	}

	return nil
}

// loadFile 加载文件中的消息和枚举定义
// 注意：不加载服务和方法，需在所有文件加载完成后调用 loadServices
func (r *Registry) loadFile(filePath string, file *protogen.File) {
	pkg := GoPackage{
		Path: string(file.GoImportPath),
		Name: string(file.GoPackageName),
	}
	if r.standalone {
		pkg.Alias = "ext" + cases.Title(language.AmericanEnglish).String(pkg.Name)
	}

	if err := r.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
		for i := 0; ; i++ {
			alias := fmt.Sprintf("%s_%d", pkg.Name, i)
			if err := r.ReserveGoPackageAlias(alias, pkg.Path); err == nil {
				pkg.Alias = alias
				break
			}
		}
	}
	f := &File{
		FileDescriptorProto:     file.Proto,
		GoPkg:                   pkg,
		GeneratedFilenamePrefix: file.GeneratedFilenamePrefix,
	}

	r.files[filePath] = f
	r.registerMsg(f, nil, file.Proto.MessageType)
	r.registerEnum(f, nil, file.Proto.EnumType)
}

// registerMsg 递归注册消息及其嵌套类型
func (r *Registry) registerMsg(file *File, outerPath []string, msgs []*descriptorpb.DescriptorProto) {
	for i, md := range msgs {
		m := &Message{
			File:              file,
			Outers:            outerPath,
			DescriptorProto:   md,
			Index:             i,
			ForcePrefixedName: r.standalone,
		}
		for _, fd := range md.GetField() {
			m.Fields = append(m.Fields, &Field{
				Message:              m,
				FieldDescriptorProto: fd,
				ForcePrefixedName:    r.standalone,
			})
		}
		file.Messages = append(file.Messages, m)
		r.msgs[m.FQMN()] = m
		if grpclog.V(1) {
			grpclog.Infof("Register name: %s", m.FQMN())
		}

		var outers []string
		outers = append(outers, outerPath...)
		outers = append(outers, m.GetName())
		r.registerMsg(file, outers, m.GetNestedType())
		r.registerEnum(file, outers, m.GetEnumType())
	}
}

// registerEnum 注册枚举类型
func (r *Registry) registerEnum(file *File, outerPath []string, enums []*descriptorpb.EnumDescriptorProto) {
	for i, ed := range enums {
		e := &Enum{
			File:                file,
			Outers:              outerPath,
			EnumDescriptorProto: ed,
			Index:               i,
			ForcePrefixedName:   r.standalone,
		}
		file.Enums = append(file.Enums, e)
		r.enums[e.FQEN()] = e
		if grpclog.V(1) {
			grpclog.Infof("Register enum name: %s", e.FQEN())
		}
	}
}

// ============================================================================ //
// 查找方法
// ============================================================================ //

// LookupMsg 根据名称查找消息类型
// 如果 name 是相对名称，则从 location 上下文开始逐级向上解析
func (r *Registry) LookupMsg(location, name string) (*Message, error) {
	if grpclog.V(1) {
		grpclog.Infof("Lookup %s from %s", name, location)
	}
	if strings.HasPrefix(name, ".") {
		m, ok := r.msgs[name]
		if !ok {
			return nil, fmt.Errorf("no message found: %s", name)
		}
		return m, nil
	}

	if !strings.HasPrefix(location, ".") {
		location = fmt.Sprintf(".%s", location)
	}
	components := strings.Split(location, ".")
	for len(components) > 0 {
		fqmn := strings.Join(append(components, name), ".")
		if m, ok := r.msgs[fqmn]; ok {
			return m, nil
		}
		components = components[:len(components)-1]
	}
	return nil, fmt.Errorf("no message found: %s", name)
}

// LookupEnum 根据名称查找枚举类型
// 如果 name 是相对名称，则从 location 上下文开始逐级向上解析
func (r *Registry) LookupEnum(location, name string) (*Enum, error) {
	if grpclog.V(1) {
		grpclog.Infof("Lookup enum %s from %s", name, location)
	}
	if strings.HasPrefix(name, ".") {
		e, ok := r.enums[name]
		if !ok {
			return nil, fmt.Errorf("no enum found: %s", name)
		}
		return e, nil
	}

	if !strings.HasPrefix(location, ".") {
		location = fmt.Sprintf(".%s", location)
	}
	components := strings.Split(location, ".")
	for len(components) > 0 {
		fqen := strings.Join(append(components, name), ".")
		if e, ok := r.enums[fqen]; ok {
			return e, nil
		}
		components = components[:len(components)-1]
	}
	return nil, fmt.Errorf("no enum found: %s", name)
}

// LookupFile 根据文件名查找文件描述符
func (r *Registry) LookupFile(name string) (*File, error) {
	f, ok := r.files[name]
	if !ok {
		return nil, fmt.Errorf("no such file given: %s", name)
	}
	return f, nil
}

// LookupExternalHTTPRules 根据全限定方法名查找外部 HTTP 规则
func (r *Registry) LookupExternalHTTPRules(qualifiedMethodName string) []*annotations.HttpRule {
	return r.externalHTTPRules[qualifiedMethodName]
}

// AddExternalHTTPRule 为指定方法添加外部 HTTP 规则
func (r *Registry) AddExternalHTTPRule(qualifiedMethodName string, rule *annotations.HttpRule) {
	r.externalHTTPRules[qualifiedMethodName] = append(r.externalHTTPRules[qualifiedMethodName], rule)
}

// UnboundExternalHTTPRules 返回注册表中没有匹配方法的外部 HTTP 规则列表
func (r *Registry) UnboundExternalHTTPRules() []string {
	allServiceMethods := make(map[string]struct{})
	for _, f := range r.files {
		for _, s := range f.GetService() {
			svc := &Service{File: f, ServiceDescriptorProto: s}
			for _, m := range s.GetMethod() {
				method := &Method{Service: svc, MethodDescriptorProto: m}
				allServiceMethods[method.FQMN()] = struct{}{}
			}
		}
	}

	var missingMethods []string
	for httpRuleMethod := range r.externalHTTPRules {
		if _, ok := allServiceMethods[httpRuleMethod]; !ok {
			missingMethods = append(missingMethods, httpRuleMethod)
		}
	}
	return missingMethods
}

// GetAllFQMNs 返回所有已注册消息的全限定名列表
func (r *Registry) GetAllFQMNs() []string {
	keys := make([]string, 0, len(r.msgs))
	for k := range r.msgs {
		keys = append(keys, k)
	}
	return keys
}

// GetAllFQENs 返回所有已注册枚举的全限定名列表
func (r *Registry) GetAllFQENs() []string {
	keys := make([]string, 0, len(r.enums))
	for k := range r.enums {
		keys = append(keys, k)
	}
	return keys
}

// GetAllFQMethNs 返回所有已注册方法的全限定名列表
func (r *Registry) GetAllFQMethNs() []string {
	keys := make([]string, 0, len(r.meths))
	for k := range r.meths {
		keys = append(keys, k)
	}
	return keys
}

// ============================================================================ //
// 配置方法
// ============================================================================ //

// AddPkgMap 添加 .proto 文件到 proto 包名的映射
func (r *Registry) AddPkgMap(file, protoPkg string) {
	r.pkgMap[file] = protoPkg
}

// SetPrefix 设置 Go 包路径前缀
func (r *Registry) SetPrefix(prefix string) {
	r.prefix = prefix
}

// SetStandalone 设置独立模式（控制包前缀）
func (r *Registry) SetStandalone(standalone bool) {
	r.standalone = standalone
}

// SetRecursiveDepth 设置字段参数的最大递归深度
func (r *Registry) SetRecursiveDepth(count int) {
	r.recursiveDepth = count
}

// GetRecursiveDepth 返回字段参数的最大递归深度
func (r *Registry) GetRecursiveDepth() int {
	return r.recursiveDepth
}

// ReserveGoPackageAlias 预留 Go 包别名
// 如果别名已被其他包占用则返回错误
func (r *Registry) ReserveGoPackageAlias(alias, pkgpath string) error {
	if taken, ok := r.pkgAliases[alias]; ok {
		if taken == pkgpath {
			return nil
		}
		return fmt.Errorf("package name %s is already taken. Use another alias", alias)
	}
	r.pkgAliases[alias] = pkgpath
	return nil
}

// SetAllowDeleteBody 设置是否允许 HTTP DELETE 方法携带请求体
func (r *Registry) SetAllowDeleteBody(allow bool) {
	r.allowDeleteBody = allow
}

// SetAllowMerge 设置是否合并多个 proto 文件为一个 OpenAPI 文件
func (r *Registry) SetAllowMerge(allow bool) {
	r.allowMerge = allow
}

// IsAllowMerge 返回是否合并 OpenAPI 文件
func (r *Registry) IsAllowMerge() bool {
	return r.allowMerge
}

// SetMergeFileName 设置合并后的 OpenAPI 文件名
func (r *Registry) SetMergeFileName(mergeFileName string) {
	r.mergeFileName = mergeFileName
}

// GetMergeFileName 返回合并后的 OpenAPI 文件名
func (r *Registry) GetMergeFileName() string {
	return r.mergeFileName
}

// SetIncludePackageInTags 设置是否在 Tags 中包含 proto 包名
func (r *Registry) SetIncludePackageInTags(allow bool) {
	r.includePackageInTags = allow
}

// IsIncludePackageInTags 返回是否在 Tags 中包含 proto 包名
func (r *Registry) IsIncludePackageInTags() bool {
	return r.includePackageInTags
}

// GetRepeatedPathParamSeparator 返回 repeated 路径参数的分隔符
func (r *Registry) GetRepeatedPathParamSeparator() rune {
	return r.repeatedPathParamSeparator.sep
}

// GetRepeatedPathParamSeparatorName 返回 repeated 路径参数分隔符的名称
func (r *Registry) GetRepeatedPathParamSeparatorName() string {
	return r.repeatedPathParamSeparator.name
}

// SetRepeatedPathParamSeparator 设置 repeated 路径参数的分隔符
// 支持的名称：csv、pipes、ssv、tsv
func (r *Registry) SetRepeatedPathParamSeparator(name string) error {
	var sep rune
	switch name {
	case "csv":
		sep = ','
	case "pipes":
		sep = '|'
	case "ssv":
		sep = ' '
	case "tsv":
		sep = '\t'
	default:
		return fmt.Errorf("unknown repeated path parameter separator: %s", name)
	}
	r.repeatedPathParamSeparator = repeatedFieldSeparator{
		name: name,
		sep:  sep,
	}
	return nil
}

// SetUseJSONNamesForFields 设置是否使用 JSON 标签名
func (r *Registry) SetUseJSONNamesForFields(use bool) {
	r.useJSONNamesForFields = use
}

// GetUseJSONNamesForFields 返回是否使用 JSON 标签名
func (r *Registry) GetUseJSONNamesForFields() bool {
	return r.useJSONNamesForFields
}

// GetUseProto3FieldSemantics 返回是否使用 proto3 字段语义
func (r *Registry) GetUseProto3FieldSemantics() bool {
	return r.useProto3FieldSemantics
}

// SetUseProto3FieldSemantics 设置是否使用 proto3 字段语义
func (r *Registry) SetUseProto3FieldSemantics(useProto3FieldSemantics bool) {
	r.useProto3FieldSemantics = useProto3FieldSemantics
}

// SetUseFQNForOpenAPIName 设置是否使用全限定名作为 OpenAPI 名称
// Deprecated: 使用 SetOpenAPINamingStrategy 替代
func (r *Registry) SetUseFQNForOpenAPIName(use bool) {
	r.openAPINamingStrategy = "fqn"
}

// GetUseFQNForOpenAPIName 返回是否使用全限定名
// Deprecated: 使用 GetOpenAPINamingStrategy 替代
func (r *Registry) GetUseFQNForOpenAPIName() bool {
	return r.openAPINamingStrategy == "fqn"
}

// SetOpenAPINamingStrategy 设置 OpenAPI 命名策略
func (r *Registry) SetOpenAPINamingStrategy(strategy string) {
	r.openAPINamingStrategy = strategy
}

// GetOpenAPINamingStrategy 返回当前 OpenAPI 命名策略
func (r *Registry) GetOpenAPINamingStrategy() string {
	return r.openAPINamingStrategy
}

// SetUseGoTemplate 设置是否使用 Go 模板
func (r *Registry) SetUseGoTemplate(use bool) {
	r.useGoTemplate = use
}

// GetUseGoTemplate 返回是否使用 Go 模板
func (r *Registry) GetUseGoTemplate() bool {
	return r.useGoTemplate
}

// SetGoTemplateArgs 设置 Go 模板参数
func (r *Registry) SetGoTemplateArgs(kvs []string) {
	r.goTemplateArgs = make(map[string]string)
	for _, kv := range kvs {
		if key, value, found := strings.Cut(kv, "="); found {
			r.goTemplateArgs[key] = value
		}
	}
}

// GetGoTemplateArgs 返回 Go 模板参数
func (r *Registry) GetGoTemplateArgs() map[string]string {
	return r.goTemplateArgs
}

// SetIgnoreComments 设置是否忽略所有 proto 注释
func (r *Registry) SetIgnoreComments(ignore bool) {
	r.ignoreComments = ignore
}

// GetIgnoreComments 返回是否忽略所有 proto 注释
func (r *Registry) GetIgnoreComments() bool {
	return r.ignoreComments
}

// SetRemoveInternalComments 设置是否移除内部注释
func (r *Registry) SetRemoveInternalComments(remove bool) {
	r.removeInternalComments = remove
}

// GetRemoveInternalComments 返回是否移除内部注释
func (r *Registry) GetRemoveInternalComments() bool {
	return r.removeInternalComments
}

// SetEnumsAsInts 设置是否将枚举渲染为整数
func (r *Registry) SetEnumsAsInts(enumsAsInts bool) {
	r.enumsAsInts = enumsAsInts
}

// GetEnumsAsInts 返回是否将枚举渲染为整数
func (r *Registry) GetEnumsAsInts() bool {
	return r.enumsAsInts
}

// SetOmitEnumDefaultValue 设置是否省略枚举默认值
func (r *Registry) SetOmitEnumDefaultValue(omit bool) {
	r.omitEnumDefaultValue = omit
}

// GetOmitEnumDefaultValue 返回是否省略枚举默认值
func (r *Registry) GetOmitEnumDefaultValue() bool {
	return r.omitEnumDefaultValue
}

// SetVisibilityRestrictionSelectors 设置可见性限制选择器
func (r *Registry) SetVisibilityRestrictionSelectors(selectors []string) {
	r.visibilityRestrictionSelectors = make(map[string]bool)
	for _, selector := range selectors {
		r.visibilityRestrictionSelectors[strings.TrimSpace(selector)] = true
	}
}

// GetVisibilityRestrictionSelectors 返回可见性限制选择器
func (r *Registry) GetVisibilityRestrictionSelectors() map[string]bool {
	return r.visibilityRestrictionSelectors
}

// SetDisableDefaultErrors 设置是否禁用默认错误类型生成
func (r *Registry) SetDisableDefaultErrors(use bool) {
	r.disableDefaultErrors = use
}

// GetDisableDefaultErrors 返回是否禁用默认错误类型生成
func (r *Registry) GetDisableDefaultErrors() bool {
	return r.disableDefaultErrors
}

// SetSimpleOperationIDs 设置是否使用简化操作 ID
func (r *Registry) SetSimpleOperationIDs(use bool) {
	r.simpleOperationIDs = use
}

// GetSimpleOperationIDs 返回是否使用简化操作 ID
func (r *Registry) GetSimpleOperationIDs() bool {
	return r.simpleOperationIDs
}

// SetWarnOnUnboundMethods 设置是否对未绑定方法发出警告
func (r *Registry) SetWarnOnUnboundMethods(warn bool) {
	r.warnOnUnboundMethods = warn
}

// SetGenerateUnboundMethods 设置是否为未绑定方法生成代理
func (r *Registry) SetGenerateUnboundMethods(generate bool) {
	r.generateUnboundMethods = generate
}

// SetOmitPackageDoc 设置是否省略生成代码的包注释
func (r *Registry) SetOmitPackageDoc(omit bool) {
	r.omitPackageDoc = omit
}

// GetOmitPackageDoc 返回是否省略生成代码的包注释
func (r *Registry) GetOmitPackageDoc() bool {
	return r.omitPackageDoc
}

// SetProto3OptionalNullable 设置是否将 proto3 optional 字段标记为 x-nullable
func (r *Registry) SetProto3OptionalNullable(proto3OptionalNullable bool) {
	r.proto3OptionalNullable = proto3OptionalNullable
}

// GetProto3OptionalNullable 返回是否将 proto3 optional 字段标记为 x-nullable
func (r *Registry) GetProto3OptionalNullable() bool {
	return r.proto3OptionalNullable
}

// SetDisableServiceTags 设置是否禁用服务标签生成
func (r *Registry) SetDisableServiceTags(use bool) {
	r.disableServiceTags = use
}

// GetDisableServiceTags 返回是否禁用服务标签生成
func (r *Registry) GetDisableServiceTags() bool {
	return r.disableServiceTags
}

// SetDisableDefaultResponses 设置是否禁用默认响应生成
func (r *Registry) SetDisableDefaultResponses(use bool) {
	r.disableDefaultResponses = use
}

// GetDisableDefaultResponses 返回是否禁用默认响应生成
func (r *Registry) GetDisableDefaultResponses() bool {
	return r.disableDefaultResponses
}

// SetUseAllOfForRefs 设置是否使用 allOf 作为 $ref 的容器
func (r *Registry) SetUseAllOfForRefs(use bool) {
	r.useAllOfForRefs = use
}

// GetUseAllOfForRefs 返回是否使用 allOf 作为 $ref 的容器
func (r *Registry) GetUseAllOfForRefs() bool {
	return r.useAllOfForRefs
}

// SetAllowPatchFeature 设置是否允许 PATCH 特性
func (r *Registry) SetAllowPatchFeature(allow bool) {
	r.allowPatchFeature = allow
}

// GetAllowPatchFeature 返回是否允许 PATCH 特性
func (r *Registry) GetAllowPatchFeature() bool {
	return r.allowPatchFeature
}

// SetPreserveRPCOrder 设置是否保持 RPC 方法顺序
func (r *Registry) SetPreserveRPCOrder(preserve bool) {
	r.preserveRPCOrder = preserve
}

// IsPreserveRPCOrder 返回是否保持 RPC 方法顺序
func (r *Registry) IsPreserveRPCOrder() bool {
	return r.preserveRPCOrder
}

// SetEnableRpcDeprecation 设置是否处理 gRPC 方法的 deprecated 选项
func (r *Registry) SetEnableRpcDeprecation(enable bool) {
	r.enableRpcDeprecation = enable
}

// GetEnableRpcDeprecation 返回是否处理 gRPC 方法的 deprecated 选项
func (r *Registry) GetEnableRpcDeprecation() bool {
	return r.enableRpcDeprecation
}

// SetExpandSlashedPathPatterns 设置是否展开包含子路径的路径参数模式
func (r *Registry) SetExpandSlashedPathPatterns(expandSlashedPathPatterns bool) {
	r.expandSlashedPathPatterns = expandSlashedPathPatterns
}

// GetExpandSlashedPathPatterns 返回是否展开包含子路径的路径参数模式
func (r *Registry) GetExpandSlashedPathPatterns() bool {
	return r.expandSlashedPathPatterns
}

// SetGenerateXGoType 设置是否生成 x-go-type 注解
func (r *Registry) SetGenerateXGoType(generateXGoType bool) {
	r.generateXGoType = generateXGoType
}

// GetGenerateXGoType 返回是否生成 x-go-type 注解
func (r *Registry) GetGenerateXGoType() bool {
	return r.generateXGoType
}

// FieldName 根据配置返回字段名（JSON 名或 proto 名）
func (r *Registry) FieldName(f *Field) string {
	if r.useJSONNamesForFields {
		return f.GetJsonName()
	}
	return f.GetName()
}

// CheckDuplicateAnnotation 检查并注册 HTTP 注解，防止重复
func (r *Registry) CheckDuplicateAnnotation(httpMethod string, httpTemplate string, svc *Service) error {
	a := annotationIdentifier{method: httpMethod, pathTemplate: httpTemplate, service: svc}
	if _, ok := r.annotationMap[a]; ok {
		return fmt.Errorf("duplicate annotation: method=%s, template=%s", httpMethod, httpTemplate)
	}
	r.annotationMap[a] = struct{}{}
	return nil
}

// ============================================================================ //
// OpenAPI 选项注册
// ============================================================================ //

// RegisterOpenAPIOptions 注册 OpenAPI 选项
func (r *Registry) RegisterOpenAPIOptions(opts *openapiconfig.OpenAPIOptions) error {
	if opts == nil {
		return nil
	}

	for _, opt := range opts.File {
		if _, ok := r.files[opt.File]; !ok {
			return fmt.Errorf("no file %s found", opt.File)
		}
		r.fileOptions[opt.File] = opt.Option
	}

	methods := make(map[string]struct{})
	services := make(map[string]struct{})
	for _, f := range r.files {
		for _, s := range f.Services {
			services[s.FQSN()] = struct{}{}
			for _, m := range s.Methods {
				methods[m.FQMN()] = struct{}{}
			}
		}
	}

	for _, opt := range opts.Method {
		qualifiedMethod := "." + opt.Method
		if _, ok := methods[qualifiedMethod]; !ok {
			return fmt.Errorf("no method %s found", opt.Method)
		}
		r.methodOptions[qualifiedMethod] = opt.Option
	}

	for _, opt := range opts.Message {
		qualifiedMessage := "." + opt.Message
		if _, ok := r.msgs[qualifiedMessage]; !ok {
			return fmt.Errorf("no message %s found", opt.Message)
		}
		r.messageOptions[qualifiedMessage] = opt.Option
	}

	for _, opt := range opts.Service {
		qualifiedService := "." + opt.Service
		if _, ok := services[qualifiedService]; !ok {
			return fmt.Errorf("no service %s found", opt.Service)
		}
		r.serviceOptions[qualifiedService] = opt.Option
	}

	fields := make(map[string]struct{})
	for _, m := range r.msgs {
		for _, f := range m.Fields {
			fields[f.FQFN()] = struct{}{}
		}
	}
	for _, opt := range opts.Field {
		qualifiedField := "." + opt.Field
		if _, ok := fields[qualifiedField]; !ok {
			return fmt.Errorf("no field %s found", opt.Field)
		}
		r.fieldOptions[qualifiedField] = opt.Option
	}
	return nil
}

// GetOpenAPIFileOption 返回文件的 OpenAPI 选项
func (r *Registry) GetOpenAPIFileOption(file string) (*options.Swagger, bool) {
	opt, ok := r.fileOptions[file]
	return opt, ok
}

// GetOpenAPIMethodOption 返回方法的 OpenAPI 选项
func (r *Registry) GetOpenAPIMethodOption(qualifiedMethod string) (*options.Operation, bool) {
	opt, ok := r.methodOptions[qualifiedMethod]
	return opt, ok
}

// GetOpenAPIMessageOption 返回消息的 OpenAPI 选项
func (r *Registry) GetOpenAPIMessageOption(qualifiedMessage string) (*options.Schema, bool) {
	opt, ok := r.messageOptions[qualifiedMessage]
	return opt, ok
}

// GetOpenAPIServiceOption 返回服务的 OpenAPI 选项
func (r *Registry) GetOpenAPIServiceOption(qualifiedService string) (*options.Tag, bool) {
	opt, ok := r.serviceOptions[qualifiedService]
	return opt, ok
}

// GetOpenAPIFieldOption 返回字段的 OpenAPI 选项
func (r *Registry) GetOpenAPIFieldOption(qualifiedField string) (*options.JSONSchema, bool) {
	opt, ok := r.fieldOptions[qualifiedField]
	return opt, ok
}
