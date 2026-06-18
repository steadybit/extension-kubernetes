// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extcommon

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	LivenessProbePathAttribute  = "k8s.specification.probes.liveness.path"
	ReadinessProbePathAttribute = "k8s.specification.probes.readiness.path"
)

// AddProbePathAttributes extracts the HTTP path of the liveness and readiness
// probes from the given containers and adds them as discovery attributes. Only
// HTTP probes (httpGet) expose a path; tcp/exec/grpc probes are ignored.
func AddProbePathAttributes(attributes map[string][]string, containers []corev1.Container) {
	probePaths := map[string][]string{}
	for i := range containers {
		if path := httpProbePath(containers[i].LivenessProbe); path != "" {
			probePaths[LivenessProbePathAttribute] = append(probePaths[LivenessProbePathAttribute], path)
		}
		if path := httpProbePath(containers[i].ReadinessProbe); path != "" {
			probePaths[ReadinessProbePathAttribute] = append(probePaths[ReadinessProbePathAttribute], path)
		}
	}
	MergeAttributes(attributes, probePaths)
}

// httpProbePath returns the HTTP path of an httpGet probe. An httpGet probe with
// an empty path targets the root "/", so that is reported. Non-HTTP probes
// return an empty string.
func httpProbePath(probe *corev1.Probe) string {
	if probe == nil || probe.HTTPGet == nil {
		return ""
	}
	if probe.HTTPGet.Path == "" {
		return "/"
	}
	return probe.HTTPGet.Path
}
