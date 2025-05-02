package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"strconv"
	"strings"
)

type HAProxyDelayTrafficState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
	PathDelay   map[string]int
}

func NewDelayTrafficAction() action_kit_sdk.Action[HAProxyDelayTrafficState] {
	return &HAProxyDelayTrafficAction{}
}

type HAProxyDelayTrafficAction struct{}

func (a *HAProxyDelayTrafficAction) NewEmptyState() HAProxyDelayTrafficState {
	return HAProxyDelayTrafficState{}
}

func (a *HAProxyDelayTrafficAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          "com.steadybit.extension_kubernetes.haproxy-delay-traffic",
		Label:       "HAProxy Delay Traffic",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: "Delay traffic by adding a response delay for requests matching specific paths.",
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiMyMjIiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIj48Y2lyY2xlIGN4PSIxMiIgY3k9IjEyIiByPSI5Ii8+PHBvbHlsaW5lIHBvaW50cz0iMTIgNyAxMiAxMiAxNiAxNCIvPjxsaW5lIHgxPSI0IiB5MT0iMjAiIHgyPSIyMCIgeTI9IjIwIiBzdHJva2UtZGFzaGFycmF5PSIzLDIiLz48L3N2Zz4="),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: HAProxyIngressTargetType,
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
				Description:  extutil.Ptr("The duration of the action. The ingress will block traffic for the specified duration."),
				Name:         "duration",
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "pathDelay",
				Label:        "Path and Delay in seconds",
				Description:  extutil.Ptr("Key is the path, value is the delay in seconds. Example: /delay=3"),
				Type:         action_kit_api.KeyValue,
				DefaultValue: extutil.Ptr("[{\"key\":\"/\", \"value\":\"3\"}]"),
				Required:     extutil.Ptr(true),
			},
		},
	}
}

func (a *HAProxyDelayTrafficAction) Prepare(_ context.Context, state *HAProxyDelayTrafficState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]
	var err error
	if (request.Config["pathDelay"]) != nil {
		pathStatusCode := make(map[string]string)
		pathStatusCode, err = extutil.ToKeyValue(request.Config, "pathDelay")
		if err != nil {
			return nil, err
		}
		// check status codes
		state.PathDelay = make(map[string]int)
		for path, statusCodeStr := range pathStatusCode {
			var statusCode int
			if statusCode, err = strconv.Atoi(statusCodeStr); err != nil {
				return nil, fmt.Errorf("invalid status code: %s", statusCodeStr)
			}
			//append to map
			state.PathDelay[path] = statusCode
		}
	}
	//Check ingress availability
	_, err = client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}
	return nil, nil
}

func (a *HAProxyDelayTrafficAction) Start(_ context.Context, state *HAProxyDelayTrafficState) (*action_kit_api.StartResult, error) {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingConfig := ingress.Annotations[AnnotationKey]

	// Build the new configuration dynamically using the state.PathStatusCode map
	var configBuilder strings.Builder
	configBuilder.WriteString(getStartMarker(state.ExecutionId) + "\n")
	for path, delay := range state.PathDelay {
		configBuilder.WriteString(fmt.Sprintf("tcp-request inspect-delay %ds\n", delay))
		configBuilder.WriteString(fmt.Sprintf("tcp-request content accept if WAIT_END || !{ path %s }\n", path))
	}
	configBuilder.WriteString(getEndMarker(state.ExecutionId) + "\n")

	newConfig := configBuilder.String()

	// Prepend the new configuration
	ingress.Annotations[AnnotationKey] = newConfig + "\n" + existingConfig

	err = updateIngress(state.Namespace, state.IngressName, AnnotationKey, ingress)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *HAProxyDelayTrafficAction) Stop(_ context.Context, state *HAProxyDelayTrafficState) (*action_kit_api.StopResult, error) {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingConfig := ingress.Annotations[AnnotationKey]

	// Remove the configuration block for this execution
	updatedConfig := removeConfigBlock(existingConfig, getStartMarker(state.ExecutionId), getEndMarker(state.ExecutionId))

	ingress.Annotations[AnnotationKey] = updatedConfig
	err = updateIngress(state.Namespace, state.IngressName, AnnotationKey, ingress)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
