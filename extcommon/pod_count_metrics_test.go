// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"testing"
	"time"

	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPodCountMetrics(t *testing.T) {
	original := extconfig.Config.ClusterName
	t.Cleanup(func() { extconfig.Config.ClusterName = original })
	extconfig.Config.ClusterName = "development"
	now := time.Now()

	metrics := BuildPodCountMetrics("k8s.deployment", "shop", "checkout", PodCountMetrics{Desired: 5, Current: 4, Ready: 3, Available: 2}, now)

	require.Len(t, metrics, 4)
	values := map[string]float64{}
	for _, m := range metrics {
		values[*m.Name] = m.Value
		assert.Equal(t, now, m.Timestamp)
		assert.Equal(t, map[string]string{
			"k8s.cluster-name": "development",
			"k8s.namespace":    "shop",
			"k8s.deployment":   "checkout",
		}, m.Metric)
	}
	assert.Equal(t, map[string]float64{
		"replicas_desired_count":   5,
		"replicas_current_count":   4,
		"replicas_ready_count":     3,
		"replicas_available_count": 2,
	}, values)
}
