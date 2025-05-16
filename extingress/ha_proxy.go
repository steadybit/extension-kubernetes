package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	networkingv1 "k8s.io/api/networking/v1"
)

// Action IDs for HAProxy actions
const (
	HAProxyBlockTrafficActionId = "com.steadybit.extension_kubernetes.haproxy-block-traffic"
	HAProxyDelayTrafficActionId = "com.steadybit.extension_kubernetes.haproxy-delay-traffic"
)

// HAProxyBaseState contains common state for HAProxy-related actions
type HAProxyBaseState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
}

// prepareHAProxyAction contains common preparation logic for HAProxy actions
func prepareHAProxyAction(state *HAProxyBaseState, request action_kit_api.PrepareActionRequestBody) (*networkingv1.Ingress, error) {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]

	// Check ingress availability
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}
	return ingress, nil
}

// startHAProxyAction contains common start logic for HAProxy actions
func startHAProxyAction(state *HAProxyBaseState, annotationConfig string) error {
	log.Debug().Msgf("Adding new HAProxy configuration: %s", annotationConfig)
	// Prepend the new configuration
	err := client.K8S.UpdateIngressAnnotation(context.Background(), state.Namespace, state.IngressName, AnnotationKey, annotationConfig)
	if err != nil {
		return err
	}

	return nil
}

// stopHAProxyAction contains common stop logic for HAProxy actions
func stopHAProxyAction(state *HAProxyBaseState) error {
	err := client.K8S.RemoveAnnotationBlock(
		context.Background(),
		state.Namespace,
		state.IngressName,
		AnnotationKey,
		state.ExecutionId,
	)
	if err != nil {
		return fmt.Errorf("failed to remove HAProxy configuration: %w", err)
	}

	return nil
}

// getCommonActionDescription returns common action description elements
func getCommonActionDescription(id string, label string, description string, icon string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr(icon),
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
				Description:  extutil.Ptr("The duration of the action. The ingress will be affected for the specified duration."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
		},
	}
}
