// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extnode

import (
	"context"
	"testing"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeCountCheckDescribe(t *testing.T) {
	action := NewNodeCountCheckAction()

	description := action.Describe()

	assert.Equal(t, NodeCountCheckActionId, description.Id)
	assert.Equal(t, action_kit_api.Check, description.Kind)
	assert.Equal(t, action_kit_api.TimeControlInternal, description.TimeControl)
	require.NotNil(t, description.Status)
	assert.Equal(t, NodeCountCheckState{}, action.NewEmptyState())

	names := make([]string, 0, len(description.Parameters))
	for _, p := range description.Parameters {
		names = append(names, p.Name)
	}
	assert.ElementsMatch(t, []string{"duration", "nodeCount", "nodeCountCheckMode"}, names)
}

func TestNodeCountCheckStartIsANoop(t *testing.T) {
	result, err := NodeCountCheckAction{}.Start(context.Background(), &NodeCountCheckState{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestNodeCountCheckPrepareFailsOnInvalidConfig(t *testing.T) {
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]any{"duration": "not-a-number"},
		Target: &action_kit_api.Target{Attributes: map[string][]string{"k8s.cluster-name": {"test"}}},
	}

	_, err := prepareNodeCountCheckInternal(nil, &NodeCountCheckState{}, request)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to unmarshal the config.")
}
