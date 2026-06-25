// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	policyv1 "k8s.io/api/policy/v1"
)

// AddHpaAttributes rolls up a single targeting flag onto a workload target's attribute map: whether
// any HPA matches the workload. The detailed HPA attributes (name, min/max replicas, metric type
// and target) were removed after they caused platform-DB churn for customers running clusters with
// thousands of pods — multi-HPA workloads (and HPA metric slices) produced multi-valued attributes
// whose order varied across discovery cycles, making the platform's target-diff detector re-write
// the target every cycle. Surfacing only the boolean keeps the targeting capability without the
// per-cycle write storm.
func AddHpaAttributes(attributes map[string][]string, hpas []*autoscalingv2.HorizontalPodAutoscaler) {
	if len(hpas) == 0 {
		attributes["k8s.specification.has-hpa"] = []string{"false"}
		return
	}
	attributes["k8s.specification.has-hpa"] = []string{"true"}
}

// AddPdbAttributes rolls up a single targeting flag onto a workload target's attribute map: whether
// any PDB matches the workload's pod template labels. See AddHpaAttributes for the rationale.
func AddPdbAttributes(attributes map[string][]string, pdbs []*policyv1.PodDisruptionBudget) {
	if len(pdbs) == 0 {
		attributes["k8s.specification.has-pdb"] = []string{"false"}
		return
	}
	attributes["k8s.specification.has-pdb"] = []string{"true"}
}
