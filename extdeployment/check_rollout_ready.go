// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"os/exec"
	"strings"
	"time"
)

type CheckDeploymentRolloutStatusAction struct {
}

type CheckDeploymentRolloutStatusState struct {
	Cluster    string `json:"cluster"`
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	TimeoutEnd *int64 `json:"timeoutEnd"`
}

type CheckDeploymentRolloutConfig struct {
	Duration int
}

func NewCheckDeploymentRolloutStatusAction() action_kit_sdk.Action[CheckDeploymentRolloutStatusState] {
	return CheckDeploymentRolloutStatusAction{}
}

var _ action_kit_sdk.Action[CheckDeploymentRolloutStatusState] = (*CheckDeploymentRolloutStatusAction)(nil)
var _ action_kit_sdk.ActionWithStatus[CheckDeploymentRolloutStatusState] = (*CheckDeploymentRolloutStatusAction)(nil)

func (f CheckDeploymentRolloutStatusAction) NewEmptyState() CheckDeploymentRolloutStatusState {
	return CheckDeploymentRolloutStatusState{}
}

func (f CheckDeploymentRolloutStatusAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          RolloutStatusActionId,
		Label:       "Deployment Rollout Status",
		Description: "Check the rollout status of the deployment. The check succeeds when no rollout is pending, i.e., `kubectl rollout status` exits with status code `0`.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(deploymentIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: DeploymentTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("Kubernetes"),
		TimeControl: action_kit_api.TimeControlInternal,
		Kind:        action_kit_api.Check,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Timeout",
				Description:  extutil.Ptr("Maximum time to wait for the rollout to be rolled out completely."),
				Name:         "duration",
				Type:         action_kit_api.Duration,
				Advanced:     extutil.Ptr(false),
				DefaultValue: extutil.Ptr("10m"),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{}),
	}
}

func (f CheckDeploymentRolloutStatusAction) Prepare(_ context.Context, state *CheckDeploymentRolloutStatusState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	var config CheckDeploymentRolloutConfig
	if err := extconversion.Convert(request.Config, &config); err != nil {
		return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
	}

	var timeoutEnd *int64
	if config.Duration != 0 {
		timeoutEnd = extutil.Ptr(time.Now().Add(time.Duration(int(time.Millisecond) * config.Duration)).Unix())
	}
	state.Cluster = request.Target.Attributes["k8s.cluster-name"][0]
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.Deployment = request.Target.Attributes["k8s.deployment"][0]
	state.TimeoutEnd = timeoutEnd
	return nil, nil
}

func (f CheckDeploymentRolloutStatusAction) Start(_ context.Context, _ *CheckDeploymentRolloutStatusState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (f CheckDeploymentRolloutStatusAction) Status(_ context.Context, state *CheckDeploymentRolloutStatusState) (*action_kit_api.StatusResult, error) {
	if state.TimeoutEnd != nil && time.Now().After(time.Unix(*state.TimeoutEnd, 0)) {
		return extutil.Ptr(action_kit_api.StatusResult{
			Completed: true,
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("Timed out waiting for deployment '%s' in namespace '%s' to complete rollout", state.Deployment, state.Namespace),
				Status: extutil.Ptr(action_kit_api.Failed),
			}),
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
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to execute rollout status check: %s", cmdOut), cmdErr)
	}

	cmdOutStr := string(cmdOut)
	completed := !strings.Contains(strings.ToLower(cmdOutStr), "waiting")
	return extutil.Ptr(action_kit_api.StatusResult{
		Completed: completed,
	}), nil
}
