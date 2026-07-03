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

// sentinelStatus is the internal HTTP status used to abort traffic when the response body is
// overwritten: Envoy aborts with this code and a responseOverride rewrites it to the configured
// status and body. 418 ("I'm a teapot") is used because backends practically never return it, and
// the client never sees it. The configured status code must therefore not be 418 (see below).
const sentinelStatus int64 = 418

func NewAbortAction(k8s *client.Client) action_kit_sdk.Action[ActionState] {
	return &backendTrafficPolicyAction{
		k8s:              k8s,
		description:      getAbortDescription(),
		subtype:          "abort",
		buildFaultSpecFn: buildAbortFaultSpec,
	}
}

func getAbortDescription() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		AbortActionId,
		"Envoy Abort Traffic",
		"Abort a percentage of the traffic on an Envoy Gateway HTTP route with a given HTTP status code. Optionally overwrite the response body returned to clients.",
	)
	desc.Parameters = append(desc.Parameters,
		action_kit_api.ActionParameter{
			Name:         "statusCode",
			Label:        "HTTP Status Code",
			Description:  extutil.Ptr("The HTTP status code returned for aborted requests."),
			Type:         action_kit_api.ActionParameterTypeInteger,
			DefaultValue: extutil.Ptr("500"),
			Required:     extutil.Ptr(true),
			MinValue:     extutil.Ptr(200),
			MaxValue:     extutil.Ptr(600),
		},
		action_kit_api.ActionParameter{
			Name:        "body",
			Label:       "Response Body",
			Description: extutil.Ptr("Optional: overwrite the response body returned to clients. Leave empty to return the default abort response."),
			Type:        action_kit_api.ActionParameterTypeTextarea,
			Required:    extutil.Ptr(false),
		},
		action_kit_api.ActionParameter{
			Name:         "contentType",
			Label:        "Content Type",
			Description:  extutil.Ptr("The Content-Type header set on the overwritten response body."),
			Type:         action_kit_api.ActionParameterTypeString,
			DefaultValue: extutil.Ptr("application/json"),
			Required:     extutil.Ptr(false),
			Advanced:     extutil.Ptr(true),
		},
	)
	return withSectionNameParameter(desc)
}

func buildAbortFaultSpec(config map[string]any) (map[string]any, error) {
	statusCode := extutil.ToInt64(config["statusCode"])
	if statusCode < 200 || statusCode > 600 {
		return nil, fmt.Errorf("statusCode must be between 200 and 600")
	}
	// 418 is reserved as the internal sentinel used to rewrite the response body, so it cannot be
	// used as the client-facing status code.
	if statusCode == sentinelStatus {
		return nil, fmt.Errorf("statusCode %d is reserved for the internal sentinel; choose a different status code", sentinelStatus)
	}
	percentage := percentageFromConfig(config)

	body := extutil.ToString(config["body"])
	if body == "" {
		// Plain abort: return the chosen status with Envoy's default abort body.
		return map[string]any{
			"faultInjection": map[string]any{
				"abort": map[string]any{
					"httpStatus": statusCode,
					"percentage": percentage,
				},
			},
		}, nil
	}

	// Overwrite the response body: abort locally with the sentinel status, then rewrite that
	// (Envoy-generated) response to the configured status/body/content-type. source=Local ensures
	// genuine backend responses are left untouched.
	contentType := extutil.ToString(config["contentType"])
	if contentType == "" {
		contentType = "application/json"
	}
	return map[string]any{
		"faultInjection": map[string]any{
			"abort": map[string]any{
				"httpStatus": sentinelStatus,
				"percentage": percentage,
			},
		},
		"responseOverride": []any{
			map[string]any{
				"match": map[string]any{
					"statusCodes": []any{
						map[string]any{"type": "Value", "value": sentinelStatus},
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
