// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extstatefulset

import (
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestStatefulSetPodCountMetrics(t *testing.T) {
	t.Run("uses the spec replicas as desired count", func(t *testing.T) {
		metrics := statefulSetPodCountMetrics(&appsv1.StatefulSet{
			Spec:   appsv1.StatefulSetSpec{Replicas: new(int32(5))},
			Status: appsv1.StatefulSetStatus{Replicas: 4, ReadyReplicas: 3, AvailableReplicas: 2},
		})
		assert.Equal(t, &extcommon.PodCountMetrics{Desired: 5, Current: 4, Ready: 3, Available: 2}, metrics)
	})

	t.Run("treats missing spec replicas as zero", func(t *testing.T) {
		metrics := statefulSetPodCountMetrics(&appsv1.StatefulSet{})
		assert.Equal(t, &extcommon.PodCountMetrics{}, metrics)
	})
}

func TestPodCountCheckActionCallbacks(t *testing.T) {
	clientset := testclient.NewClientset(&appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "xyz", Namespace: "shop"},
		Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(3))},
		Status:     appsv1.StatefulSetStatus{Replicas: 3, ReadyReplicas: 2, AvailableReplicas: 1},
	})
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), testutil.NewFakeDynamicClient())
	assert.Eventually(t, func() bool {
		return k8sclient.StatefulSetByNamespaceAndName("shop", "xyz") != nil
	}, time.Second, 100*time.Millisecond)

	action := NewStatefulSetPodCountCheckAction(k8sclient).(*extcommon.PodCountCheckAction)

	target := action.GetTarget(action_kit_api.PrepareActionRequestBody{
		Target: &action_kit_api.Target{Attributes: map[string][]string{"k8s.statefulset": {"xyz"}}},
	})
	assert.Equal(t, "xyz", target)

	metrics, err := action.GetPodCountMetrics(k8sclient, "shop", "xyz")
	require.NoError(t, err)
	// The informer's transformStatefulSet keeps only Status.ReadyReplicas, so current and
	// available replicas are always reported as zero here.
	assert.Equal(t, &extcommon.PodCountMetrics{Desired: 3, Current: 0, Ready: 2, Available: 0}, metrics)

	metrics, err = action.GetPodCountMetrics(k8sclient, "shop", "missing")
	require.NoError(t, err)
	assert.Nil(t, metrics)

	_, _, err = action.GetDesiredAndCurrentPodCount(k8sclient, "shop", "missing")
	require.EqualError(t, err, "StatefulSet missing not found.")
}
