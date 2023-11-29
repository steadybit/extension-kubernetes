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

func Test_Prepare(t *testing.T) {
	tests := []struct {
		name                 string
		podSpecContainerName string
		podSpecHostPID       bool
		configContainer      string
		wantState            CrashLoopState
		wantErr              string
	}{
		{
			name:                 "should fail if pod does not have container in spec",
			podSpecContainerName: "other",
			podSpecHostPID:       false,
			configContainer:      "example",
			wantErr:              "Container example not found in pod specification checkout-xyz1234",
		},
		{
			name:                 "should fail if pod has hostPID enabled",
			podSpecContainerName: "example",
			podSpecHostPID:       true,
			wantErr:              "Pod checkout-xyz1234 in namespace shop has hostPID enabled. This is not yet supported",
		},
		{
			name:                 "should return state for all container",
			podSpecContainerName: "example",
			podSpecHostPID:       false,
			wantState: CrashLoopState{
				Namespace: "shop",
				Pod:       "checkout-xyz1234",
			},
		},
		{
			name:                 "should return state for specific container",
			podSpecContainerName: "example",
			podSpecHostPID:       false,
			configContainer:      "example",
			wantState: CrashLoopState{
				Namespace: "shop",
				Pod:       "checkout-xyz1234",
				Container: "example",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
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
					HostPID: tt.podSpecHostPID,
					Containers: []corev1.Container{
						{Name: tt.podSpecContainerName},
					},
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

			request := action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"container": tt.configContainer,
				},
				Target: extutil.Ptr(action_kit_api.Target{
					Attributes: map[string][]string{
						"k8s.namespace": {"shop"},
						"k8s.pod.name":  {"checkout-xyz1234"},
					},
				}),
			}
			action := NewCrashLoopAction()
			state := action.NewEmptyState()

			// When
			_, err = action.Prepare(context.Background(), &state, request)

			// Then
			if tt.wantErr == "" {
				assert.Equal(t, tt.wantState, state)
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
