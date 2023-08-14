// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"os/exec"
	"strings"
)

type DeploymentRolloutRestartAction struct {
}

type DeploymentRolloutRestartState struct {
	Cluster    string `json:"cluster"`
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Wait       bool   `json:"wait"`
}

type DeploymentRolloutRestartConfig struct {
	Wait bool
}

func NewDeploymentRolloutRestartAction() action_kit_sdk.Action[DeploymentRolloutRestartState] {
	return DeploymentRolloutRestartAction{}
}

var _ action_kit_sdk.Action[DeploymentRolloutRestartState] = (*DeploymentRolloutRestartAction)(nil)
var _ action_kit_sdk.ActionWithStatus[DeploymentRolloutRestartState] = (*DeploymentRolloutRestartAction)(nil)

func (f DeploymentRolloutRestartAction) NewEmptyState() DeploymentRolloutRestartState {
	return DeploymentRolloutRestartState{}
}

func (f DeploymentRolloutRestartAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          rolloutRestartActionId,
		Label:       "Rollout Restart Deployment",
		Description: "Execute a rollout restart for a Kubernetes deployment",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(deploymentIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: deploymentTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("state"),
		TimeControl: action_kit_api.TimeControlInternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "wait for rollout completion",
				Name:         "wait",
				Type:         action_kit_api.Boolean,
				Advanced:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("false"),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/deployment/attack/rollout-restart/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/deployment/attack/rollout-restart/start",
		},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			Method: "POST",
			Path:   "/deployment/attack/rollout-restart/status",
		}),
	}
}

func (f DeploymentRolloutRestartAction) Prepare(_ context.Context, state *DeploymentRolloutRestartState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config DeploymentRolloutRestartConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}
	state.Cluster = request.Target.Attributes["k8s.cluster-name"][0]
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.Deployment = request.Target.Attributes["k8s.deployment"][0]
	state.Wait = config.Wait
	return nil, nil
}

func (f DeploymentRolloutRestartAction) Start(_ context.Context, state *DeploymentRolloutRestartState) (*action_kit_api.StartResult, error) {
	log.Info().Msgf("Starting deployment rollout restart attack for %+v", state)

	cmd := exec.Command("kubectl",
		"rollout",
		"restart",
		"--namespace",
		state.Namespace,
		fmt.Sprintf("deployment/%s", state.Deployment))
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to execute rollout restart: %s", cmdOut), cmdErr)
	}

	return nil, nil
}

func (f DeploymentRolloutRestartAction) Status(_ context.Context, state *DeploymentRolloutRestartState) (*action_kit_api.StatusResult, error) {
	if !state.Wait {
		return extutil.Ptr(action_kit_api.StatusResult{
			Completed: true,
		}), nil
	}

	cmd := exec.Command("kubectl",
		"rollout",
		"status",
		"--watch=false",
		"--namespace",
		state.Namespace,
		fmt.Sprintf("deployment/%s", state.Deployment))
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to execute rollout restart status check: %s", cmdOut), cmdErr)
	}

	cmdOutStr := string(cmdOut)
	completed := !strings.Contains(strings.ToLower(cmdOutStr), "waiting")
	return extutil.Ptr(action_kit_api.StatusResult{
		Completed: completed,
	}), nil
}
