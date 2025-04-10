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
		Id:          RolloutRestartActionId,
		Label:       "Rollout Restart Deployment",
		Description: "Execute a rollout restart for a Kubernetes deployment",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTExLjM5IDE5LjkzTDEwLjgxIDIwLjJDMTAuMzYgMjAuNDIgOS44NCAyMC40MSA5LjM5IDIwLjE5TDMuNSAxNy4yNEMyLjk1IDE2Ljk2IDIuNiAxNi40MSAyLjYgMTUuNzlWOC4xOTAwMUMyLjYgNy41NzAwMSAyLjk0IDcuMDIwMDEgMy40OSA2Ljc0MDAxTDkuMzggMy43OTAwMUM5LjgzIDMuNTcwMDEgMTAuMzUgMy41NjAwMSAxMC44IDMuNzcwMDFMMTcuMDcgNi43NTAwMUMxNy42MyA3LjAyMDAxIDE4IDcuNTkwMDEgMTggOC4yMTAwMUMxOCA4LjY1MDAxIDE4LjM2IDkuMDEwMDEgMTguOCA5LjAxMDAxQzE5LjI0IDkuMDEwMDEgMTkuNiA4LjY1MDAxIDE5LjYgOC4yMTAwMUMxOS42IDYuOTcwMDEgMTguODggNS44MzAwMSAxNy43NiA1LjMwMDAxTDExLjUgMi4zMTAwMUMxMC42IDEuODgwMDEgOS41NyAxLjg5MDAxIDguNjcgMi4zNDAwMUwyLjc4IDUuMzAwMDFDMS42OCA1Ljg1MDAxIDEgNi45NTAwMSAxIDguMTgwMDFWMTUuNzhDMSAxNy4wMSAxLjY4IDE4LjExIDIuNzggMTguNjZMOC42NyAyMS42MUM5LjEzIDIxLjg0IDkuNjIgMjEuOTUgMTAuMTEgMjEuOTVDMTAuNiAyMS45NSAxMS4wNSAyMS44NSAxMS40OSAyMS42NEwxMi4wNyAyMS4zN0MxMi40NyAyMS4xOCAxMi42NCAyMC43IDEyLjQ1IDIwLjNDMTIuMjYgMTkuOSAxMS43OCAxOS43MyAxMS4zOCAxOS45MkwxMS4zOSAxOS45M1pNMTEuMTkgNy4xNTAwMUMxMC43MiA2Ljk0MDAxIDEwLjE2IDYuOTMwMDEgOS42OSA3LjE0MDAxTDYuMTUgOC42NzAwMUM1LjU1IDguOTMwMDEgNS4xNyA5LjQ3MDAxIDUuMTcgMTAuMDdWMTMuOTFDNS4xNyAxNC41MSA1LjU1IDE1LjA2IDYuMTUgMTUuMzFMOS42OSAxNi44NEMxMC4wOCAxNy4wMSAxMC41MSAxNy4wMyAxMC45MSAxNi45MkMxMC45NiAxMy45NCAxMi44NCAxMS40MyAxNS40NCAxMC41VjEwLjA1QzE1LjQ0IDkuNDYwMDEgMTUuMDcgOC45MzAwMSAxNC40OSA4LjY2MDAxTDExLjE4IDcuMTYwMDFMMTEuMTkgNy4xNTAwMVpNMjIuMjYgMTYuMzhDMjEuOTIgMTYuMzggMjEuNjQgMTYuNjcgMjEuNjQgMTcuMDJDMjEuNjQgMTkuMjkgMTkuODIgMjEuMTggMTcuNjQgMjEuMThDMTUuNDYgMjEuMTggMTMuNjQgMTkuMjkgMTMuNjQgMTcuMDJDMTMuNjQgMTQuNzUgMTUuNDYgMTIuODYgMTcuNjQgMTIuODZIMTguNjJMMTcuODIgMTMuNjlDMTcuNTggMTMuOTQgMTcuNTggMTQuMzUgMTcuODIgMTQuNTlDMTguMDYgMTQuODQgMTguNDUgMTQuODQgMTguNjkgMTQuNTlMMjAuNTQgMTIuNjdDMjAuNzggMTIuNDIgMjAuNzggMTIuMDEgMjAuNTQgMTEuNzZMMTguNjkgOS44NDAwMUMxOC40NSA5LjU5MDAxIDE4LjA2IDkuNTkwMDEgMTcuODIgOS44NDAwMUMxNy41OCAxMC4wOSAxNy41OCAxMC41IDE3LjgyIDEwLjc1TDE4LjYyIDExLjU4SDE3LjY0QzE0Ljc3IDExLjU4IDEyLjQgMTQuMDQgMTIuNCAxNy4wMkMxMi40IDIwIDE0Ljc3IDIyLjQ2IDE3LjY0IDIyLjQ2QzIwLjUxIDIyLjQ2IDIyLjg4IDIwIDIyLjg4IDE3LjAyQzIyLjg4IDE2LjY3IDIyLjYgMTYuMzggMjIuMjYgMTYuMzhaIiBmaWxsPSIjMUQyNjMyIi8+Cjwvc3ZnPgo="),
		Technology:  extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: DeploymentTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "deployment",
					Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
		}),
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
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{}),
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

	// First check if there is already an ongoing rollout
	statusCmd := exec.Command("kubectl",
		"rollout",
		"status",
		"--watch=false",
		"--namespace",
		state.Namespace,
		fmt.Sprintf("deployment/%s", state.Deployment))
	statusOut, _ := statusCmd.CombinedOutput()
	statusOutStr := string(statusOut)

	log.Info().Msgf("Rollout status output: %s", statusOutStr)
	if strings.Contains(strings.ToLower(statusOutStr), "waiting") {
		return nil, extension_kit.ToError("Cannot start rollout restart: there is already an ongoing rollout for this deployment", nil)
	}
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
