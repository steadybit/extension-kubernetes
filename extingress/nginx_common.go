/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// Action IDs and constants for NGINX actions
const (
	NginxIngressTargetType       = "com.steadybit.extension_kubernetes.kubernetes-nginx-ingress"
	NginxAnnotationKey           = "nginx.ingress.kubernetes.io/configuration-snippet"
	NginxEnterpriseAnnotationKey = "nginx.org/server-snippets"
	NginxBlockTrafficActionId    = "com.steadybit.extension_kubernetes.nginx-block-traffic"
	NginxDelayTrafficActionId    = "com.steadybit.extension_kubernetes.nginx-delay-traffic"
	NginxActionSubTypeDelay      = "Delay"
	NginxActionSubTypeBlock      = "Block"
)

// NginxBaseState contains common state for NGINX-related actions
type NginxBaseState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
}

// prepareNginxAction contains common preparation logic for NGINX actions
func prepareNginxAction(state *NginxBaseState, request action_kit_api.PrepareActionRequestBody) (*networkingv1.Ingress, error) {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]

	// Check ingress availability
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}
	return ingress, nil
}

// startNginxAction contains common start logic for NGINX actions
func startNginxAction(state *NginxBaseState, annotationConfig string, isEnterprise bool) error {
	log.Debug().Msgf("Adding new NGINX configuration: %s", annotationConfig)

	annotationKey := NginxAnnotationKey
	if isEnterprise {
		annotationKey = NginxEnterpriseAnnotationKey
	}

	finalAnnotation, err := client.K8S.UpdateIngressAnnotationWithReturn(context.Background(), state.Namespace, state.IngressName, annotationKey, annotationConfig)
	if err != nil {
		return err
	}

	// Check for conflicting actions in the final annotation
	if finalAnnotation != "" {
		lines := strings.Split(finalAnnotation, "\n")
		hasDelayAction := false
		hasBlockAction := false

		for _, line := range lines {
			if strings.Contains(line, "BEGIN STEADYBIT - Delay") {
				hasDelayAction = true
			}
			if strings.Contains(line, "BEGIN STEADYBIT - Block") {
				hasBlockAction = true
			}
		}

		// Return error if both actions are present
		if hasDelayAction && hasBlockAction {
			return fmt.Errorf("cannot start action: both delay and block actions are already active on ingress %s/%s - they would interfere with each other on the same matching request", state.Namespace, state.IngressName)
		}
	}

	return nil
}

// stopNginxAction contains common stop logic for NGINX actions
func stopNginxAction(state *NginxBaseState, isEnterprise bool, subtype string) error {
	annotationKey := NginxAnnotationKey
	if isEnterprise {
		annotationKey = NginxEnterpriseAnnotationKey
	}

	err := client.K8S.RemoveAnnotationBlock(
		context.Background(),
		state.Namespace,
		state.IngressName,
		annotationKey,
		state.ExecutionId,
		GetNginxStartMarker(state.ExecutionId, subtype),
		GetNginxEndMarker(state.ExecutionId, subtype),
	)
	if err != nil {
		return fmt.Errorf("failed to remove NGINX configuration: %w", err)
	}

	return nil
}

