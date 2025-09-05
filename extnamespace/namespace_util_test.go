// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extnamespace

import (
	"sort"
	"testing"
	"time"

	kclient "github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
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
				"k8s.label.best-city":           {"Kevelaer"},
				"k8s.namespace.label.best-city": {"Kevelaer"},
			},
			expectedAttributesAbsence: []string{"k8s.label.secret-label", "k8s.namespace.label.secret-label"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			stopCh := make(chan struct{})
			defer close(stopCh)
			client := getTestClient(stopCh, tt.namespace)
			extconfig.Config.ClusterName = "development"
			extconfig.Config.LabelFilter = []string{"secret-label"}

			// When
			var attributes map[string][]string
			assert.EventuallyWithT(t, func(c *assert.CollectT) {
				attributes = AddNamespaceLabels(client, tt.namespace.Name, map[string][]string{})
				require.Len(t, attributes, 2)
			}, 5*time.Second, 100*time.Millisecond)

			// Then
			require.Len(t, attributes, 2)
			if len(tt.expectedAttributesExactly) > 0 {
				for _, v := range attributes {
					sort.Strings(v)
				}
				assert.Equal(t, tt.expectedAttributesExactly, attributes)
			}
			if len(tt.expectedAttributes) > 0 {
				for k, v := range tt.expectedAttributes {
					attributeValues := attributes[k]
					sort.Strings(attributeValues)
					assert.Equal(t, v, attributeValues)
				}
			}
			if len(tt.expectedAttributesAbsence) > 0 {
				for _, k := range tt.expectedAttributesAbsence {
					assert.NotContains(t, attributes, k)
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

func getTestClient(stopCh <-chan struct{}, objects ...runtime.Object) *kclient.Client {
	return kclient.CreateClient(testclient.NewClientset(objects...), stopCh, "/oapi", kclient.MockAllPermitted())
}
