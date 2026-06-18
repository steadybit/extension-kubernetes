// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func httpProbe(path string) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(8080),
			},
		},
	}
}

func tcpProbe() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(8080)},
		},
	}
}

func TestAddProbePathAttributes(t *testing.T) {
	tests := []struct {
		name       string
		containers []corev1.Container
		expected   map[string][]string
	}{
		{
			name: "liveness and readiness http probes",
			containers: []corev1.Container{
				{LivenessProbe: httpProbe("/healthz"), ReadinessProbe: httpProbe("/ready")},
			},
			expected: map[string][]string{
				LivenessProbePathAttribute:  {"/healthz"},
				ReadinessProbePathAttribute: {"/ready"},
			},
		},
		{
			name: "empty path defaults to root",
			containers: []corev1.Container{
				{LivenessProbe: httpProbe("")},
			},
			expected: map[string][]string{
				LivenessProbePathAttribute: {"/"},
			},
		},
		{
			name: "non-http probes are ignored",
			containers: []corev1.Container{
				{LivenessProbe: tcpProbe(), ReadinessProbe: nil},
			},
			expected: map[string][]string{},
		},
		{
			name: "multiple containers are deduplicated and sorted",
			containers: []corev1.Container{
				{LivenessProbe: httpProbe("/healthz")},
				{LivenessProbe: httpProbe("/alive")},
				{LivenessProbe: httpProbe("/healthz")},
			},
			expected: map[string][]string{
				LivenessProbePathAttribute: {"/alive", "/healthz"},
			},
		},
		{
			name:       "no containers",
			containers: nil,
			expected:   map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attributes := map[string][]string{}
			AddProbePathAttributes(attributes, tt.containers)
			assert.Equal(t, tt.expected, attributes)
		})
	}
}
