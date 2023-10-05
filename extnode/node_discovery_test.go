// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extnode

import (
	"context"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
	"time"
)

func Test_getDiscoveredPods(t *testing.T) {
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
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	// When
	assert.Eventually(t, func() bool {
		return len(getDiscoveredNodeTargets(client)) == 1
	}, time.Second, 100*time.Millisecond)

	// Then
	targets := getDiscoveredNodeTargets(client)
	require.Len(t, targets, 1)
	target := targets[0]
	assert.Equal(t, "node-123", target.Id)
	assert.Equal(t, "node-123", target.Label)
	assert.Equal(t, NodeTargetType, target.TargetType)
	assert.Equal(t, map[string][]string{
		"host.hostname":    {"node-123"},
		"k8s.cluster-name": {"development"},
		"k8s.node.name":    {"node-123"},
	}, target.Attributes)
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	client := client.CreateClient(clientset, stopCh, "")
	return client, clientset
}
