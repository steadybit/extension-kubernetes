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

func NewResponseBodyAction(k8s *client.Client) action_kit_sdk.Action[ActionState] {
	return &backendTrafficPolicyAction{
		k8s:              k8s,
		description:      getResponseBodyDescription(),
		subtype:          "response-body",
		buildFaultSpecFn: buildResponseBodyFaultSpec,
	}
}

func getResponseBodyDescription() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		ResponseBodyActionId,
		"Envoy Overwrite Response",
		"Overwrite the response body (and status code) for a percentage of the traffic on an Envoy Gateway HTTP route. "+
			"Internally aborts matching requests with a sentinel status which a response override rewrites to the configured body and status.",
	)
	desc.Parameters = append(desc.Parameters,
		action_kit_api.ActionParameter{
			Name:         "statusCode",
			Label:        "HTTP Status Code",
			Description:  extutil.Ptr("The HTTP status code the client will receive alongside the overwritten body."),
			Type:         action_kit_api.ActionParameterTypeInteger,
			DefaultValue: extutil.Ptr("200"),
			Required:     extutil.Ptr(true),
			MinValue:     extutil.Ptr(200),
			MaxValue:     extutil.Ptr(600),
		},
		action_kit_api.ActionParameter{
			Name:        "body",
			Label:       "Response Body",
			Description: extutil.Ptr("The response body returned to the client."),
			Type:        action_kit_api.ActionParameterTypeTextarea,
			Required:    extutil.Ptr(true),
		},
		action_kit_api.ActionParameter{
			Name:         "contentType",
			Label:        "Content Type",
			Description:  extutil.Ptr("The Content-Type header set on the overwritten response."),
			Type:         action_kit_api.ActionParameterTypeString,
			DefaultValue: extutil.Ptr("application/json"),
			Required:     extutil.Ptr(false),
			Advanced:     extutil.Ptr(true),
		},
	)
	return withSectionNameParameter(desc)
}

// sentinelStatus is the internal HTTP status used to abort matching traffic before the response
// override rewrites it. 418 ("I'm a teapot") is used because backends practically never return it,
// avoiding collisions with genuine responses. The client never sees this status.
const sentinelStatus int64 = 418

func buildResponseBodyFaultSpec(config map[string]any) (map[string]any, error) {
	statusCode := extutil.ToInt64(config["statusCode"])
	if statusCode < 200 || statusCode > 600 {
		return nil, fmt.Errorf("statusCode must be between 200 and 600")
	}
	body := extutil.ToString(config["body"])
	if body == "" {
		return nil, fmt.Errorf("body must not be empty")
	}
	contentType := extutil.ToString(config["contentType"])
	if contentType == "" {
		contentType = "application/json"
	}

	return map[string]any{
		// Abort a percentage of the traffic locally with the sentinel status...
		"faultInjection": map[string]any{
			"abort": map[string]any{
				"httpStatus": sentinelStatus,
				"percentage": percentageFromConfig(config),
			},
		},
		// ...then rewrite that locally-generated response to the configured body and status. source=Local
		// ensures genuine backend responses carrying the same status code are left untouched.
		"responseOverride": []any{
			map[string]any{
				"match": map[string]any{
					"statusCodes": []any{
						map[string]any{
							"type":  "Value",
							"value": sentinelStatus,
						},
					},
				},
				"response": map[string]any{
					"statusCode":  statusCode,
					"contentType": contentType,
					"body": map[string]any{
						"type":   "Inline",
						"inline": body,
					},
				},
				"source": "Local",
			},
		},
	}, nil
}
