// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extrollout

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_rolloutDiscovery_Describe(t *testing.T) {
	// Setup test client
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, _, _ := getTestClient(stopCh)

	discovery := &rolloutDiscovery{k8s: k8sClient}

	description := discovery.Describe()

	assert.Equal(t, RolloutTargetType, description.Id)
	assert.NotNil(t, description.Discover)

	// Check that CallInterval is set to 30s
	callInterval := description.Discover.CallInterval
	assert.Equal(t, "30s", *callInterval)
}

func Test_rolloutDiscovery_DescribeTarget(t *testing.T) {
	k8sClient := &client.Client{}
	discovery := &rolloutDiscovery{k8s: k8sClient}

	targetDescription := discovery.DescribeTarget()

	assert.Equal(t, RolloutTargetType, targetDescription.Id)
	assert.Equal(t, "Kubernetes Argo Rollout", targetDescription.Label.One)
	assert.Equal(t, "Kubernetes Argo Rollouts", targetDescription.Label.Other)
	assert.Equal(t, "Kubernetes", *targetDescription.Category)
	assert.NotEmpty(t, targetDescription.Version)
	assert.NotNil(t, targetDescription.Icon)

	assert.Len(t, targetDescription.Table.Columns, 3)
	assert.Equal(t, "k8s.argo-rollout", targetDescription.Table.Columns[0].Attribute)
	assert.Equal(t, "k8s.namespace", targetDescription.Table.Columns[1].Attribute)
	assert.Equal(t, "k8s.cluster-name", targetDescription.Table.Columns[2].Attribute)

	assert.Len(t, targetDescription.Table.OrderBy, 1)
	assert.Equal(t, "k8s.argo-rollout", targetDescription.Table.OrderBy[0].Attribute)
	assert.Equal(t, discovery_kit_api.OrderByDirection("ASC"), targetDescription.Table.OrderBy[0].Direction)
}

func Test_rolloutDiscovery_DescribeEnrichmentRules(t *testing.T) {
	k8sClient := &client.Client{}
	discovery := &rolloutDiscovery{k8s: k8sClient}

	enrichmentRules := discovery.DescribeEnrichmentRules()

	require.Len(t, enrichmentRules, 1)

	rule := enrichmentRules[0]
	assert.Equal(t, "com.steadybit.extension_kubernetes.kubernetes-argo-rollout-to-container", rule.Id)
	assert.NotEmpty(t, rule.Version)

	assert.Equal(t, RolloutTargetType, rule.Src.Type)
	assert.Equal(t, "${dest.container.id.stripped}", rule.Src.Selector["k8s.container.id.stripped"])

	assert.Equal(t, "com.steadybit.extension_container.container", rule.Dest.Type)
	assert.Equal(t, "${src.k8s.container.id.stripped}", rule.Dest.Selector["container.id.stripped"])

	assert.Len(t, rule.Attributes, 2)
	assert.Equal(t, "k8s.argo-rollout.label.", rule.Attributes[0].Name)
	assert.Equal(t, discovery_kit_api.StartsWith, rule.Attributes[0].Matcher)
	assert.Equal(t, "^k8s\\.label\\.(?!topology).*", rule.Attributes[1].Name)
	assert.Equal(t, discovery_kit_api.Regex, rule.Attributes[1].Matcher)
}

