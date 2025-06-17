/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package extingress

import (
	"context"
	"fmt"
	"strings"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
)

// NginxBlockTrafficState contains state data for the NGINX block traffic action
type NginxBlockTrafficState struct {
	NginxBaseState
	ResponseStatusCode   int
	ConditionPathPattern string
	ConditionHttpMethod  string
	ConditionHttpHeader  map[string]string
	AnnotationConfig     string
	IsEnterpriseNginx    bool
}

// NewNginxBlockTrafficAction creates a new block traffic action
func NewNginxBlockTrafficAction() action_kit_sdk.Action[NginxBlockTrafficState] {
	return &NginxBlockTrafficAction{}
}

// NginxBlockTrafficAction implements the block traffic action
type NginxBlockTrafficAction struct{}

// NewEmptyState creates an empty state object
func (a *NginxBlockTrafficAction) NewEmptyState() NginxBlockTrafficState {
	return NginxBlockTrafficState{}
}

// Describe returns the action description for the NGINX block traffic action
func (a *NginxBlockTrafficAction) Describe() action_kit_api.ActionDescription {
	desc := getNginxActionDescription(
		NginxBlockTrafficActionId,
		"NGINX Block Traffic",
		"Block traffic by returning a custom HTTP status code for requests matching specific paths.",
		"data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M18.7198%203.75989V7.79011H14.4V3.75989H18.7198ZM10.08%203.75989V7.79011H5.76018V3.75989H10.08ZM18.7198%2011.8499V15.8801H14.4V11.8499H18.7198ZM10.08%2011.8499V15.8801H5.76018V11.8499H10.08ZM18.7198%2019.9399V24H14.4V19.9399H18.7198ZM10.08%2019.9399V24H5.76018V19.9399H10.08ZM4.32016%2019.9399V24H0V19.9399H4.32016ZM24%2019.9399V24H19.6798V19.9399H24ZM4.32016%2011.8499V15.8801H0V11.8499H4.32016ZM24%2011.8499V15.8801H19.6798V11.8499H24ZM4.32016%203.75989V7.79011H0V3.75989H4.32016ZM24%203.75989V7.79011H19.6798V3.75989H24ZM24%200H0V2.27998H24V0Z%22%20fill%3D%22%23009639%22%2F%3E%0A%3C%2Fsvg%3E",
	)

	// Add block-specific parameter
	desc.Parameters = append(desc.Parameters,
		[]action_kit_api.ActionParameter{
			{
				Name:  "-response-header-",
				Type:  action_kit_api.ActionParameterTypeHeader,
				Label: "Response",
			},
			{
				Name:         "responseStatusCode",
				Label:        "Status Code",
				Description:  extutil.Ptr("The status code which should get returned."),
				Type:         action_kit_api.ActionParameterTypeInteger,
				MinValue:     extutil.Ptr(100),
				MaxValue:     extutil.Ptr(999),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("503"),
			},
			{
				Name:         "isEnterpriseNginx",
				Label:        "Force Enterprise NGINX",
				Description:  extutil.Ptr("Whether to use Enterprise NGINX configuration (nginx.org/server-snippets) instead of open source (nginx.ingress.kubernetes.io/configuration-snippet)."),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("false"),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
			},
		}...,
	)
	desc.Parameters = append(desc.Parameters, getConditionsParameters()...)

	return desc
}

// Prepare validates input parameters and prepares the state for execution
func (a *NginxBlockTrafficAction) Prepare(_ context.Context, state *NginxBlockTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	ingress, err := prepareNginxAction(&state.NginxBaseState, request)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare NGINX block action: %w", err)
	}

	// Extract parameters from request
	state.ResponseStatusCode = extutil.ToInt(request.Config["responseStatusCode"])
	state.ConditionPathPattern = extutil.ToString(request.Config["conditionPathPattern"])
	state.ConditionHttpMethod = extutil.ToString(request.Config["conditionHttpMethod"])
	state.IsEnterpriseNginx = extutil.ToBool(request.Config["isEnterpriseNginx"])

	// Check for Enterprise NGINX based on ingress controller
	if ingressClass, ok := request.Target.Attributes["k8s.ingress.class"]; ok && len(ingressClass) > 0 {
		if controller, ok := request.Target.Attributes["k8s.ingress.controller"]; ok && len(controller) > 0 {
			if controller[0] == "nginx.org/ingress-controller" {
				// Override with detected Enterprise NGINX
				state.IsEnterpriseNginx = true
			}
		}
	}
	
	if request.Config["conditionHttpHeader"] != nil {
		state.ConditionHttpHeader, err = extutil.ToKeyValue(request.Config, "conditionHttpHeader")
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTTP header condition: %w", err)
		}
	}

	// Validate conditions
	if state.ConditionPathPattern == "" && state.ConditionHttpMethod == "" && len(state.ConditionHttpHeader) == 0 {
		return nil, fmt.Errorf("at least one condition (path, method, or header) is required")
	}

	// Check for conflicts with existing rules
	annotationKey := NginxAnnotationKey
	if state.IsEnterpriseNginx {
		annotationKey = NginxEnterpriseAnnotationKey
	}

	if state.ConditionPathPattern != "" && ingress.Annotations != nil {
		if existingConfig, exists := ingress.Annotations[annotationKey]; exists && existingConfig != "" {
			existingLines := strings.Split(existingConfig, "\n")
			for _, line := range existingLines {
				// Check for location block with same path
				if strings.Contains(line, fmt.Sprintf("location ~ %s", state.ConditionPathPattern)) ||
					strings.Contains(line, fmt.Sprintf("location = %s", state.ConditionPathPattern)) {
					return nil, fmt.Errorf("a rule for path %s already exists", state.ConditionPathPattern)
				}
			}
		}
	}

	// Build NGINX configuration
	state.AnnotationConfig = buildNginxBlockConfig(state)

	return nil, nil
}

