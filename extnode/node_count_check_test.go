// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extnode

import (
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestPrepareCheckExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":           1000 * 10,
			"nodeCountCheckMode": "nodeCountAtLeast",
			"nodeCount":          2,
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.cluster-name": {"test"},
			},
		}),
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)
	action := NewNodeCountCheckAction()
	state := action.NewEmptyState()

	// When
	_, err := prepareNodeCountCheckInternal(k8sclient, &state, request)
	require.NoError(t, err)

	// Then
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, "nodeCountAtLeast", state.NodeCountCheckMode)
	require.Equal(t, "test", state.Cluster)
	require.Equal(t, 1, state.InitialNodeCount)
	require.Equal(t, 2, state.NodeCount)
}

func TestStatusCheckNodeCountAtLeastSuccess(t *testing.T) {
	// Given
	state := NodeCountCheckState{
		Timeout:            time.Now().Add(time.Minute * 1),
		NodeCountCheckMode: "nodeCountAtLeast",
		Cluster:            "test",
		NodeCount:          2,
		InitialNodeCount:   1,
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	}, &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node2",
			GenerateName: "node2",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)

	// When
	result := statusNodeCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckNodeCountAtFail(t *testing.T) {
	// Given
	state := NodeCountCheckState{
		Timeout:            time.Now().Add(time.Minute * -1),
		NodeCountCheckMode: "nodeCountAtLeast",
		Cluster:            "test",
		NodeCount:          2,
		InitialNodeCount:   1,
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)

	// When
	result := statusNodeCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "test has not enough ready nodes.", result.Error.Title)
}

func TestStatusCheckNodeCountDecreasedBySuccess(t *testing.T) {
	// Given
	state := NodeCountCheckState{
		Timeout:            time.Now().Add(time.Minute * 1),
		NodeCountCheckMode: "nodeCountDecreasedBy",
		Cluster:            "test",
		NodeCount:          2,
		InitialNodeCount:   3,
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)

	// When
	result := statusNodeCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckNodeCountDecreasedByFail(t *testing.T) {
	// Given
	state := NodeCountCheckState{
		Timeout:            time.Now().Add(time.Minute * -1),
		NodeCountCheckMode: "nodeCountDecreasedBy",
		Cluster:            "test",
		NodeCount:          2,
		InitialNodeCount:   3,
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	}, &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node2",
			GenerateName: "node2",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)

	// When
	result := statusNodeCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "test has 2 of desired 1 nodes ready.", result.Error.Title)
}

func TestStatusCheckNodeCountIncreasedBySuccess(t *testing.T) {
	// Given
	state := NodeCountCheckState{
		Timeout:            time.Now().Add(time.Minute * 1),
		NodeCountCheckMode: "nodeCountIncreasedBy",
		Cluster:            "test",
		NodeCount:          2,
		InitialNodeCount:   0,
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	}, &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node2",
			GenerateName: "node2",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)

	// When
	result := statusNodeCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Nil(t, result.Error)
}

func TestStatusCheckNodeCountIncreasedByFail(t *testing.T) {
	// Given
	state := NodeCountCheckState{
		Timeout:            time.Now().Add(time.Minute * -1),
		NodeCountCheckMode: "nodeCountIncreasedBy",
		Cluster:            "test",
		NodeCount:          2,
		InitialNodeCount:   0,
	}

	clientset := testclient.NewClientset(&corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         "node1",
			GenerateName: "node1",
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	dynamicClient := testutil.NewFakeDynamicClient()
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)

	// When
	result := statusNodeCountCheckInternal(k8sclient, &state)

	// Then
	require.True(t, result.Completed)
	require.Equal(t, "test has only 1 of desired 2 nodes ready.", result.Error.Title)
}