func Test_rolloutDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		configModifier            func(*extconfig.Specification)
		pods                      []*v1.Pod
		nodes                     []*v1.Node
		rollout                   *unstructured.Unstructured
		service                   *v1.Service
		expectedAttributesExactly map[string][]string
		expectedAttributes        map[string][]string
	}{
		{
			name: "should discover basic attributes",
			pods: []*v1.Pod{testPod("aaaaa", nil), testPod("bbbbb", func(pod *v1.Pod) {
				pod.Spec.NodeName = "worker-2"
			})},
			nodes:   []*v1.Node{testNode("worker-1"), testNode("worker-2")},
			rollout: testRollout(nil),
			expectedAttributesExactly: map[string][]string{
				"host.hostname":                       {"worker-1", "worker-2"},
				"host.domainname":                     {"worker-1.internal", "worker-2.internal"},
				"k8s.namespace":                       {"default"},
				"k8s.argo-rollout":                    {"shop"},
				"k8s.workload-type":                   {"argo-rollout"},
				"k8s.workload-owner":                  {"shop"},
				"k8s.argo-rollout.label.best-city":    {"Kevelaer"},
				"k8s.label.best-city":                 {"Kevelaer"},
				"k8s.specification.min-ready-seconds": {"10"},
				"k8s.specification.replicas":          {"3"},
				"k8s.cluster-name":                    {"development"},
				"k8s.pod.name":                        {"shop-pod-aaaaa", "shop-pod-bbbbb"},
				"k8s.container.id":                    {"crio://abcdef-aaaaa-nginx", "crio://abcdef-aaaaa-shop", "crio://abcdef-bbbbb-nginx", "crio://abcdef-bbbbb-shop"},
				"k8s.container.id.stripped":           {"abcdef-aaaaa-nginx", "abcdef-aaaaa-shop", "abcdef-bbbbb-nginx", "abcdef-bbbbb-shop"},
				"k8s.distribution":                    {"kubernetes"},
			},
		},
		{
			name:    "hostnames should be unique and not duplicated",
			nodes:   []*v1.Node{testNode("worker-1")},
			pods:    []*v1.Pod{testPod("aaaaa", nil), testPod("bbbbb", nil)},
			rollout: testRollout(nil),
			expectedAttributes: map[string][]string{
				"host.hostname":   {"worker-1"},
				"host.domainname": {"worker-1.internal"},
			},
		},
		{
			name:    "should add service name",
			pods:    []*v1.Pod{testPod("aaaaa", nil)},
			rollout: testRollout(nil),
			service: testService(nil),
			expectedAttributes: map[string][]string{
				"k8s.service.name": {"shop-kevelaer"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			extconfig.Config.ClusterName = "development"
			extconfig.Config.LabelFilter = []string{"topology.kubernetes.io/zone"}
			extconfig.Config.DiscoveryDisabledArgoRollout = false // Enable Argo Rollout discovery
			extconfig.Config.DiscoveryMaxPodCount = 50            // Set pod count limit
			if tt.configModifier != nil {
				tt.configModifier(&extconfig.Config)
			}

			// Setup test client
			stopCh := make(chan struct{})
			defer close(stopCh)
			k8sClient, clientset, dynamicClient := getTestClient(stopCh)

			// Setup test data
			if tt.rollout != nil {
				// Add rollout to the dynamic client that's used by the k8s client
				_, err := dynamicClient.Resource(schema.GroupVersionResource{
					Group:    "argoproj.io",
					Version:  "v1alpha1",
					Resource: "rollouts",
				}).Namespace(tt.rollout.GetNamespace()).Create(context.Background(), tt.rollout, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			if tt.pods != nil {
				// Add pods to client
				for _, pod := range tt.pods {
					_, err := clientset.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
					require.NoError(t, err)
				}
			}

			if tt.nodes != nil {
				// Add nodes to client
				for _, node := range tt.nodes {
					_, err := clientset.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
					require.NoError(t, err)
				}
			}

			if tt.service != nil {
				// Add service to client
				_, err := clientset.CoreV1().Services(tt.service.Namespace).Create(context.Background(), tt.service, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			// Create discovery
			discovery := &rolloutDiscovery{k8s: k8sClient}

			// Discover targets and wait for informer to sync
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				targets, err := discovery.DiscoverTargets(context.Background())
				require.NoError(t, err)

				// Assertions
				if len(tt.expectedAttributesExactly) > 0 {
					require.Len(t, targets, 1)
					target := targets[0]

					for _, v := range target.Attributes {
						sort.Strings(v)
					}
					assert.Equal(t, tt.expectedAttributesExactly, target.Attributes)
				}

				if len(tt.expectedAttributes) > 0 {
					require.Len(t, targets, 1)
					target := targets[0]

					for k, v := range tt.expectedAttributes {
						attributeValues := target.Attributes[k]
						sort.Strings(attributeValues)
						assert.Equal(t, v, attributeValues, "Attribute %s should match", k)
					}
				}
			}, 1*time.Second, 100*time.Millisecond)
		})
	}
}

func Test_getDiscoveredRolloutsShouldIgnoreLabeledRollouts(t *testing.T) {
	// Setup config
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DisableDiscoveryExcludes = false
	extconfig.Config.DiscoveryDisabledArgoRollout = false // Enable Argo Rollout discovery
	extconfig.Config.DiscoveryMaxPodCount = 50            // Set pod count limit

	// Setup test client
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, _, dynamicClient := getTestClient(stopCh)

	// Create rollout with exclusion label
	rollout := testRollout(func(rollout *unstructured.Unstructured) {
		rollout.SetLabels(map[string]string{
			"steadybit.com/discovery-disabled": "true",
		})
	})

	// Add rollout to dynamic client
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}).Namespace(rollout.GetNamespace()).Create(context.Background(), rollout, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create discovery
	discovery := &rolloutDiscovery{k8s: k8sClient}

	// Discover targets and wait for informer to sync
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		targets, err := discovery.DiscoverTargets(context.Background())
		require.NoError(t, err)
		// Should be empty because rollout is excluded
		assert.Len(c, targets, 0)
	}, 1*time.Second, 100*time.Millisecond)
}

