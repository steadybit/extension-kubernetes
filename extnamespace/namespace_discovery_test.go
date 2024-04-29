// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extnamespace

import (
	"context"
	kclient "github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"sort"
	"testing"
	"time"
)

func Test_namespaceDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		namespace                 *v1.Namespace
		services                  []*v1.Service
		expectedAttributesExactly map[string][]string
		expectedAttributes        map[string][]string
		expectedAttributesAbsence []string
	}{
		{
			name:      "should discover basic attributes",
			namespace: testNamespace(nil),
			expectedAttributesExactly: map[string][]string{
				"k8s.cluster-name":              {"development"},
				"k8s.namespace.id":              {"1234"},
				"k8s.namespace":                 {"default"},
				"k8s.label.best-city":           {"Kevelaer"},
				"k8s.namespace.label.best-city": {"Kevelaer"},
				"k8s.distribution":              {"openshift"},
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

			_, err := clientset.CoreV1().
				Namespaces().
				Create(context.Background(), tt.namespace, metav1.CreateOptions{})
			require.NoError(t, err)

			d := &namespaceDiscovery{k8s: client}
			// When
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				ed, _ := d.DiscoverEnrichmentData(context.Background())
				assert.Len(c, ed, 1)
			}, 1*time.Second, 100*time.Millisecond)

			// Then
			targets, _ := d.DiscoverEnrichmentData(context.Background())
			require.Len(t, targets, 1)
			target := targets[0]
			assert.Equal(t, "1234", target.Id)
			assert.Equal(t, KubernetesNamespaceEnrichmentDataType, target.EnrichmentDataType)
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

func testNamespace(modifier func(namespace *v1.Namespace)) *v1.Namespace {
	namespace := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  "1234",
			Name: "default",
			Labels: map[string]string{
				"best-city":    "Kevelaer",
				"secret-label": "secret-value",
			},
		},
	}
	if modifier != nil {
		modifier(namespace)
	}
	return namespace
}

func Test_getDiscoveredNamespaceShouldIgnoreLabeledNamespaces(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"

	_, err := clientset.CoreV1().
		Namespaces().
		Create(context.Background(), testNamespace(nil), metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().
		Namespaces().
		Create(context.Background(), testNamespace(func(namespace *v1.Namespace) {
			namespace.ObjectMeta.Name = "shop-ignored"
			namespace.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
		}), metav1.CreateOptions{})
	require.NoError(t, err)

	d := &namespaceDiscovery{k8s: client}

	// Then
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		ed, _ := d.DiscoverEnrichmentData(context.Background())
		assert.Len(c, ed, 1)
	}, 1*time.Second, 100*time.Millisecond)
}

func Test_getDiscoveredNamespaceShouldNotIgnoreLabeledNamespacesIfExcludesDisabled(t *testing.T) {
	// Given
	stopCh := make(chan struct{})
	defer close(stopCh)
	client, clientset := getTestClient(stopCh)
	extconfig.Config.ClusterName = "development"
	extconfig.Config.DisableDiscoveryExcludes = true

	_, err := clientset.CoreV1().
		Namespaces().
		Create(context.Background(), testNamespace(nil), metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = clientset.CoreV1().
		Namespaces().
		Create(context.Background(), testNamespace(func(namespace *v1.Namespace) {
			namespace.ObjectMeta.Name = "shop-ignored"
			namespace.ObjectMeta.Labels["steadybit.com/discovery-disabled"] = "true"
		}), metav1.CreateOptions{})
	require.NoError(t, err)

	d := &namespaceDiscovery{k8s: client}

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
