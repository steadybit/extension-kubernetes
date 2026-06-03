// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extargorollout

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScaleArgoRolloutPreparesCommands(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]any{
			"duration":     100000,
			"replicaCount": 5,
		},
		Target: new(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace":    {"default"},
				"k8s.argo-rollout": {"shop"},
			},
		}),
	}
	stopCh := make(chan struct{})
	defer close(stopCh)

	k8sClient, _, dynamicClient := getTestClient(stopCh)
	// testRollout() has spec.replicas = 3 (see rollout_discovery_test.go)
	rollout := testRollout(nil)
	_, err := dynamicClient.Resource(client.ArgoRolloutGVR).Namespace("default").Create(
		context.Background(), rollout, metav1.CreateOptions{},
	)
	require.NoError(t, err)

	client.K8S = k8sClient
	assert.Eventually(t, func() bool {
		return k8sClient.ArgoRolloutByNamespaceAndName("default", "shop") != nil
	}, time.Second, 100*time.Millisecond)

	action := NewScaleArgoRolloutAction()
	state := action.NewEmptyState()

	// When
	_, err = action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, []string{"kubectl", "scale", "--replicas=5", "--current-replicas=3", "--namespace=default", "rollout/shop"}, state.Opts.Command)
	require.Equal(t, []string{"kubectl", "scale", "--replicas=3", "--namespace=default", "rollout/shop"}, state.Opts.RollbackCommand)
}
