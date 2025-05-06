package extingress

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extutil"
	"strings"
)

// HAProxyDelayTrafficState extends base state with delay-specific fields
type HAProxyDelayTrafficState struct {
	HAProxyBaseState
	Path  string
	Delay int
}

func NewDelayTrafficAction() action_kit_sdk.Action[HAProxyDelayTrafficState] {
	return &HAProxyDelayTrafficAction{}
}

type HAProxyDelayTrafficAction struct{}

func (a *HAProxyDelayTrafficAction) NewEmptyState() HAProxyDelayTrafficState {
	return HAProxyDelayTrafficState{}
}

func (a *HAProxyDelayTrafficAction) Describe() action_kit_api.ActionDescription {
	desc := getCommonActionDescription(
		HAProxyDelayTrafficActionId,
		"HAProxy Delay Traffic",
		"Delay traffic by adding a response delay for requests matching specific paths.")

	// Override icon for delay action
	desc.Icon = extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiMyMjIiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIj48Y2lyY2xlIGN4PSIxMiIgY3k9IjEyIiByPSI5Ii8+PHBvbHlsaW5lIHBvaW50cz0iMTIgNyAxMiAxMiAxNiAxNCIvPjxsaW5lIHgxPSI0IiB5MT0iMjAiIHgyPSIyMCIgeTI9IjIwIiBzdHJva2UtZGFzaGFycmF5PSIzLDIiLz48L3N2Zz4=")

	// Add delay-specific parameters
	desc.Parameters = append(desc.Parameters,
		action_kit_api.ActionParameter{
			Name:         "path",
			Label:        "Path to be delayed",
			Description:  extutil.Ptr("The path to be delayed. Example: /delay"),
			Type:         action_kit_api.String,
			DefaultValue: extutil.Ptr("/"),
			Required:     extutil.Ptr(true),
		},
		action_kit_api.ActionParameter{
			Name:         "delay",
			Label:        "Delay",
			Description:  extutil.Ptr("The delay in for the path. Example: 5s"),
			Type:         action_kit_api.Duration,
			DefaultValue: extutil.Ptr("3s"),
			Required:     extutil.Ptr(true),
		},
	)

	return desc
}

func (a *HAProxyDelayTrafficAction) Prepare(ctx context.Context, state *HAProxyDelayTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	// Use common preparation logic first
	if err := prepareHAProxyAction(&state.HAProxyBaseState, request); err != nil {
		return nil, err
	}

	// Store values in exported fields so they're persisted
	if path, ok := request.Config["path"]; ok {
		if pathStr, isStr := path.(string); isStr {
			state.Path = pathStr
		} else {
			return nil, fmt.Errorf("path must be a string")
		}
	} else {
		return nil, fmt.Errorf("path is required")
	}

	if delay, ok := request.Config["delay"]; ok {
		switch v := delay.(type) {
		case float64:
			state.Delay = int(v)
		case int:
			state.Delay = v
		case string:
			// Try to parse string as number if needed
			return nil, fmt.Errorf("delay must be a number, got string: %s", v)
		default:
			return nil, fmt.Errorf("delay must be a number")
		}
	} else {
		return nil, fmt.Errorf("delay is required")
	}
	//ToDo: Check if annoation for delay already exists
	return nil, nil
}

func (a *HAProxyDelayTrafficAction) Start(ctx context.Context, state *HAProxyDelayTrafficState) (*action_kit_api.StartResult, error) {
	configGenerator := func() string {
		var configBuilder strings.Builder
		configBuilder.WriteString(getStartMarker(state.ExecutionId) + "\n")
		configBuilder.WriteString(fmt.Sprintf("tcp-request inspect-delay %dms\n", state.Delay))
		configBuilder.WriteString(fmt.Sprintf("tcp-request content accept if WAIT_END || !{ path %s }\n", state.Path))
		configBuilder.WriteString(getEndMarker(state.ExecutionId) + "\n")
		return configBuilder.String()
	}

	if err := startHAProxyAction(&state.HAProxyBaseState, configGenerator); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *HAProxyDelayTrafficAction) Stop(_ context.Context, state *HAProxyDelayTrafficState) (*action_kit_api.StopResult, error) {
	if err := stopHAProxyAction(&state.HAProxyBaseState); err != nil {
		return nil, err
	}

	return nil, nil
}