// buildNginxBlockConfig creates the NGINX configuration for blocking traffic
func buildNginxBlockConfig(state *NginxBlockTrafficState) string {
	var configBuilder strings.Builder
	configBuilder.WriteString(getNginxStartMarker(state.ExecutionId) + "\n")

	configBuilder.WriteString(buildNginxConfig(state))

	configBuilder.WriteString(getNginxEndMarker(state.ExecutionId) + "\n")
	return configBuilder.String()
}

// buildNginxConfig creates configuration for NGINX Ingress Controller (both open source and enterprise)
func buildNginxConfig(state *NginxBlockTrafficState) string {
	var configBuilder strings.Builder

	// Initialize the blocking flag
	configBuilder.WriteString("set $should_block 0;\n")

	// Add path pattern condition if provided
	if state.ConditionPathPattern != "" {
		configBuilder.WriteString(fmt.Sprintf("if ($request_uri ~* %s) {\n", state.ConditionPathPattern))
		configBuilder.WriteString("  set $should_block 1;\n")
		configBuilder.WriteString("}\n")
	}

	// Add HTTP method condition if provided
	if state.ConditionHttpMethod != "" && state.ConditionHttpMethod != "*" {
		if state.ConditionPathPattern == "" {
			// If no path specified, simply set should_block based on method
			configBuilder.WriteString(fmt.Sprintf("if ($request_method = %s) {\n", state.ConditionHttpMethod))
			configBuilder.WriteString("  set $should_block 1;\n")
			configBuilder.WriteString("}\n")
		} else {
			// When path is also specified, we need to check if the method matches too
			// This ensures we only block when both path AND method match
			configBuilder.WriteString(fmt.Sprintf("if ($request_method != %s) {\n", state.ConditionHttpMethod))
			configBuilder.WriteString("  set $should_block 0; # Reset if method doesn't match\n")
			configBuilder.WriteString("}\n")
		}
	}

	// Add HTTP header conditions if provided
	if len(state.ConditionHttpHeader) > 0 {
		if state.ConditionPathPattern == "" && (state.ConditionHttpMethod == "" || state.ConditionHttpMethod == "*") {
			// If no path or method specified, set should_block based only on headers
			for headerName, headerValue := range state.ConditionHttpHeader {
				normalizedHeaderName := strings.Replace(strings.ToLower(headerName), "-", "_", -1)
				configBuilder.WriteString(fmt.Sprintf("if ($http_%s ~* %s) {\n", normalizedHeaderName, headerValue))
				configBuilder.WriteString("  set $should_block 1;\n")
				configBuilder.WriteString("}\n")
			}
		} else {
			// If other conditions are specified, we need to check headers too
			// This ensures that headers must also match when combined with other conditions
			for headerName, headerValue := range state.ConditionHttpHeader {
				normalizedHeaderName := strings.Replace(strings.ToLower(headerName), "-", "_", -1)
				configBuilder.WriteString(fmt.Sprintf("if ($http_%s !~* %s) {\n", normalizedHeaderName, headerValue))
				configBuilder.WriteString("  set $should_block 0; # Reset if header doesn't match\n")
				configBuilder.WriteString("}\n")
			}
		}
	}

	// Apply the block if conditions matched
	configBuilder.WriteString("if ($should_block = 1) {\n")
	configBuilder.WriteString(fmt.Sprintf("  return %d;\n", state.ResponseStatusCode))
	configBuilder.WriteString("}\n")

	return configBuilder.String()
}

// Start applies the NGINX configuration to begin blocking traffic
func (a *NginxBlockTrafficAction) Start(ctx context.Context, state *NginxBlockTrafficState) (*action_kit_api.StartResult, error) {
	if err := startNginxAction(&state.NginxBaseState, state.AnnotationConfig, state.IsEnterpriseNginx); err != nil {
		return nil, fmt.Errorf("failed to start NGINX block traffic action: %w", err)
	}

	return nil, nil
}

// Stop removes the NGINX configuration to stop blocking traffic
func (a *NginxBlockTrafficAction) Stop(ctx context.Context, state *NginxBlockTrafficState) (*action_kit_api.StopResult, error) {
	if err := stopNginxAction(&state.NginxBaseState, state.IsEnterpriseNginx); err != nil {
		return nil, fmt.Errorf("failed to stop NGINX block traffic action: %w", err)
	}

	return nil, nil
}
