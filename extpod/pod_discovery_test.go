// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extpod

import (
	"context"
	"testing"
	"time"

	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_getDiscoveredPods(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client := getTestClient(stopCh, &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "shop",
				},
			},
			Labels: map[string]string{
				"best-city": "kevelaer",
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					ContainerID: "crio://abcdef",
					Name:        "MrFancyPants",
					Image:       "nginx",
				},
			},
		},
		Spec: v1.PodSpec{
			NodeName: "worker-1",
		},
	}, &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker-1",
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalDNS,
					Address: "worker-1.internal",
				},
			},
		},
	}, &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
		},
	})
	extconfig.Config.ClusterName = "development"
	extconfig.Config.LabelFilter = []string{"secret-label"}

	d := &podDiscovery{k8s: client}
	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 1)
	}, 5*time.Second, 100*time.Millisecond)

	// Then
	targets, _ := d.DiscoverTargets(context.Background())
	require.Len(t, targets, 1)
	target := targets[0]
	assert.Equal(t, "development/default/shop-pod", target.Id)
	assert.Equal(t, "shop-pod", target.Label)
	assert.Equal(t, PodTargetType, target.TargetType)
	assert.Equal(t, map[string][]string{
		"host.hostname":             {"worker-1"},
		"host.domainname":           {"worker-1.internal"},
		"k8s.cluster-name":          {"development"},
		"k8s.container.id":          {"crio://abcdef"},
		"k8s.container.id.stripped": {"abcdef"},
		"k8s.deployment":            {"shop"},
		"k8s.workload-type":         {"deployment"},
		"k8s.workload-owner":        {"shop"},
		"k8s.label.best-city":       {"kevelaer"},
		"k8s.namespace":             {"default"},
		"k8s.node.name":             {"worker-1"},
		"k8s.pod.name":              {"shop-pod"},
	}, target.Attributes)
}

func Test_getDiscoveredPods_ignore_empty_container_ids(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client := getTestClient(stopCh, &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-pod",
			Namespace: "default",
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:  "MrFancyPants",
					Image: "nginx",
				},
			},
		},
		Spec: v1.PodSpec{
			NodeName: "worker-1",
		},
	}, &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker-1",
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalDNS,
					Address: "worker-1.internal",
				},
			},
		},
	})
	extconfig.Config.ClusterName = "development"
	extconfig.Config.LabelFilter = []string{"secret-label"}

	d := &podDiscovery{k8s: client}

	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 1)
	}, 5*time.Second, 100*time.Millisecond)

	// Then
	targets, _ := d.DiscoverTargets(context.Background())
	require.Len(t, targets, 1)
	target := targets[0]
	assert.Equal(t, "development/default/shop-pod", target.Id)
	assert.Equal(t, "shop-pod", target.Label)
	assert.Equal(t, PodTargetType, target.TargetType)
	assert.Equal(t, map[string][]string{
		"host.domainname":  {"worker-1.internal"},
		"host.hostname":    {"worker-1"},
		"k8s.cluster-name": {"development"},
		"k8s.namespace":    {"default"},
		"k8s.node.name":    {"worker-1"},
		"k8s.pod.name":     {"shop-pod"},
	}, target.Attributes)
}

func Test_getDiscoveredPodsShouldIgnoreLabeledPods(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client := getTestClient(stopCh, &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "shop",
				},
			},
			Labels: map[string]string{
				"best-city": "kevelaer",
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					ContainerID: "crio://abcdef",
					Name:        "MrFancyPants",
					Image:       "nginx",
				},
			},
		},
		Spec: v1.PodSpec{
			NodeName: "worker-1",
		},
	}, &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-ignore",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "shop",
				},
			},
			Labels: map[string]string{
				"best-city":                        "kevelaer",
				"steadybit.com/discovery-disabled": "true",
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					ContainerID: "crio://abcdef",
					Name:        "MrFancyPants",
					Image:       "nginx",
				},
			},
		},
		Spec: v1.PodSpec{
			NodeName: "worker-1",
		},
	})
	extconfig.Config.ClusterName = "development"
	extconfig.Config.LabelFilter = []string{"secret-label"}

	d := &podDiscovery{k8s: client}
	// Then
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 1)
	}, 5*time.Second, 100*time.Millisecond)

}

func getTestClient(stopCh <-chan struct{}, objects ...runtime.Object) *client.Client {
	return client.CreateClient(testclient.NewClientset(objects...), stopCh, "", client.MockAllPermitted())
}
