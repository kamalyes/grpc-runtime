/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-05-28 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-05-28 13:26:07
 * @FilePath: \grpc-runtime\protocgen\descriptor\openapi_configuration.go
 * @Description: OpenAPI 配置加载，从 YAML 文件解析 OpenAPI 选项并注册到 Registry
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */

package descriptor

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kamalyes/grpc-runtime/protocgen/descriptor/openapiconfig"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

// loadOpenAPIConfigFromYAML 从 YAML 内容解析 OpenAPI 配置
func loadOpenAPIConfigFromYAML(yamlFileContents []byte, yamlSourceLogName string) (*openapiconfig.OpenAPIConfig, error) {
	var yamlContents interface{}
	if err := yaml.Unmarshal(yamlFileContents, &yamlContents); err != nil {
		return nil, fmt.Errorf("failed to parse gRPC API Configuration from YAML in %q: %w", yamlSourceLogName, err)
	}

	jsonContents, err := json.Marshal(yamlContents)
	if err != nil {
		return nil, err
	}

	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: false,
	}

	openapiConfiguration := openapiconfig.OpenAPIConfig{}
	if err := unmarshaler.Unmarshal(jsonContents, &openapiConfiguration); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI Configuration from YAML in %q: %w", yamlSourceLogName, err)
	}

	return &openapiConfiguration, nil
}

// registerOpenAPIOptions 将 OpenAPI 配置中的选项注册到 Registry
func registerOpenAPIOptions(registry *Registry, openAPIConfig *openapiconfig.OpenAPIConfig, yamlSourceLogName string) error {
	if openAPIConfig.OpenapiOptions == nil {
		return nil
	}

	if err := registry.RegisterOpenAPIOptions(openAPIConfig.OpenapiOptions); err != nil {
		return fmt.Errorf("failed to register option in %s: %w", yamlSourceLogName, err)
	}
	return nil
}

// LoadOpenAPIConfigFromYAML 从 YAML 文件加载 OpenAPI 配置
// 并将选项注册到 Registry 中，必须在加载 proto 文件之后调用
func (r *Registry) LoadOpenAPIConfigFromYAML(yamlFile string) error {
	yamlFileContents, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI Configuration description from %q: %w", yamlFile, err)
	}

	config, err := loadOpenAPIConfigFromYAML(yamlFileContents, yamlFile)
	if err != nil {
		return err
	}

	return registerOpenAPIOptions(r, config, yamlFile)
}
