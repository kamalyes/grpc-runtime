package descriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadGrpcAPIServiceFromYAMLInvalidType(t *testing.T) {
	// Ideally this would fail but for now this test documents that it doesn't
	service, err := loadGrpcAPIServiceFromYAML([]byte(`type: not.the.right.type`), "invalidtype")
	assert.NoError(t, err)
	assert.NotNil(t, service)
}

func TestLoadGrpcAPIServiceFromYAMLSingleRule(t *testing.T) {
	service, err := loadGrpcAPIServiceFromYAML([]byte(`
type: google.api.Service
config_version: 3

http:
 rules:
 - selector: grpctest.YourService.Echo
   post: /v1/myecho
   body: "*"
`), "example")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, service.Http) {
		return
	}
	if !assert.Len(t, service.Http.GetRules(), 1) {
		return
	}

	rule := service.Http.GetRules()[0]
	assert.Equal(t, "grpctest.YourService.Echo", rule.GetSelector())
	assert.Equal(t, "/v1/myecho", rule.GetPost())
	assert.Equal(t, "*", rule.GetBody())
}

func TestLoadGrpcAPIServiceFromYAMLRejectInvalidYAML(t *testing.T) {
	service, err := loadGrpcAPIServiceFromYAML([]byte(`
type: google.api.Service
config_version: 3

http:
 rules:
 - selector: grpctest.YourService.Echo
   - post: thislinebreakstheselectorblockabovewiththeleadingdash
   body: "*"
`), "invalidyaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line 6")
	assert.Nil(t, service)
}

func TestLoadGrpcAPIServiceFromYAMLMultipleWithAdditionalBindings(t *testing.T) {
	service, err := loadGrpcAPIServiceFromYAML([]byte(`
type: google.api.Service
config_version: 3

http:
 rules:
 - selector: first.selector
   post: /my/post/path
   body: "*"
   additional_bindings:
   - post: /additional/post/path
   - put: /additional/put/{value}/path
   - delete: "{value}"
   - patch: "/additional/patch/{value}"
 - selector: some.other.service
   delete: foo
`), "example")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, service) || !assert.NotNil(t, service.Http) {
		return
	}
	if !assert.Len(t, service.Http.GetRules(), 2) {
		return
	}

	first := service.Http.GetRules()[0]
	assert.Equal(t, "first.selector", first.GetSelector())
	assert.Equal(t, "*", first.GetBody())
	assert.Equal(t, "/my/post/path", first.GetPost())
	if !assert.Len(t, first.GetAdditionalBindings(), 4) {
		return
	}
	assert.Equal(t, "/additional/post/path", first.GetAdditionalBindings()[0].GetPost())
	assert.Equal(t, "/additional/put/{value}/path", first.GetAdditionalBindings()[1].GetPut())
	assert.Equal(t, "{value}", first.GetAdditionalBindings()[2].GetDelete())
	assert.Equal(t, "/additional/patch/{value}", first.GetAdditionalBindings()[3].GetPatch())

	second := service.Http.GetRules()[1]
	assert.Equal(t, "some.other.service", second.GetSelector())
	assert.Equal(t, "foo", second.GetDelete())
	assert.Empty(t, second.GetAdditionalBindings())
}

func TestLoadGrpcAPIServiceFromYAMLUnknownKeys(t *testing.T) {
	service, err := loadGrpcAPIServiceFromYAML([]byte(`
type: google.api.Service
config_version: 3

very: key
much: 1

http:
 rules:
 - selector: some.other.service
   delete: foo
   invalidkey: yes
`), "example")
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, service) || !assert.NotNil(t, service.Http) {
		return
	}
	if !assert.Len(t, service.Http.GetRules(), 1) {
		return
	}

	first := service.Http.GetRules()[0]
	assert.Equal(t, "some.other.service", first.GetSelector())
	assert.Equal(t, "foo", first.GetDelete())
}
