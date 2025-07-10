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

// NginxDelayTrafficState contains state data for the NGINX delay traffic action
type NginxDelayTrafficState struct {
	NginxBaseState
	ResponseDelay        int
	ConditionPathPattern string
	ConditionHttpMethod  string
	ConditionHttpHeader  map[string]string
	AnnotationConfig     string
	IsEnterpriseNginx    bool
}

// NewNginxDelayTrafficAction creates a new delay traffic action
func NewNginxDelayTrafficAction() action_kit_sdk.Action[NginxDelayTrafficState] {
	return &NginxDelayTrafficAction{}
}

// NginxDelayTrafficAction implements the delay traffic action
type NginxDelayTrafficAction struct{}

// NewEmptyState creates an empty state object
func (a *NginxDelayTrafficAction) NewEmptyState() NginxDelayTrafficState {
	return NginxDelayTrafficState{}
}

// Describe returns the action description for the NGINX delay traffic action
func (a *NginxDelayTrafficAction) Describe() action_kit_api.ActionDescription {
	desc := getNginxActionDescription(
		NginxDelayTrafficActionId,
		"NGINX Delay Traffic",
		"Delay traffic by adding a response delay for requests matching specific paths.",
		"data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M16.5123%2010.4893C19.8057%2010.4893%2022.5182%2013.2017%2022.5182%2016.4951C22.5179%2019.7883%2019.8056%2022.5%2016.5123%2022.5C13.2192%2022.4998%2010.5077%2019.7882%2010.5074%2016.4951C10.5074%2013.2018%2013.2191%2010.4895%2016.5123%2010.4893ZM16.5123%2011.8447C13.994%2011.8449%2011.8629%2013.9767%2011.8629%2016.4951C11.8631%2019.0133%2013.8973%2021.1443%2016.5123%2021.1445C19.0306%2021.1445%2021.1615%2019.0134%2021.1617%2016.4951C21.1617%2013.9766%2019.0308%2011.8447%2016.5123%2011.8447ZM16.5123%205.83984V8.74512C15.3791%208.74517%2014.304%208.99783%2013.3258%209.44336V6.85645C13.3257%206.36256%2012.919%205.92685%2012.3961%205.92676H12.3375C11.8437%205.92693%2011.4079%206.33357%2011.4078%206.85645V10.6826C11.2528%2010.8279%2011.098%2010.9642%2010.9527%2011.1191L7.09726%206.50781C6.77759%206.11076%206.24477%205.92676%205.77988%205.92676C5.15042%205.92684%204.69502%206.34319%204.69492%206.85645V13.4824C4.69505%2013.9762%205.10184%2014.4119%205.6246%2014.4121H5.6832C6.1965%2014.4121%206.603%2014.0054%206.60312%2013.4824V8.66797L9.858%2012.543C9.17037%2013.7053%208.76328%2015.0422%208.76328%2016.4854C8.76331%2017.2795%208.88915%2018.0451%209.11191%2018.7715L9.00546%2018.8291L1.49863%2014.499V5.83008L9.00546%201.5L16.5123%205.83984ZM16.5123%2013.4922C16.8998%2013.4922%2017.191%2013.7825%2017.191%2014.1699V15.8164H18.8375C19.2249%2015.8164%2019.5152%2016.1077%2019.5152%2016.4951C19.515%2016.8823%2019.2248%2017.1728%2018.8375%2017.1729H16.5123C16.1252%2017.1727%2015.8348%2016.8822%2015.8346%2016.4951V14.1699C15.8346%2013.7826%2016.125%2013.4924%2016.5123%2013.4922Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E%0A",
	)

	// Add delay-specific parameters
	desc.Parameters = append(desc.Parameters,
		[]action_kit_api.ActionParameter{
			{
				Name:  "-response-header-",
				Type:  action_kit_api.ActionParameterTypeHeader,
				Label: "Response",
			},
			{
				Name:         "responseDelay",
				Label:        "Delay",
				Description:  extutil.Ptr("The delay in milliseconds to add to matching requests"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("500ms"),
				Required:     extutil.Ptr(true),
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
func (a *NginxDelayTrafficAction) Prepare(_ context.Context, state *NginxDelayTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	ingress, err := prepareNginxAction(&state.NginxBaseState, request)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare NGINX delay action: %w", err)
	}

	// Extract and validate delay parameter
	if delay, ok := request.Config["responseDelay"]; ok {
		switch v := delay.(type) {
		case float64:
			state.ResponseDelay = int(v)
		case int:
			state.ResponseDelay = v
		case string:
			return nil, fmt.Errorf("delay must be a number, got string: %s", v)
		default:
			return nil, fmt.Errorf("delay must be a number, got %T", v)
		}
	} else {
		return nil, fmt.Errorf("responseDelay parameter is required")
	}

	// Check for Enterprise NGINX based on ingress controller
	state.IsEnterpriseNginx = extutil.ToBool(request.Config["isEnterpriseNginx"])
	if ingressClass, ok := request.Target.Attributes["k8s.ingress.class"]; ok && len(ingressClass) > 0 {
		if controller, ok := request.Target.Attributes["k8s.ingress.controller"]; ok && len(controller) > 0 {
			if controller[0] == "nginx.org/ingress-controller" {
				// Override with detected Enterprise NGINX
				state.IsEnterpriseNginx = true
			}
		}
	}

	// Parse condition parameters
	state.ConditionPathPattern = extutil.ToString(request.Config["conditionPathPattern"])
	state.ConditionHttpMethod = extutil.ToString(request.Config["conditionHttpMethod"])

	if request.Config["conditionHttpHeader"] != nil {
		state.ConditionHttpHeader, err = extutil.ToKeyValue(request.Config, "conditionHttpHeader")
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTTP header condition: %w", err)
		}
	}

	// Validate that at least one condition is specified
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

	// Check for existing delay
	if ingress.Annotations != nil {
		if existingConfig, exists := ingress.Annotations[annotationKey]; exists && existingConfig != "" {
			existingLines := strings.Split(existingConfig, "\n")
			for _, line := range existingLines {
				if strings.Contains(line, "sb_sleep_ms") {
					return nil, fmt.Errorf("a delay rule already exists - cannot add another one")
				}
			}
		}
	}

	// Build NGINX configuration
	state.AnnotationConfig = buildNginxDelayConfig(state)

	return nil, nil
}

// buildNginxDelayConfig creates the NGINX configuration for traffic delay
func buildNginxDelayConfig(state *NginxDelayTrafficState) string {
	var configBuilder strings.Builder
	configBuilder.WriteString(getNginxStartMarker(state.ExecutionId) + "\n")

	configBuilder.WriteString(buildNginxDelayConfigContent(state))

	configBuilder.WriteString(getNginxEndMarker(state.ExecutionId) + "\n")
	return configBuilder.String()
}

// buildNginxDelayConfigContent creates configuration for NGINX Ingress Controller (both open source and enterprise)
func buildNginxDelayConfigContent(state *NginxDelayTrafficState) string {
	var configBuilder strings.Builder

	// Generate unique variable names based on execution ID
	shouldDelayVar := getNginxUniqueVariableName(state.ExecutionId, "should_delay")
	sleepDurationVar := getNginxUniqueVariableName(state.ExecutionId, "sleep_ms_duration")

	// Initialize the delay flag
	configBuilder.WriteString(fmt.Sprintf("set %s 0;\n", shouldDelayVar))

	// Add path pattern condition if provided
	if state.ConditionPathPattern != "" {
		configBuilder.WriteString(fmt.Sprintf("if ($request_uri ~* %s) {\n", state.ConditionPathPattern))
		configBuilder.WriteString(fmt.Sprintf("  set %s 1;\n", shouldDelayVar))
		configBuilder.WriteString("}\n")
	}

	// Add HTTP method condition if provided
	if state.ConditionHttpMethod != "" && state.ConditionHttpMethod != "*" {
		if state.ConditionPathPattern == "" {
			// If no path specified, simply set should_delay based on method
			configBuilder.WriteString(fmt.Sprintf("if ($request_method = %s) {\n", state.ConditionHttpMethod))
			configBuilder.WriteString(fmt.Sprintf("  set %s 1;\n", shouldDelayVar))
			configBuilder.WriteString("}\n")
		} else {
			// When path is also specified, we need to check if the method matches too
			// This ensures we only delay when both path AND method match
			configBuilder.WriteString(fmt.Sprintf("if ($request_method != %s) {\n", state.ConditionHttpMethod))
			configBuilder.WriteString(fmt.Sprintf("  set %s 0; # Reset if method doesn't match\n", shouldDelayVar))
			configBuilder.WriteString("}\n")
		}
	}

	// Add HTTP header conditions if provided
	if len(state.ConditionHttpHeader) > 0 {
		if state.ConditionPathPattern == "" && (state.ConditionHttpMethod == "" || state.ConditionHttpMethod == "*") {
			// If no path or method specified, set should_delay based only on headers
			for headerName, headerValue := range state.ConditionHttpHeader {
				normalizedHeaderName := strings.Replace(strings.ToLower(headerName), "-", "_", -1)
				configBuilder.WriteString(fmt.Sprintf("if ($http_%s ~* %s) {\n", normalizedHeaderName, headerValue))
				configBuilder.WriteString(fmt.Sprintf("  set %s 1;\n", shouldDelayVar))
				configBuilder.WriteString("}\n")
			}
		} else {
			// If other conditions are specified, we need to check headers too
			// This ensures that headers must also match when combined with other conditions
			for headerName, headerValue := range state.ConditionHttpHeader {
				normalizedHeaderName := strings.Replace(strings.ToLower(headerName), "-", "_", -1)
				configBuilder.WriteString(fmt.Sprintf("if ($http_%s !~* %s) {\n", normalizedHeaderName, headerValue))
				configBuilder.WriteString(fmt.Sprintf("  set %s 0; # Reset if header doesn't match\n", shouldDelayVar))
				configBuilder.WriteString("}\n")
			}
		}
	}

	// Set up a variable for the delay and then apply it unconditionally
	configBuilder.WriteString(fmt.Sprintf("set %s 0;\n", sleepDurationVar))
	configBuilder.WriteString(fmt.Sprintf("if (%s = 1) {\n", shouldDelayVar))
	configBuilder.WriteString(fmt.Sprintf("  set %s %d;\n", sleepDurationVar, state.ResponseDelay))
	configBuilder.WriteString("}\n")
	configBuilder.WriteString(fmt.Sprintf("sb_sleep_ms %s;\n", sleepDurationVar))

	return configBuilder.String()
}

// Start applies the NGINX configuration to begin delaying traffic
func (a *NginxDelayTrafficAction) Start(ctx context.Context, state *NginxDelayTrafficState) (*action_kit_api.StartResult, error) {
	if err := startNginxAction(&state.NginxBaseState, state.AnnotationConfig, state.IsEnterpriseNginx); err != nil {
		return nil, fmt.Errorf("failed to start NGINX delay traffic action: %w", err)
	}

	return nil, nil
}

// Stop removes the NGINX configuration to stop delaying traffic
func (a *NginxDelayTrafficAction) Stop(ctx context.Context, state *NginxDelayTrafficState) (*action_kit_api.StopResult, error) {
	if err := stopNginxAction(&state.NginxBaseState, state.IsEnterpriseNginx); err != nil {
		return nil, fmt.Errorf("failed to stop NGINX delay traffic action: %w", err)
	}

	return nil, nil
}
