// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"fmt"
	"github.com/google/uuid"
	"strings"
)

const (
	HAProxyIngressTargetType = "com.steadybit.extension_kubernetes.haproxy-ingress"
  AnnotationKey            = "haproxy-ingress.github.io/backend-config-snippet"
)

func removeConfigBlock(config, startMarker, endMarker string) string {
	startIndex := strings.Index(config, startMarker)
	endIndex := strings.Index(config, endMarker)
	if startIndex == -1 || endIndex == -1 {
		return config
	}
	return config[:startIndex] + config[endIndex+len(endMarker):]
}


func getStartMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# BEGIN STEADYBIT - %s", executionId)
}

func getEndMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# END STEADYBIT - %s", executionId)
}
