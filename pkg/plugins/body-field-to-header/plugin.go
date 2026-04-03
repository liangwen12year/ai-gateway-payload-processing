/*
Copyright 2026 The opendatahub.io Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package bodyfieldtoheader overrides the upstream body-field-to-header plugin
// to gracefully skip requests that don't contain the configured body field,
// instead of returning a 400 error. This is needed because the ext_proc filter
// processes ALL gateway traffic, including non-inference requests that may have
// a body but no "model" field.
package bodyfieldtoheader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/bbr/framework"
	logutil "sigs.k8s.io/gateway-api-inference-extension/pkg/common/observability/logging"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/plugin"
)

const (
	BodyFieldToHeaderPluginType = "body-field-to-header"
)

// compile-time type validation
var _ framework.RequestProcessor = &BodyFieldToHeaderPlugin{}

// BodyFieldToHeaderConfig defines the JSON configuration structure for the plugin.
type BodyFieldToHeaderConfig struct {
	FieldName  string `json:"field_name"`
	HeaderName string `json:"header_name"`
}

// BodyFieldToHeaderPluginFactory defines the factory function for BodyFieldToHeaderPlugin.
func BodyFieldToHeaderPluginFactory(name string, rawParameters json.RawMessage, _ framework.Handle) (framework.BBRPlugin, error) {
	var config BodyFieldToHeaderConfig

	if len(rawParameters) > 0 {
		if err := json.Unmarshal(rawParameters, &config); err != nil {
			return nil, fmt.Errorf("failed to parse the parameters of the '%s' plugin - %w", BodyFieldToHeaderPluginType, err)
		}
	}

	p, err := NewBodyFieldToHeaderPlugin(config.FieldName, config.HeaderName)
	if err != nil {
		return nil, fmt.Errorf("failed to create '%s' plugin - %w", BodyFieldToHeaderPluginType, err)
	}

	return p.WithName(name), nil
}

// NewBodyFieldToHeaderPlugin initializes a new BodyFieldToHeaderPlugin.
func NewBodyFieldToHeaderPlugin(fieldName, headerName string) (*BodyFieldToHeaderPlugin, error) {
	if fieldName == "" {
		return nil, errors.New("body fieldName is required in BodyFieldToHeader plugin")
	}
	if headerName == "" {
		return nil, errors.New("headerName is required in BodyFieldToHeader plugin")
	}

	return &BodyFieldToHeaderPlugin{
		typedName: plugin.TypedName{
			Type: BodyFieldToHeaderPluginType,
			Name: BodyFieldToHeaderPluginType,
		},
		fieldName:  fieldName,
		headerName: headerName,
	}, nil
}

// BodyFieldToHeaderPlugin extracts a value from a request body field and sets it as an HTTP header.
// Unlike the upstream version, this plugin skips gracefully when the body is nil or the
// configured field is missing, allowing non-inference requests to pass through.
type BodyFieldToHeaderPlugin struct {
	typedName  plugin.TypedName
	fieldName  string
	headerName string
}

// TypedName returns the type and name tuple of this plugin instance.
func (p *BodyFieldToHeaderPlugin) TypedName() plugin.TypedName {
	return p.typedName
}

// WithName sets the name of the plugin instance.
func (p *BodyFieldToHeaderPlugin) WithName(name string) *BodyFieldToHeaderPlugin {
	p.typedName.Name = name
	return p
}

// ProcessRequest extracts the configured field from the request body and sets it as a header.
// Returns nil (skip) when the body is nil or the field is missing, so non-inference
// requests pass through without error.
func (p *BodyFieldToHeaderPlugin) ProcessRequest(ctx context.Context, _ *framework.CycleState, request *framework.InferenceRequest) error {
	if request == nil || request.Headers == nil || request.Body == nil {
		return nil
	}

	rawFieldValue, exists := request.Body[p.fieldName]
	if !exists {
		log.FromContext(ctx).V(logutil.VERBOSE).Info("field not found in request body, skipping", "field", p.fieldName)
		return nil
	}

	fieldStr := fmt.Sprintf("%v", rawFieldValue)
	if fieldStr == "" {
		log.FromContext(ctx).V(logutil.VERBOSE).Info("field is empty in request body, skipping", "field", p.fieldName)
		return nil
	}

	log.FromContext(ctx).V(logutil.VERBOSE).Info("parsed field from body", "field", p.fieldName, "value", fieldStr)
	request.SetHeader(p.headerName, fieldStr)

	return nil
}
