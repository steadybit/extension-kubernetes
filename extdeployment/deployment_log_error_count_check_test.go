// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extdeployment

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestDeploymentLogErrorCountCheck(t *testing.T) {
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "checkout"}}
	pod := func(name string, ownerUid types.UID, phase corev1.PodPhase) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: "shop", Labels: map[string]string{"app": "checkout"},
				OwnerReferences: []metav1.OwnerReference{{UID: ownerUid, Kind: "ReplicaSet", Name: "rs"}},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}},
			Status: corev1.PodStatus{Phase: phase},
		}
	}
	clientset := testclient.NewClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "shop", UID: "deployment-uid"},
			Spec:       appsv1.DeploymentSpec{Selector: selector},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "checkout-abc", Namespace: "shop", UID: "rs-uid",
				OwnerReferences: []metav1.OwnerReference{{UID: "deployment-uid", Kind: "Deployment", Name: "checkout"}},
			},
		},
		pod("checkout-1", "rs-uid", corev1.PodRunning),
		// failed pods keep their logs and are still read
		pod("checkout-2", "rs-uid", corev1.PodFailed),
		// same labels, but owned by another workload
		pod("other-1", "other-rs-uid", corev1.PodRunning),
	)

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sclient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), testutil.NewFakeDynamicClient())
	assert.Eventually(t, func() bool {
		return k8sclient.DeploymentByNamespaceAndName("shop", "checkout") != nil && len(k8sclient.Pods()) == 2
	}, time.Second, 100*time.Millisecond)

	action := NewDeploymentLogErrorCountCheckAction(k8sclient)
	description := action.Describe()
	assert.Equal(t, DeploymentLogErrorCountCheckActionId, description.Id)
	assert.Equal(t, "Deployment Error Log Count", description.Label)
	assert.Equal(t, DeploymentTargetType, description.TargetSelection.TargetType)
	assert.Equal(t, action_kit_api.Check, description.Kind)

	state := action.NewEmptyState()
	request := action_kit_api.PrepareActionRequestBody{
		Config: map[string]any{"duration": 10_000},
		Target: new(action_kit_api.Target{Attributes: map[string][]string{
			"k8s.namespace":  {"shop"},
			"k8s.deployment": {"checkout"},
		}}),
	}
	_, err := action.Prepare(context.Background(), &state, request)
	require.NoError(t, err)

	result, err := action.(action_kit_sdk.ActionWithStatus[extcommon.LogErrorCountCheckState]).Status(context.Background(), &state)
	require.NoError(t, err)
	require.NotNil(t, result.Metrics)
	var pods []string
	for _, m := range *result.Metrics {
		if *m.Name == extcommon.LogErrorCountMetricName {
			pods = append(pods, m.Metric["k8s.pod.name"])
			assert.Equal(t, "checkout", m.Metric["k8s.deployment"])
		}
	}
	assert.ElementsMatch(t, []string{"checkout-1", "checkout-2"}, pods)

	request.Target.Attributes["k8s.deployment"] = []string{"missing"}
	_, err = action.Prepare(context.Background(), &state, request)
	require.Error(t, err)
}
