// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extstatefulset

import (
	"context"
	"fmt"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"sort"
	"testing"
	"time"
)

func Test_statefulSetDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		pods                      []*v1.Pod
		statefulSet               *appsv1.StatefulSet
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
			statefulSet: testStatefulSet(nil),
			expectedAttributesExactly: map[string][]string{
				"host.hostname":                              {"worker-1", "worker-2"},
				"k8s.namespace":                              {"default"},
				"k8s.statefulset":                            {"shop"},
				"k8s.label.best-city":                        {"Kevelaer"},
				"k8s.deployment.hpa.existent":                {"false"},
				"k8s.cluster-name":                           {"development"},
				"k8s.pod.name":                               {"shop-pod-aaaaa", "shop-pod-bbbbb"},
				"k8s.container.id":                           {"crio://abcdef-aaaaa", "crio://abcdef-bbbbb"},
				"k8s.container.id.stripped":                  {"abcdef-aaaaa", "abcdef-bbbbb"},
				"k8s.distribution":                           {"kubernetes"},
				"k8s.specification.has-host-podantiaffinity": {"false"},
			},
		},
		{
			name:        "hostnames should be unique and not duplicated",
			pods:        []*v1.Pod{testPod("aaaaa", nil), testPod("bbbbb", nil)},
			statefulSet: testStatefulSet(nil),
			expectedAttributes: map[string][]string{
				"host.hostname": {"worker-1"},
			},
		},
		{
			name:        "should add service name",
			pods:        []*v1.Pod{testPod("aaaaa", nil)},
			statefulSet: testStatefulSet(nil),
			service:     testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.service.name": {"shop-kevelaer"},
			},
		},
		{
			name: "should detect host-podantiaffinity",
			pods: []*v1.Pod{testPod("aaaaa", nil)},
			statefulSet: testStatefulSet(func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.Template.ObjectMeta.Labels = map[string]string{
					"app": "foo",
				}
				statefulSet.Spec.Template.Spec.Affinity = &v1.Affinity{
					PodAntiAffinity: &v1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
							{
								TopologyKey: "kubernetes.io/hostname",
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"app": "foo",
									},
								},
							},
						},
					},
				}
			}),
			expectedAttributes: map[string][]string{
				"k8s.specification.has-host-podantiaffinity": {"true"},
			},
		},
		{
			name:        "should ignore empty container ids",
			pods:        []*v1.Pod{testPod("aaaaa", func(pod *v1.Pod) { pod.Status.ContainerStatuses[0].ContainerID = "" })},
			statefulSet: testStatefulSet(nil),
			expectedAttributesAbsence: []string{
				"k8s.container.id",
				"k8s.container.id.stripped",
			},
		},
		{
			name: "should report missing probes",
			pods: []*v1.Pod{testPod("aaaaa", nil)},
			statefulSet: testStatefulSet(func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.Template.Spec.Containers[0].LivenessProbe = nil
				statefulSet.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
				statefulSet.Spec.Template.Spec.Containers[1].LivenessProbe = nil
				statefulSet.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			expectedAttributes: map[string][]string{
				"k8s.container.probes.liveness.not-set":  {"nginx", "shop"},
				"k8s.container.probes.readiness.not-set": {"nginx", "shop"},
			},
		},
		{
			name: "should report missing limits and requests",
			pods: []*v1.Pod{testPod("aaaaa", nil)},
			statefulSet: testStatefulSet(func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.Template.Spec.Containers[0].Resources = v1.ResourceRequirements{
					Limits:   nil,
					Requests: nil,
				}
				statefulSet.Spec.Template.Spec.Containers[1].Resources = v1.ResourceRequirements{
					Limits: v1.ResourceList{
						v1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
						v1.ResourceMemory:           *resource.NewQuantity(500, resource.DecimalSI),
						v1.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
					},
					Requests: nil,
				}
			}),
			expectedAttributes: map[string][]string{
				"k8s.container.spec.limit.cpu.not-set":                 {"nginx"},
				"k8s.container.spec.limit.memory.not-set":              {"nginx"},
				"k8s.container.spec.limit.ephemeral-storage.not-set":   {"nginx"},
				"k8s.container.spec.request.cpu.not-set":               {"nginx", "shop"},
				"k8s.container.spec.request.memory.not-set":            {"nginx", "shop"},
				"k8s.container.spec.request.ephemeral-storage.not-set": {"shop"}, //should also report nginx, see PR https://github.com/zegl/kube-score/pull/573
			},
		},
		{
			name: "should report image pull policy and image tag",
			pods: []*v1.Pod{testPod("aaaaa", nil)},
			statefulSet: testStatefulSet(func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.Template.Spec.Containers[0].Image = "nginx"
				statefulSet.Spec.Template.Spec.Containers[0].ImagePullPolicy = "Never"
				statefulSet.Spec.Template.Spec.Containers[1].Image = "shop-container"
				statefulSet.Spec.Template.Spec.Containers[1].ImagePullPolicy = "Never"
			}),
			expectedAttributes: map[string][]string{
				"k8s.container.image.with-latest-tag":                  {"nginx", "shop"},
				"k8s.container.image.without-image-pull-policy-always": {"nginx", "shop"},
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

			for _, pod := range tt.pods {
				_, err := clientset.CoreV1().
					Pods("default").
					Create(context.Background(), pod, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			_, err := clientset.
				AppsV1().
				StatefulSets("default").
				Create(context.Background(), tt.statefulSet, metav1.CreateOptions{})
			require.NoError(t, err)

			if tt.service != nil {
				_, err := clientset.CoreV1().
					Services("default").
					Create(context.Background(), tt.service, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			d := &statefulSetDiscovery{k8s: client}
			// When
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				ed, _ := d.DiscoverTargets(context.Background())
				assert.Len(c, ed, 1)
			}, 1*time.Second, 100*time.Millisecond)

			// Then
			targets, _ := d.DiscoverTargets(context.Background())
			require.Len(t, targets, 1)
			target := targets[0]
			assert.Equal(t, "development/default/shop", target.Id)
			assert.Equal(t, "shop", target.Label)
			assert.Equal(t, StatefulSetTargetType, target.TargetType)
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

func testStatefulSet(modifier func(set *appsv1.StatefulSet)) *appsv1.StatefulSet {
	sfs := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
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
		modifier(sfs)
	}
	return sfs
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

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	client := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted())
	return client, clientset
}
