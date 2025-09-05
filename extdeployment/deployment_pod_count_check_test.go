// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdeployment

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestPrepareCheckExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":          1000 * 10,
			"podCountCheckMode": extcommon.PodCountEqualsDesiredCount,
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.cluster-name": {"test"},
				"k8s.namespace":    {"shop"},
				"k8s.deployment":   {"checkout"},
			},
		}),
	}

	clientset := testclient.NewClientset(&appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "checkout",
			Namespace: "shop",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: extutil.Ptr(int32(3)),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	assert.Eventually(t, func() bool {
		return k8sclient.DeploymentByNamespaceAndName("shop", "checkout") != nil
	}, time.Second, 100*time.Millisecond)

	action := NewDeploymentPodCountCheckAction(k8sclient)
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.Background(), &state, request)

	// Then
	require.Nil(t, err)
	require.Nil(t, result)
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, extcommon.PodCountEqualsDesiredCount, state.PodCountCheckMode)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "checkout", state.Target)
	require.Equal(t, 3, state.InitialCount)
}

func TestStatusCheckDeploymentNotFound(t *testing.T) {
	// Given
	state := extcommon.PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: extcommon.PodCountMin1,
		Namespace:         "shop",
		Target:            "checkout",
	}

	clientset := testclient.NewClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "xyz",
			Namespace: "shop",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())

	action := NewDeploymentPodCountCheckAction(k8sclient).(action_kit_sdk.ActionWithStatus[extcommon.PodCountCheckState])

	// When
	result, err := action.Status(context.Background(), &state)

	// Then
	require.EqualError(t, err, "Deployment checkout not found.")
	require.Nil(t, result)
}