func Test_getDiscoveredRolloutsShouldNotIgnoreLabeledRolloutsIfExcludesDisabled(t *testing.T) {
	// Setup config
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DisableDiscoveryExcludes = true
	extconfig.Config.DiscoveryDisabledArgoRollout = false // Enable Argo Rollout discovery
	extconfig.Config.DiscoveryMaxPodCount = 50            // Set pod count limit

	// Setup test client
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, _, dynamicClient := getTestClient(stopCh)

	// Create rollout with exclusion label
	rollout := testRollout(func(rollout *unstructured.Unstructured) {
		rollout.SetLabels(map[string]string{
			"steadybit.com/discovery-disabled": "true",
		})
	})

	// Add rollout to dynamic client
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}).Namespace(rollout.GetNamespace()).Create(context.Background(), rollout, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create discovery
	discovery := &rolloutDiscovery{k8s: k8sClient}

	// Discover targets and wait for informer to sync
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		targets, err := discovery.DiscoverTargets(context.Background())
		require.NoError(t, err)
		// Should not be empty because excludes are disabled
		assert.Len(c, targets, 1)
	}, 1*time.Second, 100*time.Millisecond)
}

func Test_rolloutDiscovery_Simple(t *testing.T) {
	// Setup config
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DiscoveryDisabledArgoRollout = false // Enable Argo Rollout discovery
	extconfig.Config.DiscoveryMaxPodCount = 50            // Set pod count limit

	// Setup test client
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sClient, _, dynamicClient := getTestClient(stopCh)

	// Create a simple rollout
	rollout := testRollout(nil)

	// Add rollout to dynamic client
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "rollouts",
	}).Namespace(rollout.GetNamespace()).Create(context.Background(), rollout, metav1.CreateOptions{})
	require.NoError(t, err)

	// Create discovery
	discovery := &rolloutDiscovery{k8s: k8sClient}

	// Discover targets and wait for informer to sync
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		targets, err := discovery.DiscoverTargets(context.Background())
		require.NoError(t, err)
		// Should discover at least one rollout
		assert.GreaterOrEqual(c, len(targets), 1)
		if len(targets) > 0 {
			target := targets[0]
			assert.Equal(c, "development/default/shop", target.Id)
			assert.Equal(c, "shop", target.Label)
			assert.Equal(c, RolloutTargetType, target.TargetType)
		}
	}, 1*time.Second, 100*time.Millisecond)
}

