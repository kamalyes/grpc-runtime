/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:25:57
 * @FilePath: \grpc-runtime\protocgen\descriptor\grpc_api_configuration.go
 * @Description: gRPC API 服务配置加载，从 YAML 文件解析 HttpRule 并注册到 Registry
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package descriptor

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	serviceconfig "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

// loadGrpcAPIServiceFromYAML 从 YAML 内容解析 gRPC API 服务配置
func loadGrpcAPIServiceFromYAML(yamlFileContents []byte, yamlSourceLogName string) (*serviceconfig.Service, error) {
	var yamlContents interface{}
	if err := yaml.Unmarshal(yamlFileContents, &yamlContents); err != nil {
		return nil, fmt.Errorf("failed to parse gRPC API Configuration from YAML in %q: %w", yamlSourceLogName, err)
	}

	jsonContents, err := json.Marshal(yamlContents)
	if err != nil {
		return nil, err
	}

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	serviceConfiguration := serviceconfig.Service{}
	if err := unmarshaler.Unmarshal(jsonContents, &serviceConfiguration); err != nil {
		return nil, fmt.Errorf("failed to parse gRPC API Configuration from YAML in %q: %w", yamlSourceLogName, err)
	}

	return &serviceConfiguration, nil
}

// registerHTTPRulesFromGrpcAPIService 将 gRPC API 服务配置中的 HttpRule 注册到 Registry
func registerHTTPRulesFromGrpcAPIService(registry *Registry, service *serviceconfig.Service, sourceLogName string) error {
	if service.Http == nil {
		return nil
	}

	for _, rule := range service.Http.GetRules() {
		selector := "." + strings.Trim(rule.GetSelector(), " ")
		if strings.ContainsAny(selector, "*, ") {
			return fmt.Errorf("selector %q in %v must specify a single service method without wildcards", rule.GetSelector(), sourceLogName)
		}

		registry.AddExternalHTTPRule(selector, rule)
	}

	return nil
}

// LoadGrpcAPIServiceFromYAML 从 YAML 文件加载 gRPC API 服务配置
// 并将 HttpRule 注册到 Registry 中，必须在加载 proto 文件之前调用
func (r *Registry) LoadGrpcAPIServiceFromYAML(yamlFile string) error {
	yamlFileContents, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read gRPC API Configuration description from %q: %w", yamlFile, err)
	}

	service, err := loadGrpcAPIServiceFromYAML(yamlFileContents, yamlFile)
	if err != nil {
		return err
	}

	return registerHTTPRulesFromGrpcAPIService(r, service, yamlFile)
}
