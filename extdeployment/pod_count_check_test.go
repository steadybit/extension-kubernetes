// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
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
	action := NewPodCountCheckAction()
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.TODO(), &state, request)

	// Then
	require.Nil(t, err)
	require.Nil(t, result)
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, "podCountMin1", state.PodCountCheckMode)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "checkout", state.Deployment)
}

func TestStatusCheckDeploymentNotFound(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountMin1",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.False(t, result.Completed)
	require.Equal(t, "Deployment checkout not found", result.Error.Title)
}

func TestStatusCheckPodCountMin1Success(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountMin1",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckPodCountMin1Fail(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * -1),
		PodCountCheckMode: "podCountMin1",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "checkout has no ready pods.", result.Error.Title)
}

func TestStatusCheckPodCountEqualsDesiredCountSuccess(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountEqualsDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckPodCountEqualsDesiredCountFail(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * -1),
		PodCountCheckMode: "podCountEqualsDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "checkout has only 1 of desired 2 pods ready.", result.Error.Title)
}

func TestStatusCheckPodCountLessThanDesiredCountSuccess(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: "podCountLessThanDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckPodCountLessThanDesiredCountFail(t *testing.T) {
	// Given
	state := PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * -1),
		PodCountCheckMode: "podCountLessThanDesiredCount",
		Namespace:         "shop",
		Deployment:        "checkout",
	}

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

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")

	// When
	result := statusPodCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "checkout has all 2 desired pods ready.", result.Error.Title)
}