// Helper functions

func testNode(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubernetes.io/hostname": name,
			},
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeHostName,
					Address: name,
				},
				{
					Type:    v1.NodeInternalDNS,
					Address: fmt.Sprintf("%s.internal", name),
				},
				{
					Type:    v1.NodeInternalIP,
					Address: "192.168.1.1",
				},
			},
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func testRollout(modifier func(rollout *unstructured.Unstructured)) *unstructured.Unstructured {
	rollout := &unstructured.Unstructured{}
	rollout.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Rollout",
	})
	rollout.SetName("shop")
	rollout.SetNamespace("default")
	rollout.SetLabels(map[string]string{
		"best-city": "Kevelaer",
	})

	// Set basic spec
	spec := map[string]interface{}{
		"replicas":        int64(3),
		"minReadySeconds": int64(10),
		"selector": map[string]interface{}{
			"matchLabels": map[string]interface{}{
				"app": "shop",
			},
		},
		"template": map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"app": "shop",
				},
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "nginx",
						"image": "nginx:1.19",
						"ports": []interface{}{
							map[string]interface{}{
								"containerPort": int64(80),
							},
						},
						"livenessProbe": map[string]interface{}{
							"httpGet": map[string]interface{}{
								"path": "/live",
								"port": int64(80),
							},
						},
						"readinessProbe": map[string]interface{}{
							"httpGet": map[string]interface{}{
								"path": "/ready",
								"port": int64(80),
							},
						},
					},
					map[string]interface{}{
						"name":  "shop",
						"image": "shop:1.0",
						"ports": []interface{}{
							map[string]interface{}{
								"containerPort": int64(8080),
							},
						},
					},
				},
			},
		},
	}

	if err := unstructured.SetNestedMap(rollout.Object, spec, "spec"); err != nil {
		panic(fmt.Sprintf("failed to set nested map: %v", err))
	}

	if modifier != nil {
		modifier(rollout)
	}
	return rollout
}

func testPod(nameSuffix string, modifier func(*v1.Pod)) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("shop-pod-%s", nameSuffix),
			Namespace: "default",
			Labels: map[string]string{
				"app": "shop",
			},
		},
		Spec: v1.PodSpec{
			NodeName: "worker-1",
			Containers: []v1.Container{
				{
					Name:  "nginx",
					Image: "nginx:1.19",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
					LivenessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/live",
								Port: intstr.FromInt32(80),
							},
						},
					},
					ReadinessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/ready",
								Port: intstr.FromInt32(80),
							},
						},
					},
				},
				{
					Name:  "shop",
					Image: "shop:1.0",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 8080,
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:        "nginx",
					ContainerID: fmt.Sprintf("crio://abcdef-%s-nginx", nameSuffix),
					Ready:       true,
				},
				{
					Name:        "shop",
					ContainerID: fmt.Sprintf("crio://abcdef-%s-shop", nameSuffix),
					Ready:       true,
				},
			},
		},
	}
	if modifier != nil {
		modifier(pod)
	}
	return pod
}

func testService(modifier func(service *v1.Service)) *v1.Service {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-kevelaer",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "shop",
			},
			Ports: []v1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt32(80),
				},
			},
		},
	}
	if modifier != nil {
		modifier(service)
	}
	return service
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface, dynamic.Interface) {
	clientset := testclient.NewSimpleClientset()
	dynamicClient := testutil.NewFakeDynamicClient()

	k8sClient := client.CreateClient(clientset, stopCh, "", client.MockAllPermitted(), dynamicClient)
	k8sClient.Distribution = "kubernetes"

	return k8sClient, clientset, dynamicClient
}
