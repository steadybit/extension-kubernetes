/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extevents

import (
	"context"
	"encoding/json"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func prepareTest(t *testing.T, stopCh chan struct{}) ([]byte, *client.Client) {
	reqJson := getStatusRequestBodyCheck(t, K8sEventsState{
		TimeoutEnd:    extutil.Ptr(time.Now().Add(time.Minute * 1).Unix()),
		LastEventTime: extutil.Ptr(time.Now().Add(-time.Minute * 1).Unix()),
	})

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
	return reqJson, client
}

func TestPrepareExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration": 1000 * 10,
		},
	}
	reqJson, err := json.Marshal(request)
	require.NoError(t, err)

	// When
	state, extErr := PrepareK8sEvents(reqJson)

	// Then
	require.Nil(t, extErr)
	require.True(t, *state.TimeoutEnd > time.Now().Unix())
	require.True(t, *state.LastEventTime <= time.Now().Unix())
}

func getStatusRequestBodyCheck(t *testing.T, state K8sEventsState) []byte {
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

func TestStatusEventsFound(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)

	reqJson, k8sClient := prepareTest(t, stopCh)

	// When
	result, timeout, _ := K8sLogsStatus(k8sClient, reqJson)

	// Then
	require.False(t, timeout)

	for _, message := range *(result.Messages) {
		require.Equal(t, "test", message.Message)
		require.Equal(t, "KUBERNETES_EVENTS", *message.Type)
		require.Equal(t, action_kit_api.MessageLevel("info"), *message.Level)
		require.Equal(t, action_kit_api.MessageFields(action_kit_api.MessageFields{"cluster-name": "unknown", "namespace": "shop", "object": "/", "reason": ""}), *message.Fields)
	}
}

func TestStopEventsFound(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)

	reqJson, k8sClient := prepareTest(t, stopCh)

	// When
	result, _ := K8sLogsStop(k8sClient, reqJson)

	// Then
	for _, message := range *(result.Messages) {
		require.Equal(t, "test", message.Message)
		require.Equal(t, "KUBERNETES_EVENTS", *message.Type)
		require.Equal(t, action_kit_api.MessageLevel("info"), *message.Level)
		require.Equal(t, action_kit_api.MessageFields(action_kit_api.MessageFields{"cluster-name": "unknown", "namespace": "shop", "object": "/", "reason": ""}), *message.Fields)
	}
}
