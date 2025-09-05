// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdeployment

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScaleDeploymentPreparesCommands(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":     100000,
			"replicaCount": 5,
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace":  {"demo"},
				"k8s.deployment": {"shop"},
			},
		}),
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	testClient := getTestClient(stopCh, &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "demo",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: extutil.Ptr(int32(2)),
		},
	})
	client.K8S = testClient
	assert.Eventually(t, func() bool {
		return testClient.DeploymentByNamespaceAndName("demo", "shop") != nil
	}, time.Second, 100*time.Millisecond)

	action := NewScaleDeploymentAction()
	state := action.NewEmptyState()

	// When
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, []string{"kubectl", "scale", "--replicas=5", "--current-replicas=2", "--namespace=demo", "deployment/shop"}, state.Opts.Command)
	require.Equal(t, []string{"kubectl", "scale", "--replicas=2", "--namespace=demo", "deployment/shop"}, state.Opts.RollbackCommand)
}
