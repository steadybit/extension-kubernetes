package extingress

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
	"strconv"
	"strings"
)

// HAProxyBlockTrafficState extends base state with block-specific fields
type HAProxyBlockTrafficState struct {
	HAProxyBaseState
	PathStatusCode map[string]int
}

func NewBlockTrafficAction() action_kit_sdk.Action[HAProxyBlockTrafficState] {
	return &HAProxyBlockTrafficAction{}
}

type HAProxyBlockTrafficAction struct{}

func (a *HAProxyBlockTrafficAction) NewEmptyState() HAProxyBlockTrafficState {
	return HAProxyBlockTrafficState{}
}

func (a *HAProxyBlockTrafficAction) Describe() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		HAProxyBlockTrafficActionId,
		"HAProxy Block Traffic",
		"Block traffic by returning a custom HTTP status code for requests matching specific paths.")

	// Add block-specific parameter
	desc.Parameters = append(desc.Parameters,
		action_kit_api.ActionParameter{
			Name:         "pathStatusCode",
			Label:        "Path and Statuscode",
			Description:  extutil.Ptr("Key is the path, value is the status code. Example: /block=503"),
			Type:         action_kit_api.KeyValue,
			DefaultValue: extutil.Ptr("[{\"key\":\"/\", \"value\":\"503\"}]"),
			Required:     extutil.Ptr(true),
		},
		// ToDo: Add optional delay parameter
	)

	return desc
}

func (a *HAProxyBlockTrafficAction) Prepare(ctx context.Context, state *HAProxyBlockTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	// Use common preparation logic first
	if err := prepareHAProxyAction(&state.HAProxyBaseState, request); err != nil {
		return nil, err
	}

	// Handle block-specific configuration
	var err error
	if request.Config["pathStatusCode"] != nil {
		pathStatusCode := make(map[string]string)
		pathStatusCode, err = extutil.ToKeyValue(request.Config, "pathStatusCode")
		if err != nil {
			return nil, err
		}
		// check status codes
		state.PathStatusCode = make(map[string]int)
		for path, statusCodeStr := range pathStatusCode {
			var statusCode int
			if statusCode, err = strconv.Atoi(statusCodeStr); err != nil {
				return nil, fmt.Errorf("invalid status code: %s", statusCodeStr)
			}
			//append to map
			state.PathStatusCode[path] = statusCode
		}

		//ToDo: Check if annoation for delay already exists
	}

	return nil, nil
}

func (a *HAProxyBlockTrafficAction) Start(ctx context.Context, state *HAProxyBlockTrafficState) (*action_kit_api.StartResult, error) {
	configGenerator := func() string {
		var configBuilder strings.Builder
		configBuilder.WriteString(getStartMarker(state.ExecutionId) + "\n")
		for path, statusCode := range state.PathStatusCode {
			configBuilder.WriteString(fmt.Sprintf("http-request return status %d if { path %s }\n", statusCode, path))
		}
		configBuilder.WriteString(getEndMarker(state.ExecutionId) + "\n")
		return configBuilder.String()
	}

	if err := startHAProxyAction(&state.HAProxyBaseState, configGenerator); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *HAProxyBlockTrafficAction) Stop(_ context.Context, state *HAProxyBlockTrafficState) (*action_kit_api.StopResult, error) {
	if err := stopHAProxyAction(&state.HAProxyBaseState); err != nil {
		return nil, err
	}

	return nil, nil
}
