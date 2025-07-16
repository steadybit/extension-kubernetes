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
		"data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M16.5123%2010.4893C19.8057%2010.4893%2022.5182%2013.2017%2022.5182%2016.4951C22.5179%2019.7883%2019.8056%2022.5%2016.5123%2022.5C13.2192%2022.4998%2010.5077%2019.7882%2010.5074%2016.4951C10.5074%2013.2018%2013.2191%2010.4895%2016.5123%2010.4893ZM16.5123%2011.8447C13.8971%2011.8449%2011.8629%2013.8799%2011.8629%2016.4951C11.8631%2019.1101%2013.8973%2021.1443%2016.5123%2021.1445C19.0306%2021.1445%2021.1615%2019.1103%2021.1617%2016.4951C21.1617%2013.8798%2019.1277%2011.8447%2016.5123%2011.8447ZM17.9654%2014.0732C18.256%2013.7826%2018.6436%2013.7826%2018.9342%2014.0732C19.2248%2014.267%2019.2248%2014.7514%2018.9342%2015.042L17.3844%2016.5918L18.9342%2018.1416C19.2247%2018.4322%2019.2248%2018.8198%2018.9342%2019.1104C18.6436%2019.4007%2018.2559%2019.4008%2017.9654%2019.1104L16.4156%2017.5605L14.8658%2019.1104C14.5753%2019.4007%2014.1876%2019.4008%2013.8971%2019.1104C13.6066%2018.8198%2013.6067%2018.4322%2013.8971%2018.1416L15.4469%2016.5918L13.8971%2015.042C13.6065%2014.7515%2013.6067%2014.3638%2013.8971%2014.0732C14.1877%2013.7826%2014.5752%2013.7826%2014.8658%2014.0732L16.4156%2015.623L17.9654%2014.0732ZM16.5123%205.83984V8.74512C15.3791%208.74517%2014.304%208.99783%2013.3258%209.44336V6.85645C13.3257%206.36256%2012.919%205.92685%2012.3961%205.92676H12.3375C11.8437%205.92693%2011.4079%206.33357%2011.4078%206.85645V10.6826C11.2528%2010.8279%2011.098%2010.9642%2010.9527%2011.1191L7.09726%206.50781C6.77759%206.11076%206.24477%205.92676%205.77988%205.92676C5.15042%205.92684%204.69502%206.34319%204.69492%206.85645V13.4824C4.69505%2013.9762%205.10184%2014.4119%205.6246%2014.4121H5.6832C6.1965%2014.4121%206.603%2014.0054%206.60312%2013.4824V8.66797L9.858%2012.543C9.17037%2013.7053%208.76328%2015.0422%208.76328%2016.4854C8.76331%2017.2795%208.88915%2018.0451%209.11191%2018.7715L9.00546%2018.8291L1.49863%2014.499V5.83008L9.00546%201.5L16.5123%205.83984Z%22%20fill%3D%22currentColor%22%2F%3E%0A%3C%2Fsvg%3E%0A",
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
	configBuilder.WriteString(GetNginxStartMarker(state.ExecutionId, NginxActionSubTypeBlock) + "\n")

	configBuilder.WriteString(buildNginxConfig(state))

	configBuilder.WriteString(GetNginxEndMarker(state.ExecutionId, NginxActionSubTypeBlock) + "\n")
	return configBuilder.String()
}

// buildNginxConfig creates configuration for NGINX Ingress Controller (both open source and enterprise)
func buildNginxConfig(state *NginxBlockTrafficState) string {
	var configBuilder strings.Builder

	// Generate unique variable names based on execution ID
	shouldBlockVar := getNginxUniqueVariableName(state.ExecutionId, "should_block")

	// Initialize the blocking flag
	configBuilder.WriteString(fmt.Sprintf("set %s 0;\n", shouldBlockVar))

	// Add path pattern condition if provided
	if state.ConditionPathPattern != "" {
		configBuilder.WriteString(fmt.Sprintf("if ($request_uri ~* %s) {\n", state.ConditionPathPattern))
		configBuilder.WriteString(fmt.Sprintf("  set %s 1;\n", shouldBlockVar))
		configBuilder.WriteString("}\n")
	}

	// Add HTTP method condition if provided
	if state.ConditionHttpMethod != "" && state.ConditionHttpMethod != "*" {
		if state.ConditionPathPattern == "" {
			// If no path specified, simply set should_block based on method
			configBuilder.WriteString(fmt.Sprintf("if ($request_method = %s) {\n", state.ConditionHttpMethod))
			configBuilder.WriteString(fmt.Sprintf("  set %s 1;\n", shouldBlockVar))
			configBuilder.WriteString("}\n")
		} else {
			// When path is also specified, we need to check if the method matches too
			// This ensures we only block when both path AND method match
			configBuilder.WriteString(fmt.Sprintf("if ($request_method != %s) {\n", state.ConditionHttpMethod))
			configBuilder.WriteString(fmt.Sprintf("  set %s 0; # Reset if method doesn't match\n", shouldBlockVar))
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
				configBuilder.WriteString(fmt.Sprintf("  set %s 1;\n", shouldBlockVar))
				configBuilder.WriteString("}\n")
			}
		} else {
			// If other conditions are specified, we need to check headers too
			// This ensures that headers must also match when combined with other conditions
			for headerName, headerValue := range state.ConditionHttpHeader {
				normalizedHeaderName := strings.Replace(strings.ToLower(headerName), "-", "_", -1)
				configBuilder.WriteString(fmt.Sprintf("if ($http_%s !~* %s) {\n", normalizedHeaderName, headerValue))
				configBuilder.WriteString(fmt.Sprintf("  set %s 0; # Reset if header doesn't match\n", shouldBlockVar))
				configBuilder.WriteString("}\n")
			}
		}
	}

	// Apply the block if conditions matched
	configBuilder.WriteString(fmt.Sprintf("if (%s = 1) {\n", shouldBlockVar))
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
	if err := stopNginxAction(&state.NginxBaseState, state.IsEnterpriseNginx, NginxActionSubTypeBlock); err != nil {
		return nil, fmt.Errorf("failed to stop NGINX block traffic action: %w", err)
	}

	return nil, nil
}
