// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpod

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeletePodPreparesCommands(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace": {"shop"},
				"k8s.pod.name":  {"checkout-xyz1234"},
			},
		}),
	}

	action := NewDeletePodAction()
	state := action.NewEmptyState()

	// When
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, []string{"kubectl", "delete", "pod", "--namespace", "shop", "checkout-xyz1234"}, state.Opts.Command)
	require.Nil(t, state.Opts.RollbackCommand)
}
