package extingress

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

// HAProxyBaseState contains common state for HAProxy-related actions
type HAProxyBaseState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
}

// prepareHAProxyAction contains common preparation logic for HAProxy actions
func prepareHAProxyAction(state *HAProxyBaseState, request action_kit_api.PrepareActionRequestBody) error {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]

	// Check ingress availability
	_, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return fmt.Errorf("failed to fetch ingress: %w", err)
	}
	return nil
}

// startHAProxyAction contains common start logic for HAProxy actions
func startHAProxyAction(ctx context.Context, state *HAProxyBaseState, configGenerator func() string) error {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingConfig := ingress.Annotations[AnnotationKey]

	// Get the new configuration from the provided generator function
	newConfig := configGenerator()

	// Prepend the new configuration
	ingress.Annotations[AnnotationKey] = newConfig + "\n" + existingConfig

	err = updateIngress(state.Namespace, state.IngressName, AnnotationKey, ingress)
	if err != nil {
		return err
	}

	return nil
}

// stopHAProxyAction contains common stop logic for HAProxy actions
func stopHAProxyAction(ctx context.Context, state *HAProxyBaseState) error {
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName)
	if err != nil {
		return fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingConfig := ingress.Annotations[AnnotationKey]

	// Remove the configuration block for this execution
	updatedConfig := removeConfigBlock(existingConfig, getStartMarker(state.ExecutionId), getEndMarker(state.ExecutionId))

	ingress.Annotations[AnnotationKey] = updatedConfig
	err = updateIngress(state.Namespace, state.IngressName, AnnotationKey, ingress)
	if err != nil {
		return err
	}

	return nil
}

// getCommonActionDescription returns common action description elements
func getCommonActionDescription(id string, label string, description string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9IiMyMjIiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIj4KICA8Y2lyY2xlIGN4PSIxMiIgY3k9IjEyIiByPSIxMCIvPgogIDxsaW5lIHgxPSI3IiB5MT0iNyIgeDI9IjE3IiB5Mj0iMTciLz4KPC9zdmc+"),
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
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
		},
	}
}
