// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extnode

import (
	"context"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func Test_nodeDiscovery(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"
	extconfig.Config.LabelFilter = []string{"secret-label"}

	_, err := clientset.
		CoreV1().
		Nodes().
		Create(context.Background(), &v1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-123",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = clientset.CoreV1().
		Pods("default").
		Create(context.Background(), &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shop-pod-11",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "Deployment",
						Name: "shop",
					},
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
				NodeName: "node-123",
				Containers: []v1.Container{
					{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "Always",
					},
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = clientset.CoreV1().
		Pods("default").
		Create(context.Background(), &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shop-pod-23-ignored-other-host",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "Deployment",
						Name: "shop",
					},
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				ContainerStatuses: []v1.ContainerStatus{
					{
						ContainerID: "crio://ignored",
						Name:        "MrFancyPants",
						Image:       "nginx",
					},
				},
			},
			Spec: v1.PodSpec{
				NodeName: "node-456",
				Containers: []v1.Container{
					{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "Always",
					},
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, err)
	_, err = clientset.
		AppsV1().
		Deployments("default").
		Create(context.Background(), &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shop",
				Namespace: "default",
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	d := &nodeDiscovery{k8s: client}
	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 1)
	}, 1*time.Second, 100*time.Millisecond)

	// Then
	targets, _ := d.DiscoverTargets(context.Background())
	require.Len(t, targets, 1)
	target := targets[0]
	assert.Equal(t, "node-123", target.Id)
	assert.Equal(t, "node-123", target.Label)
	assert.Equal(t, NodeTargetType, target.TargetType)
	assert.Equal(t, map[string][]string{
		"host.hostname":             {"node-123"},
		"host.domainname":           {"node-123"},
		"k8s.cluster-name":          {"development"},
		"k8s.container.id":          {"crio://abcdef"},
		"k8s.container.id.stripped": {"abcdef"},
		"k8s.deployment":            {"shop"},
		"k8s.distribution":          {"kubernetes"},
		"k8s.namespace":             {"default"},
		"k8s.node.name":             {"node-123"},
		"k8s.pod.name":              {"shop-pod-11"},
		"k8s.label.label1":          {"value1"},
		"k8s.label.label2":          {"value2"},
	}, target.Attributes)
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	client := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	return client, clientset
}
