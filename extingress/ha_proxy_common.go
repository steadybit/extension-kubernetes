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

// AnnotationKey is the name of the HAProxy configuration annotation
const (
	AnnotationKey              = "haproxy-ingress.github.io/config-snippet"
	HAProxyBlockTrafficActionId = "com.steadybit.extension_kubernetes.haproxy-block-traffic"
	HAProxyDelayTrafficActionId = "com.steadybit.extension_kubernetes.haproxy-delay-traffic"
	HAProxyIngressTargetType    = "com.steadybit.extension_kubernetes.haproxy-ingress"
)

// HAProxyBaseState contains common state data for HAProxy-related actions
type HAProxyBaseState struct {
	ExecutionId uuid.UUID
	Namespace   string
	IngressName string
}

// prepareHAProxyAction performs common preparation logic for HAProxy actions
func prepareHAProxyAction(state *HAProxyBaseState, request action_kit_api.PrepareActionRequestBody) (*networkingv1.Ingress, error) {
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]

	// Fetch and validate the ingress resource
	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress %s/%s: %w", state.Namespace, state.IngressName, err)
	}

	return ingress, nil
}

// startHAProxyAction applies HAProxy configuration via ingress annotation
func startHAProxyAction(state *HAProxyBaseState, annotationConfig string) error {
	log.Debug().
		Str("namespace", state.Namespace).
		Str("ingress", state.IngressName).
		Str("executionId", state.ExecutionId.String()).
		Msgf("Adding new HAProxy configuration")

	if err := client.K8S.UpdateIngressAnnotation(
		context.Background(),
		state.Namespace,
		state.IngressName,
		AnnotationKey,
		annotationConfig,
	); err != nil {
		return fmt.Errorf("failed to update ingress annotation: %w", err)
	}

	return nil
}

// stopHAProxyAction removes HAProxy configuration from ingress annotation
func stopHAProxyAction(state *HAProxyBaseState) error {
	log.Debug().
		Str("namespace", state.Namespace).
		Str("ingress", state.IngressName).
		Str("executionId", state.ExecutionId.String()).
		Msgf("Removing HAProxy configuration")

	if err := client.K8S.RemoveAnnotationBlock(
		context.Background(),
		state.Namespace,
		state.IngressName,
		AnnotationKey,
		state.ExecutionId,
	); err != nil {
		return fmt.Errorf("failed to remove HAProxy configuration: %w", err)
	}

	return nil
}

// getStartMarker returns the configuration block start marker
func getStartMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# BEGIN STEADYBIT %s", executionId)
}

// getEndMarker returns the configuration block end marker
func getEndMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# END STEADYBIT %s", executionId)
}

// getCommonActionDescription returns common action description properties
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

// getConditionsParameters returns the common condition parameters for HAProxy actions
func getConditionsParameters() []action_kit_api.ActionParameter {
	return []action_kit_api.ActionParameter{
		{
			Name:  "-conditions-separator-",
			Label: "-",
			Type:  action_kit_api.ActionParameterTypeSeparator,
		},
		{
			Name:  "-conditions-header-",
			Type:  action_kit_api.ActionParameterTypeHeader,
			Label: "Conditions",
		},
		{
			Name:        "conditionPathPattern",
			Label:       "Path Pattern",
			Description: extutil.Ptr("The path patterns to compare against the request URL."),
			Type:        action_kit_api.ActionParameterTypeRegex,
			Required:    extutil.Ptr(false),
		},
		{
			Name:         "conditionHttpMethod",
			Label:        "HTTP Method",
			Description:  extutil.Ptr("The name of the request method."),
			Type:         action_kit_api.ActionParameterTypeString,
			DefaultValue: extutil.Ptr("*"),
			Required:     extutil.Ptr(false),
			Options: extutil.Ptr([]action_kit_api.ParameterOption{
				action_kit_api.ExplicitParameterOption{
					Label: "*",
					Value: "*",
				},
				action_kit_api.ExplicitParameterOption{
					Label: "GET",
					Value: "GET",
				},
				action_kit_api.ExplicitParameterOption{
					Label: "POST",
					Value: "POST",
				},
				action_kit_api.ExplicitParameterOption{
					Label: "PUT",
					Value: "PUT",
				},
				action_kit_api.ExplicitParameterOption{
					Label: "PATCH",
					Value: "PATCH",
				},
				action_kit_api.ExplicitParameterOption{
					Label: "HEAD",
					Value: "HEAD",
				},
				action_kit_api.ExplicitParameterOption{
					Label: "DELETE",
					Value: "DELETE",
				},
			}),
		},
		{
			Name:        "conditionHttpHeader",
			Label:       "HTTP Header",
			Description: extutil.Ptr("The name of the HTTP header field with a maximum size of 40 characters. And a value to compare against the value of the HTTP header. The maximum size of each string is 128 characters. The comparison strings are case insensitive. The following wildcard characters are supported: * (matches 0 or more characters) and ? (matches exactly 1 character). Currently only a single header name with a single value is allowed."),
			Type:        action_kit_api.ActionParameterTypeKeyValue,
			Required:    extutil.Ptr(false),
		},
	}
}
