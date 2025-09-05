package extstatefulset

import (
	"context"
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
	"testing"
	"time"
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
				"k8s.statefulset":  {"xyz"},
			},
		}),
	}

	clientset := testclient.NewClientset()
	_, err := clientset.
		AppsV1().
		StatefulSets("shop").
		Create(context.Background(), &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "xyz",
				Namespace: "shop",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: extutil.Ptr(int32(3)),
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: 3,
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	assert.Eventually(t, func() bool {
		return k8sclient.StatefulSetByNamespaceAndName("shop", "xyz") != nil
	}, time.Second, 100*time.Millisecond)

	action := NewStatefulSetPodCountCheckAction(k8sclient)
	state := action.NewEmptyState()

	// When
	result, err := action.Prepare(context.Background(), &state, request)

	// Then
	require.Nil(t, err)
	require.Nil(t, result)
	require.True(t, state.Timeout.After(time.Now()))
	require.Equal(t, extcommon.PodCountEqualsDesiredCount, state.PodCountCheckMode)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "xyz", state.Target)
	require.Equal(t, 3, state.InitialCount)
}

func TestStatusCheckStatefulSetNotFound(t *testing.T) {
	// Given
	state := extcommon.PodCountCheckState{
		Timeout:           time.Now().Add(time.Minute * 1),
		PodCountCheckMode: extcommon.PodCountMin1,
		Namespace:         "shop",
		Target:            "xyz",
	}

	clientset := testclient.NewClientset()
	_, err := clientset.
		AppsV1().
		StatefulSets("shop").
		Create(context.Background(), &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "123",
				Namespace: "shop",
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())

	action := NewStatefulSetPodCountCheckAction(k8sclient).(action_kit_sdk.ActionWithStatus[extcommon.PodCountCheckState])

	// When
	result, err := action.Status(context.Background(), &state)

	// Then
	require.EqualError(t, err, "StatefulSet xyz not found.")
	require.Nil(t, result)
}
