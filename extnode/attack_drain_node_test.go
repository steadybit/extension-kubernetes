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

func TestDrainNodePrepareCommands(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 100000,
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"host.hostname": {"test"},
			},
		}),
	}

	action := NewDrainNodeAction()
	state := action.NewEmptyState()

	// When
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, []string{"kubectl", "drain", "test", "--pod-selector=steadybit.com/extension!=true,steadybit.com/agent!=true", "--delete-emptydir-data", "--ignore-daemonsets", "--force"}, state.Opts.Command)
	require.Equal(t, []string{"kubectl", "get", "node", "test"}, state.Opts.RollbackPreconditionCommand)
	require.Equal(t, []string{"kubectl", "uncordon", "test"}, state.Opts.RollbackCommand)
}
