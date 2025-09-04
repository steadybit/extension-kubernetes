// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extreplicaset

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_replicasetDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		configModifier            func(*extconfig.Specification)
		pods                      []*v1.Pod
		nodes                     []*v1.Node
		replicaset                *appsv1.ReplicaSet
		service                   *v1.Service
		expectedAttributesExactly map[string][]string
		expectedAttributes        map[string][]string
		expectedAttributesAbsence []string
	}{
		{
			name: "should discover basic attributes",
			pods: []*v1.Pod{testPod("aaaaa", nil), testPod("bbbbb", func(pod *v1.Pod) {
				pod.Spec.NodeName = "worker-2"
			})},
			nodes:      []*v1.Node{testNode("worker-1"), testNode("worker-2")},
			replicaset: testReplicaSet(nil),
			expectedAttributesExactly: map[string][]string{
				"host.hostname":                  {"worker-1", "worker-2"},
				"host.domainname":                {"worker-1.internal", "worker-2.internal"},
				"k8s.namespace":                  {"default"},
				"k8s.replicaset":                 {"shop"},
				"k8s.deployment":                 {"shop"},
				"k8s.workload-type":              {"deployment"},
				"k8s.workload-owner":             {"shop"},
				"k8s.replicaset.label.best-city": {"Kevelaer"},
				"k8s.label.best-city":            {"Kevelaer"},
				"k8s.specification.replicas":     {"3"},
				"k8s.cluster-name":               {"development"},
				"k8s.pod.name":                   {"shop-pod-aaaaa", "shop-pod-bbbbb"},
				"k8s.container.id":               {"crio://abcdef-aaaaa", "crio://abcdef-bbbbb"},
				"k8s.container.id.stripped":      {"abcdef-aaaaa", "abcdef-bbbbb"},
				"k8s.distribution":               {"kubernetes"},
				"k8s.container.name":             {"nginx", "shop"},
				"k8s.replicaset.revision":        {"15"},
			},
		},
		{
			name:       "hostnames should be unique and not duplicated",
			nodes:      []*v1.Node{testNode("worker-1")},
			pods:       []*v1.Pod{testPod("aaaaa", nil), testPod("bbbbb", nil)},
			replicaset: testReplicaSet(nil),
			expectedAttributes: map[string][]string{
				"host.hostname":   {"worker-1"},
				"host.domainname": {"worker-1.internal"},
			},
		},
		{
			name:       "should add service name",
			pods:       []*v1.Pod{testPod("aaaaa", nil)},
			replicaset: testReplicaSet(nil),
			service:    testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.service.name": {"shop-kevelaer"},
			},
		},
		{
			name:       "should ignore empty container ids",
			pods:       []*v1.Pod{testPod("aaaaa", func(pod *v1.Pod) { pod.Status.ContainerStatuses[0].ContainerID = "" })},
			replicaset: testReplicaSet(nil),
			expectedAttributesAbsence: []string{
				"k8s.container.id",
				"k8s.container.id.stripped",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stopCh := make(chan struct{})
			defer close(stopCh)
			client, clientset := getTestClient(stopCh)
			extconfig.Config.ClusterName = "development"
			extconfig.Config.LabelFilter = []string{"secret-label"}
			extconfig.Config.DiscoveryMaxPodCount = 50
			extconfig.Config.AdviceSingleReplicaMinReplicas = 2
			if tt.configModifier != nil {
				tt.configModifier(&extconfig.Config)
			}

			for _, pod := range tt.pods {
				_, err := clientset.CoreV1().
					Pods("default").
					Create(context.Background(), pod, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			for _, node := range tt.nodes {
				_, err := clientset.CoreV1().
					Nodes().
					Create(context.Background(), node, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			_, err := clientset.
				AppsV1().
				ReplicaSets("default").
				Create(context.Background(), tt.replicaset, metav1.CreateOptions{})
			require.NoError(t, err)

			tt.replicaset.Name = "shop-1"
			tt.replicaset.Annotations["deployment.kubernetes.io/revision"] = "13"
			_, err = clientset.
				AppsV1().
				ReplicaSets("default").
				Create(context.Background(), tt.replicaset, metav1.CreateOptions{})
			require.NoError(t, err)

			tt.replicaset.Name = "shop-2"
			tt.replicaset.Annotations["deployment.kubernetes.io/revision"] = "14"
			_, err = clientset.
				AppsV1().
				ReplicaSets("default").
				Create(context.Background(), tt.replicaset, metav1.CreateOptions{})
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

			if tt.service != nil {
				_, err := clientset.CoreV1().
					Services("default").
					Create(context.Background(), tt.service, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			d := &replicasetDiscovery{k8s: client}
			// When
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				ed, _ := d.DiscoverTargets(context.Background())
				assert.Len(c, ed, 1)
			}, 5*time.Second, 100*time.Millisecond)

			// Then
			targets, _ := d.DiscoverTargets(context.Background())
			require.Len(t, targets, 1)
			target := targets[0]
			assert.Equal(t, "development/default/shop", target.Id)
			assert.Equal(t, "shop", target.Label)
			assert.Equal(t, ReplicaSetTargetType, target.TargetType)
			if len(tt.expectedAttributesExactly) > 0 {
				for _, v := range target.Attributes {
					sort.Strings(v)
				}
				assert.Equal(t, tt.expectedAttributesExactly, target.Attributes)
			}
			if len(tt.expectedAttributes) > 0 {
				for k, v := range tt.expectedAttributes {
					attributeValues := target.Attributes[k]
					sort.Strings(attributeValues)
					assert.Equal(t, v, attributeValues, "attribute %s", k)
				}
			}
			if len(tt.expectedAttributesAbsence) > 0 {
				for _, k := range tt.expectedAttributesAbsence {
					assert.NotContains(t, target.Attributes, k)
				}
			}
		})
	}
}

func testNode(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalDNS,
					Address: fmt.Sprintf("%s.internal", name),
				},
			},
		},
	}
}

