// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdeployment

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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_deploymentDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		configModifier            func(*extconfig.Specification)
		pods                      []*corev1.Pod
		nodes                     []*corev1.Node
		deployment                *appsv1.Deployment
		hpa                       *autoscalingv2.HorizontalPodAutoscaler
		service                   *corev1.Service
		expectedAttributesExactly map[string][]string
		expectedAttributes        map[string][]string
		expectedAttributesAbsence []string
	}{
		{
			name: "should discover basic attributes",
			pods: []*corev1.Pod{testPod("aaaaa", nil), testPod("bbbbb", func(pod *corev1.Pod) {
				pod.Spec.NodeName = "worker-2"
			})},
			nodes:      []*corev1.Node{testNode("worker-1"), testNode("worker-2")},
			deployment: testDeployment(nil),
			expectedAttributesExactly: map[string][]string{
				"host.hostname":                              {"worker-1", "worker-2"},
				"host.domainname":                            {"worker-1.internal", "worker-2.internal"},
				"k8s.namespace":                              {"default"},
				"k8s.deployment":                             {"shop"},
				"k8s.workload-type":                          {"deployment"},
				"k8s.workload-owner":                         {"shop"},
				"k8s.deployment.label.best-city":             {"Kevelaer"},
				"k8s.deployment.label":                       {"best-city"},
				"k8s.label.best-city":                        {"Kevelaer"},
				"k8s.label":                                  {"best-city"},
				"k8s.deployment.min-ready-seconds":           {"10"},
				"k8s.specification.replicas":                 {"3"},
				"k8s.cluster-name":                           {"development"},
				"k8s.pod.name":                               {"shop-pod-aaaaa", "shop-pod-bbbbb"},
				"k8s.container.id":                           {"crio://abcdef-aaaaa", "crio://abcdef-bbbbb"},
				"k8s.container.id.stripped":                  {"abcdef-aaaaa", "abcdef-bbbbb"},
				"k8s.distribution":                           {"kubernetes"},
				"k8s.specification.has-host-podantiaffinity": {"false"},
				"k8s.container.name":                         {"nginx", "shop"},
			},
		},
		{
			name:       "hostnames should be unique and not duplicated",
			nodes:      []*corev1.Node{testNode("worker-1")},
			pods:       []*corev1.Pod{testPod("aaaaa", nil), testPod("bbbbb", nil)},
			deployment: testDeployment(nil),
			expectedAttributes: map[string][]string{
				"host.hostname":   {"worker-1"},
				"host.domainname": {"worker-1.internal"},
			},
		},
		{
			name:       "should add service name",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.service.name": {"shop-kevelaer"},
			},
		},
		{
			name: "should detect host-podantiaffinity",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.ObjectMeta.Labels = map[string]string{
					"app": "foo",
				}
				deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
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
			name:       "should ignore empty container ids",
			pods:       []*corev1.Pod{testPod("aaaaa", func(pod *corev1.Pod) { pod.Status.ContainerStatuses[0].ContainerID = "" })},
			deployment: testDeployment(nil),
			expectedAttributesAbsence: []string{
				"k8s.container.id",
				"k8s.container.id.stripped",
			},
		},
		{
			name: "should not add probe summary if no service is defined",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe = nil
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
				deployment.Spec.Template.Spec.Containers[1].LivenessProbe = nil
				deployment.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			expectedAttributesAbsence: []string{
				"k8s.specification.probes.summary",
			},
		},
		{
			name: "should report probes ok",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/live",
							Port: intstr.FromInt32(80),
						},
					},
				}
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/ready",
							Port: intstr.FromInt32(80),
						},
					},
				}
				deployment.Spec.Template.Spec.Containers[1].LivenessProbe = nil
				deployment.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"OK"},
			},
		},
		{
			name: "should report equal probes",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/",
							Port: intstr.FromInt32(80),
						},
					},
				}
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/",
							Port: intstr.FromInt32(80),
						},
					},
				}
				deployment.Spec.Template.Spec.Containers[1].LivenessProbe = nil
				deployment.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"*Same readiness and liveness probe*\n\nMake sure to not use the same probes for readiness and liveness."},
			},
		},
		{
			name: "should report missing readiness probe",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = nil
				deployment.Spec.Template.Spec.Containers[1].ReadinessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"*Missing readinessProbe*\n\nWhen Kubernetes redeploys, it can't determine when the pod is ready to accept incoming requests. They may receive requests before being able to handle them properly."},
			},
		},
		{
			name: "should report missing liveness probe",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].LivenessProbe = nil
				deployment.Spec.Template.Spec.Containers[1].LivenessProbe = nil
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.probes.summary": {"*Missing livenessProbe*\n\nKubernetes cannot detect unresponsive pods/container and thus will never restart them automatically."},
			},
		},
		{
			name: "should report missing limits and requests",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
					Limits:   nil,
					Requests: nil,
				}
				deployment.Spec.Template.Spec.Containers[1].Resources = corev1.ResourceRequirements{
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
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Template.Spec.Containers[0].Image = "nginx"
				deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = "Never"
				deployment.Spec.Template.Spec.Containers[1].Image = "shop-container"
				deployment.Spec.Template.Spec.Containers[1].ImagePullPolicy = "Never"
			}),
			expectedAttributes: map[string][]string{
				"k8s.container.image.with-latest-tag":                  {"nginx", "shop"},
				"k8s.container.image.without-image-pull-policy-always": {"nginx", "shop"},
			},
		},
		{
			name: "should report wrong rollout strategy",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.has-rolling-update-strategy": {"false"},
			},
		},
		{
			name: "should not report wrong rollout strategy if no service is defined",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
			}),
			expectedAttributesAbsence: []string{"k8s.specification.has-rolling-update-strategy"},
		},
		{
			name:       "should report good rollout strategy",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.has-rolling-update-strategy": {"true"},
			},
		},
		{
			name: "should report deployment redundancy false",
			pods: []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(func(deployment *appsv1.Deployment) {
				deployment.Spec.Replicas = extutil.Ptr(int32(1))
			}),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.is-redundant": {"false"},
			},
		},
		{
			name:       "should report deployment redundancy true",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.is-redundant": {"true"},
			},
		},
		{
			name:       "should report deployment redundancy true and consider config",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			configModifier: func(specification *extconfig.Specification) {
				specification.AdviceSingleReplicaMinReplicas = 4
			},
			expectedAttributes: map[string][]string{
				"k8s.specification.is-redundant": {"false"},
			},
		},
		{
			name:       "should report hpa redundancy false",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			hpa: testHPA(func(hpa *autoscalingv2.HorizontalPodAutoscaler) {
				hpa.Spec.MinReplicas = extutil.Ptr(int32(1))
			}),
			expectedAttributes: map[string][]string{
				"k8s.specification.is-redundant": {"false"},
			},
		},
		{
			name:       "should report hpa redundancy true",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			hpa:        testHPA(nil),
			expectedAttributes: map[string][]string{
				"k8s.specification.is-redundant": {"true"},
			},
		},
		{
			name:       "should report hpa redundancy true and consider config",
			pods:       []*corev1.Pod{testPod("aaaaa", nil)},
			deployment: testDeployment(nil),
			service:    testService(nil),
			hpa:        testHPA(nil),
			configModifier: func(specification *extconfig.Specification) {
				specification.AdviceSingleReplicaMinReplicas = 4
			},
			expectedAttributes: map[string][]string{
				"k8s.specification.is-redundant": {"false"},
			},
		},
		{
			name:                      "should not report multiple replicas if no service is defined",
			pods:                      []*corev1.Pod{testPod("aaaaa", nil)},
			deployment:                testDeployment(nil),
			expectedAttributesAbsence: []string{"k8s.specification.is-redundant"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			extconfig.Config.ClusterName = "development"
			extconfig.Config.LabelFilter = []string{"secret-label"}
			extconfig.Config.DiscoveryMaxPodCount = 50
			extconfig.Config.AdviceSingleReplicaMinReplicas = 2
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
			objects = append(objects, tt.deployment)
			if tt.hpa != nil {
				objects = append(objects, tt.hpa)
			}
			if tt.service != nil {
				objects = append(objects, tt.service)
			}

			stopCh := make(chan struct{})
			defer close(stopCh)
			client := getTestClient(stopCh, objects...)

			d := &deploymentDiscovery{k8s: client}
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
			assert.Equal(t, DeploymentTargetType, target.TargetType)
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

func testHPA(modifier func(autoscaler *autoscalingv2.HorizontalPodAutoscaler)) *autoscalingv2.HorizontalPodAutoscaler {
	autoscaler := &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MinReplicas: extutil.Ptr(int32(2)),
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "shop",
				APIVersion: "apps/v1",
			},
		},
	}
	if modifier != nil {
		modifier(autoscaler)
	}

	return autoscaler
}

