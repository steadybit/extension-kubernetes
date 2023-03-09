// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpodcount

import (
	"context"
	"encoding/json"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func getStatusRequestBody(t *testing.T, state PodCountMetricsState) []byte {
	var encodedState action_kit_api.ActionState
	err := extconversion.Convert(state, &encodedState)
	require.NoError(t, err)
	request := action_kit_api.ActionStatusRequestBody{
		State: encodedState,
	}
	reqJson, err := json.Marshal(request)
	require.NoError(t, err)
	return reqJson
}

func TestPrepareExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 1000 * 60,
		},
	}
	reqJson, err := json.Marshal(request)
	require.NoError(t, err)

	// When
	state, extErr := preparePodCountMetricsInternal(reqJson)

	// Then
	require.Nil(t, extErr)
	require.True(t, state.End.After(time.Now()))
}

func TestStatusReturnsMetrics(t *testing.T) {
	// Given
	reqJson := getStatusRequestBody(t, PodCountMetricsState{
		End:         time.Now().Add(time.Minute * -1),
		LastMetrics: make(map[string]int32),
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	extconfig.Config.ClusterName = "development"

	desiredCount := int32(5)
	currentCount := int32(3)
	availableCount := int32(2)
	readyCount := int32(1)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("default").
		Create(context.Background(), &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "v1",
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
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountMetricsInternal(client, reqJson)

	// Then
	require.Len(t, (*result.State)["LastMetrics"], 4)
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
