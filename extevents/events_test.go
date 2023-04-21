/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extevents

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func TestPrepareExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 1000 * 10,
		},
	}

	action := NewK8sEventsAction()
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.TODO(), &state, request)

	// Then
	require.Nil(t, result)
	require.Nil(t, err)
	require.True(t, *state.TimeoutEnd > time.Now().Unix())
	require.True(t, *state.LastEventTime <= time.Now().Unix())
}

func TestStatusEventsFound(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)

	state, k8sClient := prepareTest(t, stopCh)

	// When
	result := statusInternal(k8sClient, state)

	// Then
	for _, message := range *(result.Messages) {
		require.Equal(t, "test", message.Message)
		require.Equal(t, "KUBERNETES_EVENTS", *message.Type)
		require.Equal(t, action_kit_api.MessageLevel("info"), *message.Level)
		require.Equal(t, action_kit_api.MessageFields{"cluster-name": "unknown", "namespace": "shop", "object": "/", "reason": ""}, *message.Fields)
	}
}

func TestStopEventsFound(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)

	state, k8sClient := prepareTest(t, stopCh)

	// When
	result := stopInternal(k8sClient, state)

	// Then
	for _, message := range *(result.Messages) {
		require.Equal(t, "test", message.Message)
		require.Equal(t, "KUBERNETES_EVENTS", *message.Type)
		require.Equal(t, action_kit_api.MessageLevel("info"), *message.Level)
		require.Equal(t, action_kit_api.MessageFields{"cluster-name": "unknown", "namespace": "shop", "object": "/", "reason": ""}, *message.Fields)
	}
}

func prepareTest(t *testing.T, stopCh chan struct{}) (*K8sEventsState, *client.Client) {
	state := K8sEventsState{
		TimeoutEnd:    extutil.Ptr(time.Now().Add(time.Minute * 1).Unix()),
		LastEventTime: extutil.Ptr(time.Now().Add(-time.Minute * 1).Unix()),
	}

	clientset := testclient.NewSimpleClientset()
	_, err := clientset.
		CoreV1().
		Events("shop").
		Create(context.Background(), &corev1.Event{
			LastTimestamp: metav1.Time{Time: time.Now()},
			Message:       "test",
			Type:          "Normal",
		}, metav1.CreateOptions{})

	require.NoError(t, err)
	client := client.CreateClient(clientset, stopCh, "")
	return &state, client
}
