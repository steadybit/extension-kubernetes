// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
)

// Action IDs for HAProxy actions
const (
	HAProxyIngressTargetType    = "com.steadybit.extension_kubernetes.kubernetes-haproxy-ingress"
	HAProxyBlockTrafficActionId = "com.steadybit.extension_kubernetes.haproxy-block-traffic"
	HAProxyDelayTrafficActionId = "com.steadybit.extension_kubernetes.haproxy-delay-traffic"
	haProxyAnnotationKey        = "haproxy.org/backend-config-snippet"
)

// HAProxyState contains common state for HAProxy-related actions
type HAProxyState struct {
	ExecutionId      uuid.UUID
	Namespace        string
	IngressName      string
	AnnotationKey    string
	Matcher          RequestMatcher
	AnnotationConfig string
}

type haProxyAction struct {
	description        action_kit_api.ActionDescription
	annotationConfigFn func(state *HAProxyState, config map[string]interface{}) string
	checkExistingFn    func(lines []string) error
}

// NewEmptyState creates an empty state object
func (a *haProxyAction) NewEmptyState() HAProxyState {
	return HAProxyState{}
}

func (a *haProxyAction) Describe() action_kit_api.ActionDescription {
	return a.description
}

// Prepare validates input parameters and prepares the state for execution
func (a *haProxyAction) Prepare(_ context.Context, state *HAProxyState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var err error
	state.ExecutionId = request.ExecutionId
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.IngressName = request.Target.Attributes["k8s.ingress"][0]

	state.AnnotationKey = haProxyAnnotationKey

	state.Matcher, err = parseRequestMatcher(request.Config)
	if err != nil {
		return nil, err
	}

	ingress, err := client.K8S.IngressByNamespaceAndName(state.Namespace, state.IngressName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingress: %w", err)
	}

	existingSnippet := strings.Split(ingress.Annotations[state.AnnotationKey], "\n")
	if err = checkHAProxyRuleConflicts(existingSnippet, state.Matcher); err != nil {
		return nil, err
	}

	if a.checkExistingFn != nil {
		if err = a.checkExistingFn(existingSnippet); err != nil {
			return nil, err
		}
	}

	state.AnnotationConfig = a.annotationConfigFn(state, request.Config)
	return nil, nil
}

// Start applies the HAProxy configuration to begin blocking traffic
func (a *haProxyAction) Start(_ context.Context, state *HAProxyState) (*action_kit_api.StartResult, error) {
	log.Debug().Msgf("Adding new %s configuration: %s", a.description.Label, state.AnnotationConfig)

	err := client.K8S.UpdateIngressAnnotation(context.Background(), state.Namespace, state.IngressName, state.AnnotationKey, state.AnnotationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start %s action: %w", a.description.Label, err)
	}
	return nil, nil
}

// Stop removes the HAProxy configuration to stop blocking traffic
func (a *haProxyAction) Stop(_ context.Context, state *HAProxyState) (*action_kit_api.StopResult, error) {
	err := client.K8S.RemoveAnnotationBlock(
		context.Background(),
		state.Namespace,
		state.IngressName,
		state.AnnotationKey,
		state.ExecutionId,
		getHAProxyStartMarker(state.ExecutionId),
		getHAProxyEndMarker(state.ExecutionId),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to stop %s action: %w", a.description.Label, err)
	}

	return nil, nil
}

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
			Type:        action_kit_api.ActionParameterTypeKeyValue,
			Description: extutil.Ptr("The name of the HTTP header field . And a value to compare against the value of the HTTP header as a regular expression."),
			Required:    extutil.Ptr(false),
		},
	}
}

func getHAProxyStartMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# BEGIN STEADYBIT - %s\n", executionId)
}

func getHAProxyEndMarker(executionId uuid.UUID) string {
	return fmt.Sprintf("# END STEADYBIT - %s\n", executionId)
}

// checkHAProxyRuleConflicts checks if the new rules would conflict with existing ones
func checkHAProxyRuleConflicts(lines []string, matcher RequestMatcher) error {
	if matcher.PathPattern == "" {
		return nil
	}

	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("path_reg %s", matcher.PathPattern)) {
			return fmt.Errorf("a rule for path %s already exists", matcher.PathPattern)
		}
	}
	return nil
}
