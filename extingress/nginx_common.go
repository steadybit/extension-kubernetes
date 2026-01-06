/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

// Action IDs and constants for NGINX actions
const (
	NginxIngressTargetType       = "com.steadybit.extension_kubernetes.kubernetes-nginx-ingress"
	NginxBlockTrafficActionId    = "com.steadybit.extension_kubernetes.nginx-block-traffic"
	NginxDelayTrafficActionId    = "com.steadybit.extension_kubernetes.nginx-delay-traffic"
	nginxAnnotationKey           = "nginx.ingress.kubernetes.io/configuration-snippet"
	nginxEnterpriseAnnotationKey = "nginx.org/server-snippets"
	nginxActionSubTypeDelay      = "Delay"
	nginxActionSubTypeBlock      = "Block"
)

type NginxState struct {
	ExecutionId      uuid.UUID
	Namespace        string
	IngressName      string
	Matcher          RequestMatcher
	AnnotationKey    string
	AnnotationConfig string
}

type nginxAction struct {
	description             action_kit_api.ActionDescription
	subtype                 string
	annotationConfigFn      func(state *NginxState, config map[string]interface{}) string
	checkExistingFn         func(lines []string) error
	requiresSteadybitModule bool
}

// NewEmptyState creates an empty state object
func (a *nginxAction) NewEmptyState() NginxState {
	return NginxState{}
}

func (a *nginxAction) Describe() action_kit_api.ActionDescription {
	return a.description
}

// Prepare validates input parameters and prepares the state for execution
func (a *nginxAction) Prepare(_ context.Context, state *NginxState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	if a.requiresSteadybitModule {
		if err := validateNginxSteadybitModule(request.Target.Attributes); err != nil {
			return nil, fmt.Errorf("NGINX steadybit sleep module validation failed: %w", err)
		}
	}

	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]

	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	state.AnnotationKey = getNginxAnnotationKey(request)

	state.Matcher, err = parseRequestMatcher(request.Config)
	if err != nil {
		return nil, err
	}

	existingSnippet := strings.Split(ingress.Annotations[state.AnnotationKey], "\n")
	if err = checkNginxRuleConflicts(existingSnippet, state.Matcher); err != nil {
		return nil, err
	}

	if a.checkExistingFn != nil {
		if err = a.checkExistingFn(existingSnippet); err != nil {
			return nil, err
		}
	}

	state.AnnotationConfig = a.annotationConfigFn(state, request.Config)

	return nil, nil
}

// checkNginxRuleConflicts checks if the new rules would conflict with existing ones
func checkNginxRuleConflicts(existingLines []string, matcher RequestMatcher) error {
	if matcher.PathPattern == "" {
		return nil
	}

	for _, line := range existingLines {
		if strings.Contains(line, fmt.Sprintf("location ~ %s", matcher.PathPattern)) ||
			strings.Contains(line, fmt.Sprintf("location = %s", matcher.PathPattern)) {
			return fmt.Errorf("a rule for path %s already exists", matcher.PathPattern)
		}
	}

	return nil
}

// Start applies the NGINX configuration to begin blocking traffic
func (a *nginxAction) Start(_ context.Context, state *NginxState) (*action_kit_api.StartResult, error) {
	log.Debug().Msgf("Adding new %s configuration %s:%s", a.description.Label, state.AnnotationKey, state.AnnotationConfig)

	finalAnnotation, err := client.K8S.UpdateIngressAnnotationWithReturn(context.Background(), state.Namespace, state.IngressName, state.AnnotationKey, state.AnnotationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start %s action: %w", a.description.Label, err)
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
			return nil, fmt.Errorf("cannot start action: both delay and block actions are already active on ingress %s/%s - they would interfere with each other on the same matching request", state.Namespace, state.IngressName)
		}
	}

	return nil, nil
}

// Stop removes the NGINX configuration to stop blocking traffic
func (a *nginxAction) Stop(_ context.Context, state *NginxState) (*action_kit_api.StopResult, error) {
	err := client.K8S.RemoveAnnotationBlock(
		context.Background(),
		state.Namespace,
		state.IngressName,
		state.AnnotationKey,
		state.ExecutionId,
		getNginxStartMarker(state.ExecutionId, a.subtype),
		getNginxEndMarker(state.ExecutionId, a.subtype),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to stop %s action: %w", a.description.Label, err)
	}

	return nil, nil
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

// getNginxStartMarker Helper functions similar to HAProxy implementation
func getNginxStartMarker(executionId uuid.UUID, subtype string) string {
	return fmt.Sprintf("# BEGIN STEADYBIT - %s - %s\n", subtype, executionId)
}

func getNginxEndMarker(executionId uuid.UUID, subtype string) string {
	return fmt.Sprintf("# END STEADYBIT - %s - %s\n", subtype, executionId)
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

// getNginxAnnotationKey determines the correct annotation key based on the request and whether Enterprise NGINX is used
func getNginxAnnotationKey(request action_kit_api.PrepareActionRequestBody) string {
	annotationKey := nginxAnnotationKey

	if ingressClass, ok := request.Target.Attributes["k8s.ingress.class"]; ok && len(ingressClass) > 0 {
		if controller, ok := request.Target.Attributes["k8s.ingress.controller"]; ok && len(controller) > 0 {
			if controller[0] == "nginx.org/ingress-controller" {
				// Override with detected Enterprise NGINX
				annotationKey = nginxEnterpriseAnnotationKey
			}
		}
	}
	if extutil.ToBool(request.Config["isEnterpriseNginx"]) {
		annotationKey = nginxEnterpriseAnnotationKey
	}

	return annotationKey
}

func buildConfigForMatcher(matcher RequestMatcher, varName string) string {
	var config strings.Builder

	config.WriteString(fmt.Sprintf("set %s 1;\n", varName))

	if matcher.PathPattern != "" {
		config.WriteString(fmt.Sprintf("if ($request_uri !~* %s) { set %s 0; }\n", matcher.PathPattern, varName))
	}

	if matcher.HttpMethod != "" && matcher.HttpMethod != "*" {
		config.WriteString(fmt.Sprintf("if ($request_method != %s) { set %s 0; }\n", matcher.HttpMethod, varName))
	}

	for headerName, headerValue := range matcher.HttpHeader {
		normalizedHeaderName := strings.Replace(strings.ToLower(headerName), "-", "_", -1)
		config.WriteString(fmt.Sprintf("if ($http_%s !~* %s) { set %s 0; }\n", normalizedHeaderName, headerValue, varName))
	}

	return config.String()
}
