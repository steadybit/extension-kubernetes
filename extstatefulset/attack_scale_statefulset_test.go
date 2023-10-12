package extstatefulset

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestScaleStatefulSetPreparesCommands(t *testing.T) {
	// Given
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]interface{}{
			"duration":     100000,
			"replicaCount": 5,
		},
		Target: extutil.Ptr(action_kit_api.Target{
			Attributes: map[string][]string{
				"k8s.namespace":   {"demo"},
				"k8s.statefulset": {"shop"},
			},
		}),
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	testClient, clientset := getTestClient(stopCh)
	ss := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "demo",
			Labels: map[string]string{
				"best-city":    "Kevelaer",
				"secret-label": "secret-value",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: extutil.Ptr(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"best-city": "kevelaer",
				},
			}),
			Replicas: extutil.Ptr(int32(2)),
		},
	}
	_, err := clientset.
		AppsV1().
		StatefulSets("demo").
		Create(context.Background(), ss, metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Eventually(t, func() bool {
		return testClient.StatefulSetByNamespaceAndName("demo", "shop") != nil
	}, time.Second, 100*time.Millisecond)

	client.K8S = testClient

	action := NewScaleStatefulSetAction()
	state := action.NewEmptyState()

	// When
	_, err = action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	// Then
	require.Equal(t, []string{"kubectl", "scale", "--replicas=5", "--current-replicas=2", "--namespace=demo", "statefulset/shop"}, state.Opts.Command)
	require.Equal(t, []string{"kubectl", "scale", "--replicas=2", "--namespace=demo", "statefulset/shop"}, *state.Opts.RollbackCommand)
}
