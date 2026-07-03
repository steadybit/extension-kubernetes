// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

func NewStatusAction(k8s *client.Client) action_kit_sdk.Action[ActionState] {
	return &backendTrafficPolicyAction{
		k8s:              k8s,
		description:      getStatusDescription(),
		subtype:          "status",
		buildFaultSpecFn: buildStatusFaultSpec,
	}
}

func getStatusDescription() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		StatusActionId,
		"Envoy Abort Traffic",
		"Abort a percentage of the traffic on an Envoy Gateway HTTP route with a given HTTP status code using a BackendTrafficPolicy.",
	)
	desc.Parameters = append(desc.Parameters, action_kit_api.ActionParameter{
		Name:         "statusCode",
		Label:        "HTTP Status Code",
		Description:  extutil.Ptr("The HTTP status code returned for aborted requests."),
		Type:         action_kit_api.ActionParameterTypeInteger,
		DefaultValue: extutil.Ptr("500"),
		Required:     extutil.Ptr(true),
		MinValue:     extutil.Ptr(200),
		MaxValue:     extutil.Ptr(600),
	})
	return withSectionNameParameter(desc)
}

func buildStatusFaultSpec(config map[string]any) (map[string]any, error) {
	statusCode := extutil.ToInt64(config["statusCode"])
	if statusCode < 200 || statusCode > 600 {
		return nil, fmt.Errorf("statusCode must be between 200 and 600")
	}
	return map[string]any{
		"faultInjection": map[string]any{
			"abort": map[string]any{
				"httpStatus": statusCode,
				"percentage": percentageFromConfig(config),
			},
		},
	}, nil
}