func testDeployment(modifier func(*appsv1.Deployment)) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
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
		Spec: appsv1.DeploymentSpec{
			Selector: extutil.Ptr(metav1.LabelSelector{
				MatchLabels: map[string]string{
					"best-city": "kevelaer",
				},
			}),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			MinReadySeconds: 10,
			Replicas:        extutil.Ptr(int32(3)),
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
							Image:           "nginx:v1.2.3",
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
		modifier(deployment)
	}
	return deployment
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

func Test_getDiscoveredDeploymentsShouldIgnoreLabeledDeployments(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client := getTestClient(stopCh, testDeployment(nil), testDeployment(func(deployment *appsv1.Deployment) {
		deployment.ObjectMeta.Name = "shop-ignore"
		deployment.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
	}))
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DiscoveryMaxPodCount = 50

	d := &deploymentDiscovery{k8s: client}
	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 1)
	}, 5*time.Second, 100*time.Millisecond)
}

func Test_getDiscoveredDeploymentsShouldNotIgnoreLabeledDeploymentsIfExcludesDisabled(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client := getTestClient(stopCh, testDeployment(nil), testDeployment(func(deployment *appsv1.Deployment) {
		deployment.ObjectMeta.Name = "shop-ignore"
		deployment.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
	}))
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DisableDiscoveryExcludes = true
	extconfig.Config.DiscoveryMaxPodCount = 50

	d := &deploymentDiscovery{k8s: client}
	// When
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverTargets(context.Background())
		assert.Len(c, ed, 2)
	}, 5*time.Second, 100*time.Millisecond)
}

func getTestClient(stopCh <-chan struct{}, objects ...runtime.Object) *client.Client {
	dynamicClient := testutil.NewFakeDynamicClient()
	return client.CreateClient(testclient.NewClientset(objects...), stopCh, "", client.MockAllPermitted(), dynamicClient)
}
