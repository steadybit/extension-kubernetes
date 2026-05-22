// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"fmt"
	"strconv"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	policyv1 "k8s.io/api/policy/v1"
)

// AddHpaAttributes rolls up HPA presence + min/max/metric onto a workload target's attribute map.
// When more than one HPA matches a workload, all min/max values are surfaced as multi-valued
// attributes. Per-metric details are only surfaced for the single-HPA case to keep the attribute
// shape unambiguous.
func AddHpaAttributes(attributes map[string][]string, hpas []*autoscalingv2.HorizontalPodAutoscaler) {
	if len(hpas) == 0 {
		attributes["k8s.specification.has-hpa"] = []string{"false"}
		return
	}
	attributes["k8s.specification.has-hpa"] = []string{"true"}

	names := make([]string, 0, len(hpas))
	for _, h := range hpas {
		names = append(names, h.Name)
	}
	attributes["k8s.hpa.name"] = names

	if len(hpas) > 1 {
		mins := make([]string, 0, len(hpas))
		maxs := make([]string, 0, len(hpas))
		for _, h := range hpas {
			mins = append(mins, hpaMinReplicas(h))
			maxs = append(maxs, strconv.Itoa(int(h.Spec.MaxReplicas)))
		}
		attributes["k8s.hpa.min-replicas"] = mins
		attributes["k8s.hpa.max-replicas"] = maxs
		return
	}

	h := hpas[0]
	attributes["k8s.hpa.min-replicas"] = []string{hpaMinReplicas(h)}
	attributes["k8s.hpa.max-replicas"] = []string{strconv.Itoa(int(h.Spec.MaxReplicas))}
	if metricTypes := hpaMetricTypes(h); len(metricTypes) > 0 {
		attributes["k8s.hpa.metric.type"] = metricTypes
	}
	if metricTargets := hpaMetricTargets(h); len(metricTargets) > 0 {
		attributes["k8s.hpa.metric.target"] = metricTargets
	}
}

func hpaMinReplicas(h *autoscalingv2.HorizontalPodAutoscaler) string {
	if h.Spec.MinReplicas == nil {
		// K8s defaults MinReplicas to 1 when unset; reflect that to the AI rather than emitting an empty string.
		return "1"
	}
	return strconv.Itoa(int(*h.Spec.MinReplicas))
}

func hpaMetricTypes(h *autoscalingv2.HorizontalPodAutoscaler) []string {
	out := make([]string, 0, len(h.Spec.Metrics))
	for _, m := range h.Spec.Metrics {
		out = append(out, string(m.Type))
	}
	return out
}

// hpaMetricTargets emits one string per HPA metric in the form "<metric-name>=<target>", e.g. "cpu=70%"
// or "memory=500Mi" or "requests_per_second=1k". The metric name and target shape vary by metric kind
// (Resource / Pods / Object / External / ContainerResource); we render whichever target value is set.
func hpaMetricTargets(h *autoscalingv2.HorizontalPodAutoscaler) []string {
	out := make([]string, 0, len(h.Spec.Metrics))
	for _, m := range h.Spec.Metrics {
		var name string
		var target autoscalingv2.MetricTarget
		switch m.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if m.Resource == nil {
				continue
			}
			name = string(m.Resource.Name)
			target = m.Resource.Target
		case autoscalingv2.ContainerResourceMetricSourceType:
			if m.ContainerResource == nil {
				continue
			}
			name = string(m.ContainerResource.Name)
			target = m.ContainerResource.Target
		case autoscalingv2.PodsMetricSourceType:
			if m.Pods == nil {
				continue
			}
			name = m.Pods.Metric.Name
			target = m.Pods.Target
		case autoscalingv2.ObjectMetricSourceType:
			if m.Object == nil {
				continue
			}
			name = m.Object.Metric.Name
			target = m.Object.Target
		case autoscalingv2.ExternalMetricSourceType:
			if m.External == nil {
				continue
			}
			name = m.External.Metric.Name
			target = m.External.Target
		default:
			continue
		}
		out = append(out, fmt.Sprintf("%s=%s", name, formatMetricTarget(target)))
	}
	return out
}

func formatMetricTarget(t autoscalingv2.MetricTarget) string {
	switch t.Type {
	case autoscalingv2.UtilizationMetricType:
		if t.AverageUtilization != nil {
			return fmt.Sprintf("%d%%", *t.AverageUtilization)
		}
	case autoscalingv2.AverageValueMetricType:
		if t.AverageValue != nil {
			return t.AverageValue.String()
		}
	case autoscalingv2.ValueMetricType:
		if t.Value != nil {
			return t.Value.String()
		}
	}
	return string(t.Type)
}

// AddPdbAttributes rolls up PDB presence + min-available / max-unavailable onto a workload target's
// attribute map. When more than one PDB matches the same workload, all values are surfaced as
// multi-valued attributes.
func AddPdbAttributes(attributes map[string][]string, pdbs []*policyv1.PodDisruptionBudget) {
	if len(pdbs) == 0 {
		attributes["k8s.specification.has-pdb"] = []string{"false"}
		return
	}
	attributes["k8s.specification.has-pdb"] = []string{"true"}

	names := make([]string, 0, len(pdbs))
	mins := make([]string, 0, len(pdbs))
	maxs := make([]string, 0, len(pdbs))
	for _, p := range pdbs {
		names = append(names, p.Name)
		if p.Spec.MinAvailable != nil {
			mins = append(mins, p.Spec.MinAvailable.String())
		}
		if p.Spec.MaxUnavailable != nil {
			maxs = append(maxs, p.Spec.MaxUnavailable.String())
		}
	}
	attributes["k8s.pdb.name"] = names
	if len(mins) > 0 {
		attributes["k8s.pdb.min-available"] = mins
	}
	if len(maxs) > 0 {
		attributes["k8s.pdb.max-unavailable"] = maxs
	}
}
