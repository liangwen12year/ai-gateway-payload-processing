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

package bodyfieldtoheader

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/bbr/framework"
)

func TestNewBodyFieldToHeaderPlugin(t *testing.T) {
	tests := []struct {
		name       string
		fieldName  string
		headerName string
		wantErr    bool
	}{
		{name: "valid config", fieldName: "model", headerName: "X-Gateway-Model-Name"},
		{name: "missing field name", fieldName: "", headerName: "X-Header", wantErr: true},
		{name: "missing header name", fieldName: "model", headerName: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewBodyFieldToHeaderPlugin(tt.fieldName, tt.headerName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.fieldName, p.fieldName)
			assert.Equal(t, tt.headerName, p.headerName)
		})
	}
}

func TestProcessRequest_FieldFound(t *testing.T) {
	p, err := NewBodyFieldToHeaderPlugin("model", "X-Gateway-Model-Name")
	require.NoError(t, err)

	req := framework.NewInferenceRequest()
	req.Body["model"] = "gpt-4o"

	err = p.ProcessRequest(context.Background(), framework.NewCycleState(), req)
	require.NoError(t, err)

	assert.Equal(t, "gpt-4o", req.MutatedHeaders()["X-Gateway-Model-Name"])
}

func TestProcessRequest_FieldMissing(t *testing.T) {
	p, err := NewBodyFieldToHeaderPlugin("model", "X-Gateway-Model-Name")
	require.NoError(t, err)

	req := framework.NewInferenceRequest()
	req.Body["messages"] = []any{map[string]any{"role": "user", "content": "hello"}}

	err = p.ProcessRequest(context.Background(), framework.NewCycleState(), req)
	assert.NoError(t, err, "missing field should skip gracefully")
	assert.Empty(t, req.MutatedHeaders(), "no headers should be set when field is missing")
}

func TestProcessRequest_NilBody(t *testing.T) {
	p, err := NewBodyFieldToHeaderPlugin("model", "X-Gateway-Model-Name")
	require.NoError(t, err)

	req := framework.NewInferenceRequest()
	req.Body = nil

	err = p.ProcessRequest(context.Background(), framework.NewCycleState(), req)
	assert.NoError(t, err, "nil body should skip gracefully")
}

func TestProcessRequest_NilRequest(t *testing.T) {
	p, err := NewBodyFieldToHeaderPlugin("model", "X-Gateway-Model-Name")
	require.NoError(t, err)

	err = p.ProcessRequest(context.Background(), framework.NewCycleState(), nil)
	assert.NoError(t, err, "nil request should skip gracefully")
}

func TestProcessRequest_EmptyFieldValue(t *testing.T) {
	p, err := NewBodyFieldToHeaderPlugin("model", "X-Gateway-Model-Name")
	require.NoError(t, err)

	req := framework.NewInferenceRequest()
	req.Body["model"] = ""

	err = p.ProcessRequest(context.Background(), framework.NewCycleState(), req)
	assert.NoError(t, err, "empty field value should skip gracefully")
	assert.Empty(t, req.MutatedHeaders(), "no headers should be set for empty field")
}

func TestFactory_Success(t *testing.T) {
	params := json.RawMessage(`{"field_name":"model","header_name":"X-Gateway-Model-Name"}`)
	p, err := BodyFieldToHeaderPluginFactory("model-extractor", params, nil)
	require.NoError(t, err)
	assert.Equal(t, "model-extractor", p.TypedName().Name)
	assert.Equal(t, BodyFieldToHeaderPluginType, p.TypedName().Type)
}

func TestFactory_MissingFieldName(t *testing.T) {
	params := json.RawMessage(`{"header_name":"X-Header"}`)
	_, err := BodyFieldToHeaderPluginFactory("test", params, nil)
	require.Error(t, err)
}

func TestFactory_InvalidJSON(t *testing.T) {
	params := json.RawMessage(`{invalid`)
	_, err := BodyFieldToHeaderPluginFactory("test", params, nil)
	require.Error(t, err)
}
