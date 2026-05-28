package descriptor

import (
	"testing"

	"github.com/kamalyes/grpc-runtime/protocgen/openapiv2/options"
	"github.com/stretchr/testify/assert"
)

func TestLoadOpenAPIConfigFromYAMLRejectInvalidYAML(t *testing.T) {
	config, err := loadOpenAPIConfigFromYAML([]byte(`
openapiOptions:
file:
- file: test.proto
  - option:
      schemes:
        - HTTP
        - HTTPS
        - WSS
      securityDefinitions:
        security:
          ApiKeyAuth:
            type: TYPE_API_KEY
            in: IN_HEADER
            name: "X-API-Key"
`), "invalidyaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line 3")
	assert.Nil(t, config)
}

func TestLoadOpenAPIConfigFromYAML(t *testing.T) {
	config, err := loadOpenAPIConfigFromYAML([]byte(`
openapiOptions:
  file:
  - file: test.proto
    option:
      schemes:
      - HTTP
      - HTTPS
      - WSS
      securityDefinitions:
        security:
          ApiKeyAuth:
            type: TYPE_API_KEY
            in: IN_HEADER
            name: "X-API-Key"
`), "openapi_options")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, config.OpenapiOptions) {
		return
	}

	opts := config.OpenapiOptions
	if !assert.Len(t, opts.File, 1) {
		return
	}

	fileOpt := opts.File[0]
	assert.Equal(t, "test.proto", fileOpt.File)

	swaggerOpt := fileOpt.Option

	if !assert.NotNil(t, swaggerOpt) {
		return
	}

	assert.Equal(t, []options.Scheme{options.Scheme_HTTP, options.Scheme_HTTPS, options.Scheme_WSS}, swaggerOpt.Schemes)

	if !assert.NotNil(t, swaggerOpt.SecurityDefinitions) {
		return
	}
	assert.Len(t, swaggerOpt.SecurityDefinitions.Security, 1)
	secOpt, ok := swaggerOpt.SecurityDefinitions.Security["ApiKeyAuth"]
	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, options.SecurityScheme_TYPE_API_KEY, secOpt.Type)
	assert.Equal(t, options.SecurityScheme_IN_HEADER, secOpt.In)
	assert.Equal(t, "X-API-Key", secOpt.Name)
}

func TestLoadOpenAPIConfigFromYAMLUnknownKeys(t *testing.T) {
	_, err := loadOpenAPIConfigFromYAML([]byte(`
closedapiOptions:
  get: it?
openapiOptions:
  file:
  - file: test.proto
    option:
      schemes:
      - HTTP
`), "openapi_options")
	assert.Error(t, err)
}
