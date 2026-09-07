package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewFakeDynamicClient_KnowsTheWatchedCrdsByDefault(t *testing.T) {
	c := NewFakeDynamicClient()

	for gvr := range defaultListKinds {
		list, err := c.Resource(gvr).Namespace("default").List(context.Background(), metav1.ListOptions{})
		require.NoError(t, err, "listing %s must not panic or fail", gvr.Resource)
		assert.Empty(t, list.Items)
	}
}

func TestNewFakeDynamicClient_RegistersOnlyTheGivenKinds(t *testing.T) {
	c := NewFakeDynamicClient(
		schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"},
		schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "WidgetList"},
	)

	list, err := c.Resource(schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestKindFromListKind(t *testing.T) {
	assert.Equal(t, "Gateway", kindFromListKind("GatewayList"))
}
