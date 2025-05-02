// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"fmt"
	"github.com/google/uuid"
	extension_kit "github.com/steadybit/extension-kit"
	networkingv1 "k8s.io/api/networking/v1"
	"os/exec"
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

func updateIngress(namespace string, ingressName string, annotationKey string, ingress *networkingv1.Ingress) error {
	cmd := exec.Command("kubectl", "annotate", "ingress", fmt.Sprintf("%s", ingressName), fmt.Sprintf("%s=%s", annotationKey, ingress.Annotations[annotationKey]), "--overwrite", fmt.Sprintf("--namespace=%s", namespace), "--overwrite")
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return extension_kit.ToError(fmt.Sprintf("Failed to update ingress: %s", cmdOut), cmdErr)
	}
	return nil
}