// getNginxActionDescription returns common action description elements
func getNginxActionDescription(id string, label string, description string, icon string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr(icon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: NginxIngressTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "ingress",
					Description: extutil.Ptr("Find ingress by cluster, namespace and ingress"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.ingress=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  extutil.Ptr("The duration of the action. The ingress will be affected for the specified duration."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
		},
	}
}

// GetNginxStartMarker Helper functions similar to HAProxy implementation
func GetNginxStartMarker(executionId uuid.UUID, subtype string) string {
	return fmt.Sprintf("# BEGIN STEADYBIT - %s - %s", subtype, executionId)
}

func GetNginxEndMarker(executionId uuid.UUID, subtype string) string {
	return fmt.Sprintf("# END STEADYBIT - %s - %s", subtype, executionId)
}

// getNginxVariablePrefix generates a unique variable prefix based on execution ID
func getNginxVariablePrefix(executionId uuid.UUID) string {
	// Use only the first 8 characters of the UUID (without hyphens) to keep variable names manageable
	return strings.Replace(executionId.String(), "-", "", -1)
}

// getNginxUniqueVariableName generates a unique NGINX variable name
func getNginxUniqueVariableName(executionId uuid.UUID, baseName string) string {
	return fmt.Sprintf("$sb_%s_%s", baseName, getNginxVariablePrefix(executionId))
}

// validateNginxSteadybitModule checks if the ngx_steadybit_sleep_module.so is loaded by checking nginx configuration and module files
func validateNginxSteadybitModule(targetAttributes map[string][]string) error {
	// Get the controller namespace from target attributes
	var controllerNamespace string
	if ns, exists := targetAttributes["k8s.nginx.controller.namespace"]; exists && len(ns) > 0 {
		controllerNamespace = ns[0]
	} else {
		// Fallback to common NGINX controller namespaces
		namespaces := []string{
			"ingress-nginx",
			"nginx-ingress",
			"kube-system",
			"default",
		}

		// Try to find NGINX controller pods in common namespaces
		for _, ns := range namespaces {
			labelSelector := &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "ingress-nginx",
				},
			}
			pods := client.K8S.PodsByLabelSelector(labelSelector, ns)

			if len(pods) == 0 {
				labelSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "nginx-ingress",
					},
				}
				pods = client.K8S.PodsByLabelSelector(labelSelector, ns)
			}

			if len(pods) > 0 {
				controllerNamespace = ns
				break
			}
		}

		if controllerNamespace == "" {
			return fmt.Errorf("no NGINX ingress controller pods found in any common namespace")
		}
	}

	// Get NGINX ingress controller pods using label selector
	labelSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/name": "ingress-nginx",
		},
	}
	pods := client.K8S.PodsByLabelSelector(labelSelector, controllerNamespace)

	if len(pods) == 0 {
		// Try alternative label selector for different NGINX ingress installations
		labelSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "nginx-ingress",
			},
		}
		pods = client.K8S.PodsByLabelSelector(labelSelector, controllerNamespace)
	}

	if len(pods) == 0 {
		return fmt.Errorf("no NGINX ingress controller pods found in namespace %s", controllerNamespace)
	}

	// Check the first available pod for the module
	pod := pods[0]
	containerName := "controller" // Default container name for NGINX ingress controller

	// Try to find the correct container name
	for _, container := range pod.Spec.Containers {
		if strings.Contains(container.Name, "nginx") || strings.Contains(container.Name, "controller") {
			containerName = container.Name
			break
		}
	}

	// Check multiple possible nginx.conf locations
	configPaths := []string{
		"/etc/nginx/nginx.conf",
		"/usr/local/nginx/conf/nginx.conf",
		"/opt/nginx/conf/nginx.conf",
	}

	var configContent string
	var configPath string

	for _, path := range configPaths {
		output, err := client.K8S.ExecInPod(context.Background(), controllerNamespace, pod.Name, containerName, []string{"cat", path})
		if err == nil {
			configContent = output
			configPath = path
			break
		}
	}

	if configContent == "" {
		return fmt.Errorf("failed to read nginx.conf from pod %s: could not find configuration at any of the expected paths %v", pod.Name, configPaths)
	}

	// Check if the steadybit sleep module is loaded via load_module directive
	if strings.Contains(configContent, "ngx_steadybit_sleep_module") {
		log.Debug().Msgf("NGINX steadybit sleep module is loaded via load_module directive in %s in pod %s", configPath, pod.Name)
		return nil
	}

	// If not found in main config, check if module file exists in common module directories
	modulePaths := []string{
		"/etc/nginx/modules/ngx_steadybit_sleep_module.so",
		"/usr/local/nginx/modules/ngx_steadybit_sleep_module.so",
		"/opt/nginx/modules/ngx_steadybit_sleep_module.so",
		"/usr/lib/nginx/modules/ngx_steadybit_sleep_module.so",
	}

	for _, modulePath := range modulePaths {
		exists, err := client.K8S.FileExistsInPod(context.Background(), controllerNamespace, pod.Name, containerName, modulePath)
		if err == nil && exists {
			log.Debug().Msgf("Found ngx_steadybit_sleep_module.so at %s in pod %s, but it's not loaded in nginx.conf", modulePath, pod.Name)
			return fmt.Errorf("ngx_steadybit_sleep_module.so exists at %s but is not loaded. Please add 'load_module %s;' to the nginx configuration", modulePath, modulePath)
		}
	}

	return fmt.Errorf("ngx_steadybit_sleep_module is not loaded in NGINX ingress controller pod %s. Please ensure the module is installed and loaded with 'load_module /path/to/ngx_steadybit_sleep_module.so;' in the nginx configuration at %s", pod.Name, configPath)
}

// NginxModuleValidator interface for validating NGINX modules
type NginxModuleValidator interface {
	ValidateNginxSteadybitModule(targetAttributes map[string][]string) error
}

// DefaultNginxModuleValidator is the default implementation
type DefaultNginxModuleValidator struct{}

// ValidateNginxSteadybitModule validates the NGINX steadybit module
func (v *DefaultNginxModuleValidator) ValidateNginxSteadybitModule(targetAttributes map[string][]string) error {
	return validateNginxSteadybitModule(targetAttributes)
}

// NoOpNginxModuleValidator is a no-op validator for testing
type NoOpNginxModuleValidator struct{}

// ValidateNginxSteadybitModule does nothing (for testing)
func (v *NoOpNginxModuleValidator) ValidateNginxSteadybitModule(targetAttributes map[string][]string) error {
	return nil
}

// Global validator instance (can be overridden for testing)
var nginxModuleValidator NginxModuleValidator = &DefaultNginxModuleValidator{}
