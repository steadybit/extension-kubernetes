// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extargorollout

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestArgoRolloutRestartAction_Prepare(t *testing.T) {
	request := action_kit_api.PrepareActionRequestBody{
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.cluster-name": {"development"},
				"k8s.namespace":    {"default"},
				"k8s.argo-rollout": {"shop"},
			},
		}),
	}

	action := NewArgoRolloutRestartAction(nil)
	state := action.NewEmptyState()

	_, err := action.Prepare(context.Background(), &state, request)

	require.NoError(t, err)
	assert.Equal(t, "default", state.Namespace)
	assert.Equal(t, "shop", state.ArgoRollout)
}

func TestArgoRolloutRestartAction_Prepare_MissingAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string][]string
	}{
		{
			name: "missing namespace",
			attributes: map[string][]string{
				"k8s.argo-rollout": {"shop"},
			},
		},
		{
			name: "missing rollout",
			attributes: map[string][]string{
				"k8s.namespace": {"default"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := action_kit_api.PrepareActionRequestBody{
				Target: extutil.Ptr(action_kit_api.Target{
					Attributes: tt.attributes,
				}),
			}

			action := NewArgoRolloutRestartAction(nil)
			state := action.NewEmptyState()

			_, err := action.Prepare(context.Background(), &state, request)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "Missing required target attribute(s)")
		})
	}
}

func TestArgoRolloutRestartAction_Start(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	k8sClient, _, dynamicClient := getTestClient(stopCh)

	// Create rollout
	rollout := testRollout(nil)
	_, err := dynamicClient.Resource(ArgoRolloutGVR).Namespace("default").Create(
		context.Background(), rollout, metav1.CreateOptions{},
	)
	require.NoError(t, err)

	action := NewArgoRolloutRestartAction(k8sClient)
	state := ArgoRolloutRestartState{
		Namespace:   "default",
		ArgoRollout: "shop",
	}

	result, err := action.Start(context.Background(), &state)

	require.NoError(t, err)
	require.NotNil(t, result.Messages)
	require.Len(t, *result.Messages, 1)
	assert.Contains(t, (*result.Messages)[0].Message, "Restart triggered for Argo Rollout default/shop")

	updatedRollout, err := dynamicClient.Resource(ArgoRolloutGVR).Namespace("default").Get(
		context.Background(), "shop", metav1.GetOptions{},
	)
	require.NoError(t, err)

	restartAt, found, err := unstructured.NestedString(updatedRollout.Object, "spec", "restartAt")
	require.NoError(t, err)
	require.True(t, found, "restartAt should be set")

	_, err = time.Parse(time.RFC3339, restartAt)
	require.NoError(t, err, "restartAt should be a valid RFC3339 timestamp")
}

func TestArgoRolloutRestartAction_Start_RolloutNotFound(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	k8sClient, _, _ := getTestClient(stopCh)

	action := NewArgoRolloutRestartAction(k8sClient)
	state := ArgoRolloutRestartState{
		Namespace:   "default",
		ArgoRollout: "nonexistent",
	}

	_, err := action.Start(context.Background(), &state)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to restart Argo Rollout")
}