func testReplicaSet(modifier func(*appsv1.ReplicaSet)) *appsv1.ReplicaSet {
	replicaset := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: "shop",
				},
			},
			Labels: map[string]string{
				"best-city":    "Kevelaer",
				"secret-label": "secret-value",
			},
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "15",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: extutil.Ptr(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"best-city": "kevelaer",
				},
			}),
			Replicas: extutil.Ptr(int32(3)),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"best-city": "Kevelaer",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "nginx",
							Image:           "nginx:v1.2.3",
							ImagePullPolicy: "Always",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									v1.ResourceMemory:           *resource.NewQuantity(500, resource.DecimalSI),
									v1.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									v1.ResourceMemory:           *resource.NewQuantity(250, resource.DecimalSI),
									v1.ResourceEphemeralStorage: *resource.NewQuantity(500, resource.DecimalSI),
								},
							},
							LivenessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{},
							},
							ReadinessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{},
							},
						},
						{
							Name:            "shop",
							Image:           "shop-container:v5",
							ImagePullPolicy: "Always",
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									v1.ResourceMemory:           *resource.NewQuantity(500, resource.DecimalSI),
									v1.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									v1.ResourceMemory:           *resource.NewQuantity(250, resource.DecimalSI),
									v1.ResourceEphemeralStorage: *resource.NewQuantity(500, resource.DecimalSI),
								},
							},
							LivenessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{},
							},
							ReadinessProbe: &v1.Probe{
								ProbeHandler: v1.ProbeHandler{},
							},
						},
					},
				},
			},
		},
	}
	if modifier != nil {
		modifier(replicaset)
	}
	return replicaset
}

func testPod(nameSuffix string, modifier func(*v1.Pod)) *v1.Pod {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("shop-pod-%s", nameSuffix),
			Namespace: "default",
			Labels: map[string]string{
				"best-city": "kevelaer",
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					ContainerID: fmt.Sprintf("crio://abcdef-%s", nameSuffix),
					Name:        "MrFancyPants",
					Image:       "nginx",
				},
			},
		},
		Spec: v1.PodSpec{
			NodeName: "worker-1",
		},
	}
	if modifier != nil {
		modifier(pod)
	}
	return pod
}

func testService(modifier func(service *v1.Service)) *v1.Service {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-kevelaer",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"best-city": "Kevelaer",
			},
		},
	}
	if modifier != nil {
		modifier(service)
	}
	return service
}

func Test_getDiscoveredReplicaSetsShouldIgnoreLabeledReplicaSets(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DiscoveryMaxPodCount = 50

	_, err := clientset.
		AppsV1().
		ReplicaSets("default").
		Create(context.Background(), testReplicaSet(nil), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = clientset.
		AppsV1().
		ReplicaSets("default").
		Create(context.Background(), testReplicaSet(func(replicaset *appsv1.ReplicaSet) {
			replicaset.ObjectMeta.Name = "shop-ignore"
			replicaset.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
		}), metav1.CreateOptions{})
	require.NoError(t, err)

	d := &replicasetDiscovery{k8s: client}
	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 1)
	}, 5*time.Second, 100*time.Millisecond)
}

func Test_getDiscoveredReplicaSetsShouldNotIgnoreLabeledReplicaSetsIfExcludesDisabled(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DisableDiscoveryExcludes = true
	extconfig.Config.DiscoveryMaxPodCount = 50

	_, err := clientset.
		AppsV1().
		ReplicaSets("default").
		Create(context.Background(), testReplicaSet(nil), metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = clientset.
		AppsV1().
		ReplicaSets("default").
		Create(context.Background(), testReplicaSet(func(replicaset *appsv1.ReplicaSet) {
			replicaset.ObjectMeta.Name = "shop-ignore"
			replicaset.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
		}), metav1.CreateOptions{})
	require.NoError(t, err)

	d := &replicasetDiscovery{k8s: client}
	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 2)
	}, 5*time.Second, 100*time.Millisecond)
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewClientset()
	client := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	return client, clientset
}
