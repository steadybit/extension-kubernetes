// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/assert"
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
			"duration":            1000 * 10,
			"podCountCheckMode":   "podCountIncreased",
			"expectedChangeCount": 1,
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

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 3,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")
	assert.Eventually(t, func() bool {
		return k8sclient.DeploymentByNamespaceAndName("shop", "checkout") != nil
	}, time.Second, 100*time.Millisecond)

	// When
	result, err := preparePodCountCheckInternal(k8sclient, &state, request)

	// Then
	require.Nil(t, err)
	require.Nil(t, result)
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, "podCountIncreased", state.PodCountCheckMode)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "checkout", state.Deployment)
	require.Equal(t, 3, state.InitialCount)
	require.Equal(t, 1, *state.ExpectedChangeCount)
}

func TestPrepareCheckExtractsStateSupportOmittingExpectedChangeCount(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":          1000 * 10,
			"podCountCheckMode": "podCountIncreased",
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

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		AppsV1().
		Deployments("shop").
		Create(context.Background(), &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout",
				Namespace: "shop",
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 3,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "")
	assert.Eventually(t, func() bool {
		return k8sclient.DeploymentByNamespaceAndName("shop", "checkout") != nil
	}, time.Second, 100*time.Millisecond)

	// When
	result, err := preparePodCountCheckInternal(k8sclient, &state, request)

	// Then
	require.Nil(t, err)
	require.Nil(t, result)
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, "podCountIncreased", state.PodCountCheckMode)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "checkout", state.Deployment)
	require.Equal(t, 3, state.InitialCount)
	require.Nil(t, state.ExpectedChangeCount)
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

func Test_statusPodCountCheckInternal(t *testing.T) {
	type preparedState struct {
		podCountCheckMode   string
		initialCount        int
		expectedChangeCount *int
	}
	tests := []struct {
		name               string
		preparedState      preparedState
		readyCount         int
		desiredCount       int
		wantedErrorMessage *string
	}{
		{
			name: "podCountMin1Success",
			preparedState: preparedState{
				podCountCheckMode: podCountMin1,
			},
			readyCount:         1,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountMin1Failure",
			preparedState: preparedState{
				podCountCheckMode: podCountMin1,
			},
			readyCount:         0,
			wantedErrorMessage: extutil.Ptr("checkout has no ready pods."),
		},
		{
			name: "podCountEqualsDesiredCountSuccess",
			preparedState: preparedState{
				podCountCheckMode: podCountEqualsDesiredCount,
			},
			readyCount:         2,
			desiredCount:       2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountEqualsDesiredCountFailure",
			preparedState: preparedState{
				podCountCheckMode: podCountEqualsDesiredCount,
			},
			readyCount:         1,
			desiredCount:       2,
			wantedErrorMessage: extutil.Ptr("checkout has only 1 of desired 2 pods ready."),
		},
		{
			name: "podCountLessThanDesiredCountSuccess",
			preparedState: preparedState{
				podCountCheckMode: podCountLessThanDesiredCount,
			},
			readyCount:         1,
			desiredCount:       2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountLessThanDesiredCountFailure",
			preparedState: preparedState{
				podCountCheckMode: podCountLessThanDesiredCount,
			},
			readyCount:         2,
			desiredCount:       2,
			wantedErrorMessage: extutil.Ptr("checkout has all 2 desired pods ready."),
		},
		{
			name: "podCountIncreasedSuccess",
			preparedState: preparedState{
				podCountCheckMode: podCountIncreased,
				initialCount:      1,
			},
			readyCount:         2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountIncreasedFailure",
			preparedState: preparedState{
				podCountCheckMode: podCountIncreased,
				initialCount:      2,
			},
			readyCount:         2,
			wantedErrorMessage: extutil.Ptr("checkout's pod count didn't increase. Initial count: 2, current count: 2."),
		},
		{
			name: "podCountIncreasedByXSuccess",
			preparedState: preparedState{
				podCountCheckMode:   podCountIncreased,
				initialCount:        1,
				expectedChangeCount: extutil.Ptr(3),
			},
			readyCount:         4,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountIncreasedByXFailure",
			preparedState: preparedState{
				podCountCheckMode:   podCountIncreased,
				initialCount:        1,
				expectedChangeCount: extutil.Ptr(3),
			},
			readyCount:         3,
			wantedErrorMessage: extutil.Ptr("checkout's pod count didn't increase by 3. Initial count: 1, current count: 3."),
		},
		{
			name: "podCountDecreasedSuccess",
			preparedState: preparedState{
				podCountCheckMode: podCountDecreased,
				initialCount:      2,
			},
			readyCount:         1,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountDecreasedFailure",
			preparedState: preparedState{
				podCountCheckMode: podCountDecreased,
				initialCount:      2,
			},
			readyCount:         2,
			wantedErrorMessage: extutil.Ptr("checkout's pod count didn't decrease. Initial count: 2, current count: 2."),
		},
		{
			name: "podCountDecreasedByXSuccess",
			preparedState: preparedState{
				podCountCheckMode:   podCountDecreased,
				initialCount:        4,
				expectedChangeCount: extutil.Ptr(2),
			},
			readyCount:         2,
			wantedErrorMessage: nil,
		},
		{
			name: "podCountDecreasedByXFailure",
			preparedState: preparedState{
				podCountCheckMode:   podCountDecreased,
				initialCount:        4,
				expectedChangeCount: extutil.Ptr(2),
			},
			readyCount:         3,
			wantedErrorMessage: extutil.Ptr("checkout's pod count didn't decrease by 2. Initial count: 4, current count: 3."),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			state := PodCountCheckState{
				Timeout:             time.Now().Add(time.Minute * -1),
				PodCountCheckMode:   tt.preparedState.podCountCheckMode,
				Namespace:           "shop",
				Deployment:          "checkout",
				InitialCount:        tt.preparedState.initialCount,
				ExpectedChangeCount: tt.preparedState.expectedChangeCount,
			}

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
						Replicas: extutil.Ptr(int32(tt.desiredCount)),
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: int32(tt.readyCount),
					},
				}, metav1.CreateOptions{})
			require.NoError(t, err)

			stopCh := make(chan struct{})
			defer close(stopCh)
			k8sclient := client.CreateClient(clientset, stopCh, "")

			result := statusPodCountCheckInternal(k8sclient, &state)
			require.True(t, result.Completed)
			if tt.wantedErrorMessage != nil {
				assert.Equalf(t, *tt.wantedErrorMessage, result.Error.Title, "Error message should be %s", *tt.wantedErrorMessage)
			} else {
				assert.Nil(t, result.Error, "Error should be nil")
			}
		})
	}
}
