// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTaintNodeExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 100000,
			"key":      "test",
			"value":    "abc",
			"effect":   "NoSchedule",
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"host.hostname": {"test"},
			},
		}),
	}

	action := NewTaintNodeAction()
	state := action.NewEmptyState()

	// When
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, "test", state.Node)
	require.Equal(t, "test=abc:NoSchedule", state.Taint)
}
