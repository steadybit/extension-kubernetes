// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdaemonset

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_daemonSetDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		pods                      []*corev1.Pod
		nodes                     []*corev1.Node
		daemonSet                 *appsv1.DaemonSet
		service                   *corev1.Service
		configModifier            func(*extconfig.Specification)
		expectedAttributesExactly map[string][]string
		expectedAttributes        map[string][]string
		expectedAttributesAbsence []string
	}{
		{
			name: "should discover basic attributes",
			pods: []*corev1.Pod{testPod("aaaaa", nil), testPod("bbbbb", func(pod *corev1.Pod) {
				pod.Spec.NodeName = "worker-2"
			})},
			nodes:     []*corev1.Node{testNode("worker-1"), testNode("worker-2")},
			daemonSet: testDaemonSet(nil),
			expectedAttributesExactly: map[string][]string{
				"host.hostname":                 {"worker-1", "worker-2"},
				"host.domainname":               {"worker-1.internal", "worker-2.internal"},
				"k8s.namespace":                 {"default"},
				"k8s.daemonset":                 {"shop"},
				"k8s.workload-type":             {"daemonset"},
				"k8s.workload-owner":            {"shop"},
				"k8s.label.best-city":           {"Kevelaer"},
				"k8s.label":                     {"best-city"},
				"k8s.daemonset.label.best-city": {"Kevelaer"},
				"k8s.daemonset.label":           {"best-city"},
				"k8s.cluster-name":              {"development"},
				"k8s.pod.name":                  {"shop-pod-aaaaa", "shop-pod-bbbbb"},
				"k8s.container.id":              {"crio://abcdef-aaaaa", "crio://abcdef-bbbbb"},
				"k8s.container.id.stripped":     {"abcdef-aaaaa", "abcdef-bbbbb"},
				"k8s.distribution":              {"kubernetes"},
			},
		},
		{
			name:      "hostnames should be unique and not duplicated",
			pods:      []*corev1.Pod{testPod("aaaaa", nil), testPod("bbbbb", nil)},
			nodes:     []*corev1.Node{testNode("worker-1")},
			daemonSet: testDaemonSet(nil),
			expectedAttributes: map[string][]string{
				"host.hostname": {"worker-1"},
			},
		},
		{
			name:      "should add service name",
			pods:      []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(nil),
			service:   testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.service.name": {"shop-kevelaer"},
			},
		},
		{
			name:      "should ignore empty container ids",
			pods:      []*corev1.Pod{testPod("aaaaa", func(pod *corev1.Pod) { pod.Status.ContainerStatuses[0].ContainerID = "" })},
			daemonSet: testDaemonSet(nil),
			expectedAttributesAbsence: []string{
				"k8s.container.id",
				"k8s.container.id.stripped",
			},
		},
		{
			name: "should not add probe summary if no service is defined",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe = nil
				daemonSet.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
				daemonSet.Spec.Template.Spec.Containers[1].LivenessProbe = nil
				daemonSet.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			expectedAttributesAbsence: []string{
				"k8s.specification.probes.summary",
			},
		},
		{
			name: "should report equal probes",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/",
							Port: intstr.FromInt32(80),
						},
					},
				}
				daemonSet.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/",
							Port: intstr.FromInt32(80),
						},
					},
				}
				daemonSet.Spec.Template.Spec.Containers[1].LivenessProbe = nil
				daemonSet.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"*Same readiness and liveness probe*\n\nMake sure to not use the same probes for readiness and liveness."},
			},
		},
		{
			name: "should report missing readiness probe",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
				daemonSet.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"*Missing readinessProbe*\n\nWhen Kubernetes redeploys, it can't determine when the pod is ready to accept incoming requests. They may receive requests before being able to handle them properly."},
			},
		},
		{
			name: "should report missing liveness probe",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.Containers[0].LivenessProbe = nil
				daemonSet.Spec.Template.Spec.Containers[1].LivenessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"*Missing livenessProbe*\n\nKubernetes cannot detect unresponsive pods/container and thus will never restart them automatically."},
			},
		},
		{
			name: "should report missing limits and requests",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
					Limits:   nil,
					Requests: nil,
				}
				daemonSet.Spec.Template.Spec.Containers[1].Resources = corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
						corev1.ResourceMemory:           *resource.NewQuantity(500, resource.DecimalSI),
						corev1.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
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
				"k8s.container.spec.request.ephemeral-storage.not-set": {"nginx", "shop"},
			},
		},
		{
			name: "should report image pull policy and image tag",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			daemonSet: testDaemonSet(func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.Containers[0].Image = "nginx"
				daemonSet.Spec.Template.Spec.Containers[0].ImagePullPolicy = "Never"
				daemonSet.Spec.Template.Spec.Containers[1].Image = "shop-container"
				daemonSet.Spec.Template.Spec.Containers[1].ImagePullPolicy = "Never"
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
			extconfig.Config.ClusterName = "development"
			extconfig.Config.LabelFilter = []string{"secret-label"}
			extconfig.Config.DiscoveryMaxPodCount = 50
			if tt.configModifier != nil {
				tt.configModifier(&extconfig.Config)
			}

			var objects []runtime.Object
			for _, pod := range tt.pods {
				objects = append(objects, pod)
			}
			for _, node := range tt.nodes {
				objects = append(objects, node)
			}
			objects = append(objects, tt.daemonSet)
			if tt.service != nil {
				objects = append(objects, tt.service)
			}

			stopCh := make(chan struct{})
			defer close(stopCh)
			client := getTestClient(stopCh, objects...)

			d := &daemonSetDiscovery{k8s: client}
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
			assert.Equal(t, DaemonSetTargetType, target.TargetType)
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
					assert.Equal(t, v, attributeValues)
				}
			}
			if len(tt.expectedAttributesAbsence) > 0 {
				for _, k := range tt.expectedAttributesAbsence {
					assert.NotContains(t, target.Attributes, k, "attribute %s", k)
				}
			}
		})
	}
}

func testNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalDNS,
					Address: fmt.Sprintf("%s.internal", name),
				},
			},
		},
	}
}

func testPod(nameSuffix string, modifier func(*corev1.Pod)) *corev1.Pod {
	pod := &corev1.Pod{
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
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					ContainerID: fmt.Sprintf("crio://abcdef-%s", nameSuffix),
					Name:        "MrFancyPants",
					Image:       "nginx",
				},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "worker-1",
		},
	}
	if modifier != nil {
		modifier(pod)
	}
	return pod
}

func testDaemonSet(modifier func(*appsv1.DaemonSet)) *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
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
		Spec: appsv1.DaemonSetSpec{
			Selector: extutil.Ptr(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"best-city": "kevelaer",
				},
			}),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"best-city": "Kevelaer",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "nginx",
							Image:           "nginx:corev1.2.3",
							ImagePullPolicy: "Always",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									corev1.ResourceMemory:           *resource.NewQuantity(500, resource.DecimalSI),
									corev1.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									corev1.ResourceMemory:           *resource.NewQuantity(250, resource.DecimalSI),
									corev1.ResourceEphemeralStorage: *resource.NewQuantity(500, resource.DecimalSI),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{},
							},
						},
						{
							Name:            "shop",
							Image:           "shop-container:v5",
							ImagePullPolicy: "Always",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									corev1.ResourceMemory:           *resource.NewQuantity(500, resource.DecimalSI),
									corev1.ResourceEphemeralStorage: *resource.NewQuantity(1000, resource.DecimalSI),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:              *resource.NewQuantity(1, resource.BinarySI),
									corev1.ResourceMemory:           *resource.NewQuantity(250, resource.DecimalSI),
									corev1.ResourceEphemeralStorage: *resource.NewQuantity(500, resource.DecimalSI),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{},
							},
						},
					},
				},
			},
		},
	}
	if modifier != nil {
		modifier(ds)
	}
	return ds
}

func testService(modifier func(service *corev1.Service)) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-kevelaer",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
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

func getTestClient(stopCh <-chan struct{}, objects ...runtime.Object) *client.Client {
	dynamicClient := testutil.NewFakeDynamicClient()
	return client.CreateClient(testclient.NewClientset(objects...), stopCh, "", client.MockAllPermitted(), dynamicClient)
}
