package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

type HAProxyTrafficActionState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
}

func NewDelayTrafficAction() action_kit_sdk.Action[HAProxyTrafficActionState] {
	return &HAProxyTrafficAction{
		actionId:    "com.steadybit.extension_kubernetes.haproxy-delay-traffic",
		label:       "HAProxy Delay Traffic",
		description: "Delay traffic by adding a response delay for requests matching specific paths.",
		delay:       "2s",
	}
}

type HAProxyTrafficAction struct {
	actionId    string
	label       string
	description string
	statusCode  int
	delay       string
}

func (a *HAProxyTrafficAction) NewEmptyState() HAProxyTrafficActionState {
	return HAProxyTrafficActionState{}
}

func (a *HAProxyTrafficAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          a.actionId,
		Label:       a.label,
		Description: a.description,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,..."), // Add appropriate icon
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: HAProxyIngressTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "HAProxy Ingress",
					Description: extutil.Ptr("Select an HAProxy ingress by namespace and name."),
					Query:       "k8s.namespace=\"\" AND k8s.ingress=\"\"",
				},
			}),
		}),
		Kind: action_kit_api.Attack,
	}
}

func (a *HAProxyTrafficAction) Prepare(_ context.Context, state *HAProxyTrafficActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]
	return nil, nil
}

func (a *HAProxyTrafficAction) Start(_ context.Context, state *HAProxyTrafficActionState) (*action_kit_api.StartResult, error) {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	annotationKey := "haproxy-ingress.github.io/config-backend"
	existingConfig := ingress.Annotations[annotationKey]
	newConfig := a.buildConfig(state.ExecutionId)

	// Prepend the new configuration
	updatedConfig := newConfig + "\n" + existingConfig
	ingress.Annotations[annotationKey] = updatedConfig

	err = updateIngress(state.Namespace, state.IngressName, annotationKey, ingress)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *HAProxyTrafficAction) Stop(_ context.Context, state *HAProxyTrafficActionState) (*action_kit_api.StopResult, error) {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	annotationKey := "haproxy-ingress.github.io/config-backend"
	existingConfig := ingress.Annotations[annotationKey]

	// Remove the configuration block for this execution
	startMarker := fmt.Sprintf("# BEGIN STEADYBIT - %s", state.ExecutionId)
	endMarker := fmt.Sprintf("# END STEADYBIT - %s", state.ExecutionId)
	updatedConfig := removeConfigBlock(existingConfig, startMarker, endMarker)

	ingress.Annotations[annotationKey] = updatedConfig
	err = updateIngress(state.Namespace, state.IngressName, annotationKey, ingress)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *HAProxyTrafficAction) buildConfig(executionId uuid.UUID) string {
	var config strings.Builder
	config.WriteString(fmt.Sprintf("# BEGIN STEADYBIT - %s\n", executionId))
	if a.delay != "" {
		config.WriteString(fmt.Sprintf("tcp-request inspect-delay %s\n", a.delay))
		config.WriteString("tcp-request content accept if WAIT_END || !{ path /delay }\n")
	}
	if a.statusCode != 0 {
		config.WriteString(fmt.Sprintf("http-request return status %d if { path /inject }\n", a.statusCode))
	}
	config.WriteString(fmt.Sprintf("# END STEADYBIT - %s", executionId))
	return config.String()
}
