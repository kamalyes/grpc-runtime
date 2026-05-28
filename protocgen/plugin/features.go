/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 00:00:00
 * @FilePath: \grpc-runtime\protocgen\plugin\features.go
 * @Description: protoc 插件特性声明，支持 proto3 optional 字段
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package plugin

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

// supportedCodeGeneratorFeatures 返回插件支持的代码生成特性
func supportedCodeGeneratorFeatures() uint64 {
	return uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
}

// SetSupportedFeaturesOnPlugin 将支持的 proto3 特性写入 protogen.Plugin
func SetSupportedFeaturesOnPlugin(gen *protogen.Plugin) {
	gen.SupportedFeatures = supportedCodeGeneratorFeatures()
}

// SetSupportedFeaturesOnResponse 将支持的 proto3 特性写入 CodeGeneratorResponse
func SetSupportedFeaturesOnResponse(resp *pluginpb.CodeGeneratorResponse) {
	sf := supportedCodeGeneratorFeatures()
	resp.SupportedFeatures = &sf
}
