/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

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

// Action IDs and constants for NGINX actions
const (
	NginxIngressTargetType       = "com.steadybit.extension_kubernetes.kubernetes-nginx-ingress"
	NginxAnnotationKey           = "nginx.ingress.kubernetes.io/configuration-snippet"
	NginxEnterpriseAnnotationKey = "nginx.org/server-snippets"
	NginxBlockTrafficActionId         = "com.steadybit.extension_kubernetes.nginx-block-traffic"
	NginxDelayTrafficActionId         = "com.steadybit.extension_kubernetes.nginx-delay-traffic"
)

// NginxBaseState contains common state for NGINX-related actions
type NginxBaseState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
}

// prepareNginxAction contains common preparation logic for NGINX actions
func prepareNginxAction(state *NginxBaseState, request action_kit_api.PrepareActionRequestBody) (*networkingv1.Ingress, error) {
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

// startNginxAction contains common start logic for NGINX actions
func startNginxAction(state *NginxBaseState, annotationConfig string, isEnterprise bool) error {
	log.Debug().Msgf("Adding new NGINX configuration: %s", annotationConfig)

	annotationKey := NginxAnnotationKey
	if isEnterprise {
		annotationKey = NginxEnterpriseAnnotationKey
	}
	err := client.K8S.UpdateIngressAnnotation(context.Background(), state.Namespace, state.IngressName, annotationKey, annotationConfig)
	if err != nil {
		return err
	}

	return nil
}

// stopNginxAction contains common stop logic for NGINX actions
func stopNginxAction(state *NginxBaseState, isEnterprise bool) error {
	annotationKey := NginxAnnotationKey
	if isEnterprise {
		annotationKey = NginxEnterpriseAnnotationKey
	}

	err := client.K8S.RemoveAnnotationBlock(
		context.Background(),
		state.Namespace,
		state.IngressName,
		annotationKey,
		state.ExecutionId,
	)
	if err != nil {
		return fmt.Errorf("failed to remove NGINX configuration: %w", err)
	}

	return nil
}

// getNginxActionDescription returns common action description elements
func getNginxActionDescription(id string, label string, description string, icon string) action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          id,
		Label:       label,
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Description: description,
		Technology:  extutil.Ptr("Kubernetes"),
		Icon:        extutil.Ptr(icon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: NginxIngressTargetType,
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

// Helper functions similar to HAProxy implementation
func getNginxStartMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# BEGIN STEADYBIT - %s", executionId)
}

func getNginxEndMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# END STEADYBIT - %s", executionId)
}
