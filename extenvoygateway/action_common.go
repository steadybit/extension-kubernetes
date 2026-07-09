// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extenvoygateway

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ActionState is shared by all three Envoy Gateway HTTPRoute attacks.
type ActionState struct {
	Namespace   string         `json:"namespace"`
	RouteName   string         `json:"routeName"`
	SectionName string         `json:"sectionName"`
	PolicyName  string         `json:"policyName"`
	ExecutionId string         `json:"executionId"`
	FaultSpec   map[string]any `json:"faultSpec"`
}

// backendTrafficPolicyAction is the common attack implementation. Each attack supplies a description
// and a buildFaultSpecFn that produces the faultInjection/responseOverride portion of the BTP spec.
type backendTrafficPolicyAction struct {
	k8s              *client.Client
	description      action_kit_api.ActionDescription
	subtype          string
	buildFaultSpecFn func(config map[string]any) (map[string]any, error)
}

func (a *backendTrafficPolicyAction) NewEmptyState() ActionState {
	return ActionState{}
}

func (a *backendTrafficPolicyAction) Describe() action_kit_api.ActionDescription {
	return a.description
}

func (a *backendTrafficPolicyAction) Prepare(ctx context.Context, state *ActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	namespace := request.Target.Attributes["k8s.namespace"]
	routeName := request.Target.Attributes[attrHttpRoute]
	if len(namespace) == 0 || len(routeName) == 0 {
		return nil, extension_kit.ToError("Missing required target attributes k8s.namespace and/or k8s.envoy-gateway.http-route.", nil)
	}

	state.Namespace = namespace[0]
	state.RouteName = routeName[0]
	state.SectionName = extutil.ToString(request.Config["sectionName"])
	state.ExecutionId = request.ExecutionId.String()
	state.PolicyName = fmt.Sprintf("steadybit-%s-%s", a.subtype, request.ExecutionId.String())

	faultSpec, err := a.buildFaultSpecFn(request.Config)
	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to build fault configuration: %v", err), err)
	}
	state.FaultSpec = faultSpec

	if err := a.checkConflict(ctx, state); err != nil {
		return nil, err
	}

	return nil, nil
}

func (a *backendTrafficPolicyAction) Start(ctx context.Context, state *ActionState) (*action_kit_api.StartResult, error) {
	// Re-check immediately before create to narrow the Prepare→Start window and catch a concurrent
	// attack that already created a policy on the same route. This is best-effort: List→Create is not
	// atomic, so two executions started at the same instant can still both create a policy, in which
	// case Envoy Gateway resolves the conflict oldest-wins. Fully closing the window would require
	// server-side admission control, which is out of scope.
	if err := a.checkConflict(ctx, state); err != nil {
		return nil, err
	}

	policy := buildBackendTrafficPolicy(state.Namespace, state.PolicyName, state.ExecutionId, state.RouteName, state.SectionName, state.FaultSpec)
	_, err := a.k8s.DynamicClient().Resource(client.BackendTrafficPolicyGVR).Namespace(state.Namespace).Create(ctx, policy, metav1.CreateOptions{})
	if err != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to create BackendTrafficPolicy %s/%s: %v", state.Namespace, state.PolicyName, err), err)
	}

	log.Info().Msgf("Created BackendTrafficPolicy %s/%s targeting HTTPRoute %s", state.Namespace, state.PolicyName, state.RouteName)
	return &action_kit_api.StartResult{
		Messages: new([]action_kit_api.Message{
			{
				Level:   extutil.Ptr(action_kit_api.Info),
				Message: fmt.Sprintf("Applied BackendTrafficPolicy %s to HTTPRoute %s/%s", state.PolicyName, state.Namespace, state.RouteName),
			},
		}),
	}, nil
}

func (a *backendTrafficPolicyAction) Stop(ctx context.Context, state *ActionState) (*action_kit_api.StopResult, error) {
	err := a.k8s.DynamicClient().Resource(client.BackendTrafficPolicyGVR).Namespace(state.Namespace).Delete(ctx, state.PolicyName, metav1.DeleteOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to delete BackendTrafficPolicy %s/%s: %v", state.Namespace, state.PolicyName, err), err)
	}

	log.Info().Msgf("Removed BackendTrafficPolicy %s/%s", state.Namespace, state.PolicyName)
	return nil, nil
}

func (a *backendTrafficPolicyAction) checkConflict(ctx context.Context, state *ActionState) error {
	list, err := a.k8s.DynamicClient().Resource(client.BackendTrafficPolicyGVR).Namespace(state.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return extension_kit.ToError(fmt.Sprintf("Failed to list existing BackendTrafficPolicies in namespace %s: %v", state.Namespace, err), err)
	}
	if conflict := findConflictingPolicy(list.Items, state.RouteName, state.SectionName, state.PolicyName); conflict != "" {
		return extension_kit.ToError(fmt.Sprintf(
			"An existing BackendTrafficPolicy %q already targets HTTPRoute %s/%s. Envoy Gateway resolves conflicts oldest-wins, so this attack would have no effect. Remove the existing policy or target a different route.",
			conflict, state.Namespace, state.RouteName), nil)
	}
	return nil
}

// percentageFromConfig extracts the traffic percentage as a float (Envoy Gateway's percentage field
// is a float32 with 0.0001% accuracy). Defaults to 100 when unset.
func percentageFromConfig(config map[string]any) float64 {
	switch v := config["percentage"].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case nil:
		return 100
	default:
		return float64(extutil.ToInt64(v))
	}
}

// getCommonActionDescription returns the base action description with duration, percentage and the
// advanced sectionName parameter, plus the HTTPRoute target selection.
func getCommonActionDescription(id, label, description string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  new("Kubernetes"),
		Icon:        new(EnvoyGatewayIcon),
		TargetSelection: new(action_kit_api.TargetSelection{
			TargetType: EnvoyGatewayHttpRouteTargetType,
			SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "HTTP route",
					Description: new("Find HTTP route by cluster, namespace and route name"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.envoy-gateway.http-route=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  new("The duration of the attack. The HTTP route will be affected for the specified duration."),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("30s"),
				Required:     new(true),
			},
			{
				Name:         "percentage",
				Label:        "Traffic Percentage",
				Description:  new("The percentage of requests the fault is applied to."),
				Type:         action_kit_api.ActionParameterTypePercentage,
				DefaultValue: new("50"),
				Required:     new(true),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Stop:    new(action_kit_api.MutatingEndpointReference{}),
	}
}

// withSectionNameParameter appends the advanced sectionName parameter. Call it after the attack's own
// parameters so the advanced parameter renders last.
func withSectionNameParameter(desc action_kit_api.ActionDescription) action_kit_api.ActionDescription {
	desc.Parameters = append(desc.Parameters, action_kit_api.ActionParameter{
		Name:        "sectionName",
		Label:       "Route Rule Name",
		Description: new("Optional: restrict the attack to a single named route rule (spec.rules[].name) instead of the whole route."),
		Type:        action_kit_api.ActionParameterTypeString,
		Required:    new(false),
		Advanced:    new(true),
	})
	return desc
}
