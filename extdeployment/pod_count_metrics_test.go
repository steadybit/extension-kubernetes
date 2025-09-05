// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdeployment

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestPrepareMetricsExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 1000 * 60,
		},
	}

	action := NewPodCountMetricsAction()
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.TODO(), &state, request)

	// Then
	require.Nil(t, err)
	require.Nil(t, result)
	require.True(t, state.End.After(time.Now()))
}

func TestStatusReturnsMetrics(t *testing.T) {
	// Given
	state := PodCountMetricsState{
		End:         time.Now().Add(time.Minute * -1),
		LastMetrics: make(map[string]int32),
	}

	extconfig.Config.ClusterName = "development"

	desiredCount := int32(5)
	currentCount := int32(3)
	availableCount := int32(2)
	readyCount := int32(1)

	clientset := testclient.NewClientset(&appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &desiredCount,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          currentCount,
			AvailableReplicas: availableCount,
			ReadyReplicas:     readyCount,
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	client := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())

	// When
	result := statusPodCountMetricsInternal(client, &state)

	// Then
	require.Len(t, state.LastMetrics, 4)
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
	require.Len(t, *result.Metrics, 4)
}

func TestCreateMetrics(t *testing.T) {
	// Given
	now := time.Now()
	extconfig.Config.ClusterName = "development"
	desiredCount := int32(5)
	currentCount := int32(3)
	availableCount := int32(2)
	readyCount := int32(1)
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &desiredCount,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          currentCount,
			AvailableReplicas: availableCount,
			ReadyReplicas:     readyCount,
		},
	}

	// When
	metrics := toMetrics(&deployment, now)

	// Then
	for _, metric := range metrics {
		require.Equal(t, "development", metric.Metric["k8s.cluster-name"])
		require.Equal(t, "default", metric.Metric["k8s.namespace"])
		require.Equal(t, "shop", metric.Metric["k8s.deployment"])
		require.Equal(t, now, metric.Timestamp)

		if *metric.Name == "replicas_desired_count" {
			require.Equal(t, float64(desiredCount), metric.Value)
		} else if *metric.Name == "replicas_current_count" {
			require.Equal(t, float64(currentCount), metric.Value)
		} else if *metric.Name == "replicas_ready_count" {
			require.Equal(t, float64(readyCount), metric.Value)
		} else if *metric.Name == "replicas_available_count" {
			require.Equal(t, float64(availableCount), metric.Value)
		} else {
			t.Fail()
			t.Logf("Unexpected metric %s", *metric.Name)
		}
	}
}
