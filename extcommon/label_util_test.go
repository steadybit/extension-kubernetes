// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extcommon

import (
	"sort"
	"strings"
	"testing"
	"time"

	kclient "github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/steadybit/extension-kubernetes/v2/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func Test_namespaceDiscovery(t *testing.T) {
	tests := []struct {
		name                      string
		namespace                 *corev1.Namespace
		services                  []*corev1.Service
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
				"k8s.label":                     {"best-city"},
				"k8s.namespace.label":           {"best-city"},
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
				attributes = AddNamespaceLabels(map[string][]string{}, client, tt.namespace.Name)
				require.Len(t, attributes, len(tt.expectedAttributesExactly))
			}, 5*time.Second, 100*time.Millisecond)

			// Then
			require.Len(t, attributes, len(tt.expectedAttributesExactly))
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

func testNamespace(modifier func(namespace *corev1.Namespace)) *corev1.Namespace {
	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "corev1",
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
	dynamicClient := testutil.NewFakeDynamicClient()
	return kclient.CreateClient(testclient.NewClientset(objects...), stopCh, "/oapi", kclient.MockAllPermitted(), dynamicClient)
}

func TestAddNodeLabels(t *testing.T) {
	type args struct {
		nodes      []*corev1.Node
		nodeName   string
		attributes map[string][]string
	}
	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{
			name: "should add label to attributes",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/region": "eu-central-1",
							},
						},
					},
				},
				nodeName:   "node1",
				attributes: map[string][]string{},
			},

			want: map[string][]string{
				"k8s.label.topology.kubernetes.io/region":      {"eu-central-1"},
				"k8s.node.label.topology.kubernetes.io/region": {"eu-central-1"},
				"k8s.label":      {"topology.kubernetes.io/region"},
				"k8s.node.label": {"topology.kubernetes.io/region"},
			},
		},
		{
			name: "should append label to existing attributes",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/region": "eu-central-1",
							},
						},
					},
				},
				nodeName: "node1",
				attributes: map[string][]string{
					"k8s.label.topology.kubernetes.io/region":      {"us-central-1"},
					"k8s.node.label.topology.kubernetes.io/region": {"us-central-1"},
				},
			},

			want: map[string][]string{
				"k8s.label.topology.kubernetes.io/region":      {"eu-central-1", "us-central-1"},
				"k8s.node.label.topology.kubernetes.io/region": {"eu-central-1", "us-central-1"},
				"k8s.label":      {"topology.kubernetes.io/region"},
				"k8s.node.label": {"topology.kubernetes.io/region"},
			},
		},
		{
			name: "should not append same label twice",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/region": "eu-central-1",
							},
						},
					},
				},
				nodeName: "node1",
				attributes: map[string][]string{
					"k8s.label.topology.kubernetes.io/region": {"eu-central-1"},
				},
			},

			want: map[string][]string{
				"k8s.label.topology.kubernetes.io/region":      {"eu-central-1"},
				"k8s.node.label.topology.kubernetes.io/region": {"eu-central-1"},
				"k8s.label":      {"topology.kubernetes.io/region"},
				"k8s.node.label": {"topology.kubernetes.io/region"},
			},
		},
		{
			name: "should not add filtered label",
			args: args{
				nodes: []*corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"topology.kubernetes.io/ignore-me": "foobar",
							},
						},
					},
				},
				nodeName:   "node1",
				attributes: map[string][]string{},
			},

			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, AddNodeLabels(tt.args.nodes, tt.args.nodeName, tt.args.attributes), "AddNodeLabels(%v, %v, %v)", tt.args.nodes, tt.args.nodeName, tt.args.attributes)
		})
	}
}

func TestAddFilteredLabels(t *testing.T) {
	filter := func(key string) bool {
		return !strings.Contains(key, "ignore")
	}

	tests := []struct {
		name             string
		labels           map[string]string
		attributes       map[string][]string
		prefixes         []string
		wantedAttributes map[string][]string
	}{
		{
			name: "should do noting",
		},
		{
			name:       "should add label to attributes",
			attributes: map[string][]string{},
			labels: map[string]string{
				"best-city":           "Kevelaer",
				"most-beautiful-city": "Brühl",
				"ignore-me":           "foobar",
			},
			prefixes: []string{"label", "another.label"},
			wantedAttributes: map[string][]string{
				"another.label.best-city":           {"Kevelaer"},
				"another.label.most-beautiful-city": {"Brühl"},
				"another.label":                     {"best-city", "most-beautiful-city"},
				"label.best-city":                   {"Kevelaer"},
				"label.most-beautiful-city":         {"Brühl"},
				"label":                             {"best-city", "most-beautiful-city"},
			},
		},
		{
			name: "should add label to existing attributes",
			attributes: map[string][]string{
				"label.expensive-city": {"Düsseldorf"},
				"label.best-city":      {"Brühl"},
				"label":                {"best-city", "expensive-city"},
			},
			labels: map[string]string{
				"best-city":           "Kevelaer",
				"most-beautiful-city": "Brühl",
				"ignore-me":           "foobar",
			},
			prefixes: []string{"label"},
			wantedAttributes: map[string][]string{
				"label.best-city":           {"Brühl", "Kevelaer"},
				"label.most-beautiful-city": {"Brühl"},
				"label.expensive-city":      {"Düsseldorf"},
				"label":                     {"best-city", "expensive-city", "most-beautiful-city"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.wantedAttributes, AddFilteredLabels(tt.attributes, filter, tt.labels, tt.prefixes...), "AddFilteredLabels(%v, filter, %v, %v)", tt.labels, tt.attributes, tt.prefixes)
		})
	}
}
