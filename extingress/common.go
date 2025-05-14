// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"fmt"
	"github.com/google/uuid"
)

const (
	HAProxyIngressTargetType = "com.steadybit.extension_kubernetes.kubernetes-haproxy-ingress"
	AnnotationKey            = "haproxy-ingress.github.io/backend-config-snippet"
)

func getStartMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# BEGIN STEADYBIT - %s", executionId)
}

func getEndMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# END STEADYBIT - %s", executionId)
}
