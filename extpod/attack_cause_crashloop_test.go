package extpod

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestCrashLoopExtractsState(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"container": "example",
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace": {"shop"},
				"k8s.pod.name":  {"checkout-xyz1234"},
			},
		}),
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	testClient, clientset := getTestClient(stopCh)
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "checkout-xyz1234",
			Namespace: "shop",
		},
		Spec: corev1.PodSpec{
			HostPID: false,
		},
	}
	_, err := clientset.
		CoreV1().
		Pods("shop").
		Create(context.Background(), pod, metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Eventually(t, func() bool {
		return testClient.PodByNamespaceAndName("shop", "checkout-xyz1234") != nil
	}, time.Second, 100*time.Millisecond)

	client.K8S = testClient

	action := NewCrashLoopAction()
	state := action.NewEmptyState()

	// When
	_, err = action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, "checkout-xyz1234", state.Pod)
	require.Equal(t, "shop", state.Namespace)
	require.Equal(t, "example", *state.Container)
}
