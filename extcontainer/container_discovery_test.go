// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

import (
	"context"
	kclient "github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"sort"
	"testing"
	"time"
)

func Test_containerDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		pod                       *v1.Pod
		services                  []*v1.Service
		expectedAttributesExactly map[string][]string
		expectedAttributes        map[string][]string
		expectedAttributesAbsence []string
	}{
		{
			name: "should discover basic attributes",
			pod:  testPod(nil),
			expectedAttributesExactly: map[string][]string{
				"k8s.cluster-name":                        {"development"},
				"k8s.container.id":                        {"crio://abcdef"},
				"k8s.container.id.stripped":               {"abcdef"},
				"k8s.container.name":                      {"MrFancyPants"},
				"k8s.container.image":                     {"nginx"},
				"k8s.container.image.pull-policy":         {"Always"},
				"k8s.container.limit.cpu":                 {"1000"},
				"k8s.container.limit.memory":              {"2000"},
				"k8s.container.probes.liveness.existent":  {"true"},
				"k8s.container.probes.readiness.existent": {"true"},
				"k8s.namespace":                           {"default"},
				"k8s.node.name":                           {"worker-1"},
				"k8s.pod.name":                            {"shop"},
				"k8s.pod.label.best-city":                 {"Kevelaer"},
				"k8s.label.best-city":                     {"Kevelaer"},
				"k8s.distribution":                        {"openshift"},
			},
		},
		{
			name: "should add service names",
			pod:  testPod(nil),
			services: []*v1.Service{
				testService(nil),
				testService(func(service *v1.Service) {
					service.ObjectMeta.Name = "shop-kevelaer-v2"
				}),
				testService(func(service *v1.Service) {
					service.ObjectMeta.Name = "shop-solingen"
					service.Spec.Selector["best-city"] = "Solingen"
				}),
			},
			expectedAttributes: map[string][]string{
				"k8s.service.name": {"shop-kevelaer", "shop-kevelaer-v2"},
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

			_, err := clientset.CoreV1().
				Pods("default").
				Create(context.Background(), tt.pod, metav1.CreateOptions{})
			require.NoError(t, err)

			for _, service := range tt.services {
				_, err := clientset.CoreV1().
					Services("default").
					Create(context.Background(), service, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			d := &containerDiscovery{k8s: client}
			// When
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				ed, _ := d.DiscoverEnrichmentData(context.Background())
				assert.Len(c, ed, 1)
			}, 1*time.Second, 100*time.Millisecond)

			// Then
			targets, _ := d.DiscoverEnrichmentData(context.Background())
			require.Len(t, targets, 1)
			target := targets[0]
			assert.Equal(t, "crio://abcdef", target.Id)
			assert.Equal(t, KubernetesContainerEnrichmentDataType, target.EnrichmentDataType)
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
					assert.NotContains(t, target.Attributes, k)
				}
			}
		})
	}
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

func testPod(modifier func(pod *v1.Pod)) *v1.Pod {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop",
			Namespace: "default",
			Labels: map[string]string{
				"best-city":    "Kevelaer",
				"secret-label": "secret-value",
			},
		},
		Status: v1.PodStatus{
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
			Containers: []v1.Container{
				{
					Name:            "MrFancyPants",
					Image:           "nginx",
					ImagePullPolicy: "Always",
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"cpu":    resource.MustParse("1"),
							"memory": resource.MustParse("2"),
						},
					},
					LivenessProbe: &v1.Probe{
						PeriodSeconds: 5,
					},
					ReadinessProbe: &v1.Probe{
						PeriodSeconds: 5,
					},
				},
			},
		},
	}
	if modifier != nil {
		modifier(pod)
	}
	return pod
}

func Test_getDiscoveredContainerShouldIgnoreLabeledPods(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"

	_, err := clientset.CoreV1().
		Pods("default").
		Create(context.Background(), testPod(nil), metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().
		Pods("default").
		Create(context.Background(), testPod(func(pod *v1.Pod) {
			pod.ObjectMeta.Name = "shop-ignored"
			pod.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
		}), metav1.CreateOptions{})
	require.NoError(t, err)

	d := &containerDiscovery{k8s: client}

	// Then
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverEnrichmentData(context.Background())
		assert.Len(c, ed, 1)
	}, 1*time.Second, 100*time.Millisecond)
}

func Test_getDiscoveredContainerShouldNotIgnoreLabeledPodsIfExcludesDisabled(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DisableDiscoveryExcludes = true

	_, err := clientset.CoreV1().
		Pods("default").
		Create(context.Background(), testPod(nil), metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().
		Pods("default").
		Create(context.Background(), testPod(func(pod *v1.Pod) {
			pod.ObjectMeta.Name = "shop-ignored"
			pod.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
		}), metav1.CreateOptions{})

	d := &containerDiscovery{k8s: client}

	// Then
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverEnrichmentData(context.Background())
		assert.Len(c, ed, 2)
	}, 1*time.Second, 100*time.Millisecond)
}

func getTestClient(stopCh <-chan struct{}) (*kclient.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	client := kclient.CreateClient(clientset, stopCh, "/oapi", kclient.MockAllPermitted())
	return client, clientset
}
