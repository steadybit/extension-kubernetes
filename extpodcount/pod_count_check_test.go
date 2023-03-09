// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpodcount

import (
	"context"
	"encoding/json"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func TestPrepareCheckExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":          1000 * 10,
			"podCountCheckMode": "podCountMin1",
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.cluster-name": {"test"},
				"k8s.namespace":    {"shop"},
				"k8s.deployment":   {"checkout"},
			},
		}),
	}
	reqJson, err := json.Marshal(request)
	require.NoError(t, err)

	// When
	state, extErr := preparePodCountCheckInternal(reqJson)

	// Then
	require.Nil(t, extErr)
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, "podCountMin1", state.PodCountCheckMode)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "checkout", state.Deployment)
}

func getStatusRequestBodyCheck(t *testing.T, state PodCountCheckState) []byte {
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

func TestStatusCheckDeploymentNotFound(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountMin1",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "xyz",
				Namespace: "shop",
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)
	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.False(t, result.Completed)
	require.Equal(t, "Deployment checkout not found", result.Error.Title)
}

func TestStatusCheckPodCountMin1Success(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountMin1",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	desiredCount := int32(1)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desiredCount,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.False(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckPodCountMin1Fail(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * -1),
		PodCountCheckMode: "podCountMin1",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	desiredCount := int32(1)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desiredCount,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 0,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "checkout has no ready pods.", result.Error.Title)
}

func TestStatusCheckPodCountEqualsDesiredCountSuccess(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountEqualsDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	desiredCount := int32(2)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desiredCount,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 2,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.False(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckPodCountEqualsDesiredCountFail(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * -1),
		PodCountCheckMode: "podCountEqualsDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	desiredCount := int32(2)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desiredCount,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "checkout has only 1 of desired 2 pods ready.", result.Error.Title)
}

func TestStatusCheckPodCountLessThanDesiredCountSuccess(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountLessThanDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	desiredCount := int32(2)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desiredCount,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.False(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckPodCountLessThanDesiredCountFail(t *testing.T) {
	// Given
	reqJson := getStatusRequestBodyCheck(t, PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * -1),
		PodCountCheckMode: "podCountLessThanDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	desiredCount := int32(2)

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desiredCount,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 2,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	client := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(client, reqJson)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "checkout has all 2 desired pods ready.", result.Error.Title)
}
