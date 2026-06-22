// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extcommon

import (
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
)

// PodCountMetrics holds the four replica counts needed to build pod-count metric series.
type PodCountMetrics struct {
	Desired   int32
	Current   int32
	Ready     int32
	Available int32
}

// BuildPodCountMetrics produces the four replicas_* metrics consumed by the
// DeploymentReadinessWidget. labelKey is the resource-type label
// (e.g. "k8s.deployment" or "k8s.statefulset").
func BuildPodCountMetrics(labelKey, namespace, name string, counts PodCountMetrics, now time.Time) []action_kit_api.Metric {
	labels := map[string]string{
		"k8s.cluster-name": extconfig.Config.ClusterName,
		"k8s.namespace":    namespace,
		labelKey:           name,
	}
	return []action_kit_api.Metric{
		{Name: new("replicas_desired_count"), Metric: labels, Timestamp: now, Value: float64(counts.Desired)},
		{Name: new("replicas_current_count"), Metric: labels, Timestamp: now, Value: float64(counts.Current)},
		{Name: new("replicas_ready_count"), Metric: labels, Timestamp: now, Value: float64(counts.Ready)},
		{Name: new("replicas_available_count"), Metric: labels, Timestamp: now, Value: float64(counts.Available)},
	}
}
