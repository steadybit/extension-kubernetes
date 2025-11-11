// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
)

type CheckDeploymentRolloutStatusAction struct {
}

var referenceTime = time.Now()

type CheckDeploymentRolloutStatusState struct {
	Cluster    string        `json:"cluster"`
	Namespace  string        `json:"namespace"`
	Deployment string        `json:"deployment"`
	EndOffset  time.Duration `json:"endOffset"`
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
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTExLjM5IDE5LjkzTDEwLjgxIDIwLjJDMTAuMzYgMjAuNDIgOS44NCAyMC40MSA5LjM5IDIwLjE5TDMuNSAxNy4yNEMyLjk1IDE2Ljk2IDIuNiAxNi40MSAyLjYgMTUuNzlWOC4xOTAwMUMyLjYgNy41NzAwMSAyLjk0IDcuMDIwMDEgMy40OSA2Ljc0MDAxTDkuMzggMy43OTAwMUM5LjgzIDMuNTcwMDEgMTAuMzUgMy41NjAwMSAxMC44IDMuNzcwMDFMMTcuMDcgNi43NTAwMUMxNy42MyA3LjAyMDAxIDE4IDcuNTkwMDEgMTggOC4yMTAwMUMxOCA4LjY1MDAxIDE4LjM2IDkuMDEwMDEgMTguOCA5LjAxMDAxQzE5LjI0IDkuMDEwMDEgMTkuNiA4LjY1MDAxIDE5LjYgOC4yMTAwMUMxOS42IDYuOTcwMDEgMTguODggNS44MzAwMSAxNy43NiA1LjMwMDAxTDExLjUgMi4zMTAwMUMxMC42IDEuODgwMDEgOS41NyAxLjg5MDAxIDguNjcgMi4zNDAwMUwyLjc4IDUuMzAwMDFDMS42OCA1Ljg1MDAxIDEgNi45NTAwMSAxIDguMTgwMDFWMTUuNzhDMSAxNy4wMSAxLjY4IDE4LjExIDIuNzggMTguNjZMOC42NyAyMS42MUM5LjEzIDIxLjg0IDkuNjIgMjEuOTUgMTAuMTEgMjEuOTVDMTAuNiAyMS45NSAxMS4wNSAyMS44NSAxMS40OSAyMS42NEwxMi4wNyAyMS4zN0MxMi40NyAyMS4xOCAxMi42NCAyMC43IDEyLjQ1IDIwLjNDMTIuMjYgMTkuOSAxMS43OCAxOS43MyAxMS4zOCAxOS45MkwxMS4zOSAxOS45M1pNMTEuMTkgNy4xNTAwMUMxMC43MiA2Ljk0MDAxIDEwLjE2IDYuOTMwMDEgOS42OSA3LjE0MDAxTDYuMTUgOC42NzAwMUM1LjU1IDguOTMwMDEgNS4xNyA5LjQ3MDAxIDUuMTcgMTAuMDdWMTMuOTFDNS4xNyAxNC41MSA1LjU1IDE1LjA2IDYuMTUgMTUuMzFMOS42OSAxNi44NEMxMC4wOCAxNy4wMSAxMC41MSAxNy4wMyAxMC45MSAxNi45MkMxMC45NiAxMy45NCAxMi44NCAxMS40MyAxNS40NCAxMC41VjEwLjA1QzE1LjQ0IDkuNDYwMDEgMTUuMDcgOC45MzAwMSAxNC40OSA4LjY2MDAxTDExLjE4IDcuMTYwMDFMMTEuMTkgNy4xNTAwMVpNMTcuMzQgMTEuMTlDMTQuMzkgMTEuMTkgMTIgMTMuNTggMTIgMTYuNTNDMTIgMTkuNDggMTQuMzkgMjEuODcgMTcuMzQgMjEuODdDMjAuMjkgMjEuODcgMjIuNjggMTkuNDggMjIuNjggMTYuNTNDMjIuNjggMTMuNTggMjAuMjkgMTEuMTkgMTcuMzQgMTEuMTlaTTE3LjM0IDIwLjY4QzE1LjA1IDIwLjY4IDEzLjE5IDE4LjgyIDEzLjE5IDE2LjUzQzEzLjE5IDE0LjI0IDE1LjA1IDEyLjM4IDE3LjM0IDEyLjM4QzE5LjYzIDEyLjM4IDIxLjQ5IDE0LjI0IDIxLjQ5IDE2LjUzQzIxLjQ5IDE4LjgyIDE5LjYzIDIwLjY4IDE3LjM0IDIwLjY4Wk0xOC40NSAxMy41NkMxOC4xNiAxMy4zNiAxNy44MiAxMy4yNCAxNy40NyAxMy4yMUMxNy4xMiAxMy4xOSAxNi43NyAxMy4yNiAxNi40NSAxMy40MkMxNi4xNCAxMy41OCAxNS44NyAxMy44MyAxNS42OSAxNC4xM0MxNS41MSAxNC40MyAxNS40MSAxNC43OCAxNS40MSAxNS4xM0MxNS40MSAxNS40MiAxNS42NCAxNS42NSAxNS45MyAxNS42NUMxNi4yMiAxNS42NSAxNi40NSAxNS40MiAxNi40NSAxNS4xM0MxNi40NSAxNC45NyAxNi40OSAxNC44MSAxNi41OCAxNC42OEMxNi42NiAxNC41NCAxNi43OCAxNC40MyAxNi45MyAxNC4zNkMxNy4wOCAxNC4yOSAxNy4yMyAxNC4yNSAxNy4zOSAxNC4yNkMxNy41NSAxNC4yNyAxNy43IDE0LjMzIDE3LjgzIDE0LjQyQzE3Ljk2IDE0LjUxIDE4LjA2IDE0LjY0IDE4LjEzIDE0Ljc5QzE4LjE5IDE0Ljk0IDE4LjIyIDE1LjEgMTguMTkgMTUuMjZDMTguMTcgMTUuNDIgMTguMSAxNS41NyAxOCAxNS42OUMxNy45IDE1LjgxIDE3Ljc3IDE1LjkxIDE3LjYxIDE1Ljk2QzE3LjM3IDE2LjA0IDE3LjE2IDE2LjIgMTcuMDIgMTYuNDFDMTYuODcgMTYuNjIgMTYuOCAxNi44NiAxNi44IDE3LjEyVjE3LjMzQzE2LjggMTcuNjIgMTcuMDMgMTcuODUgMTcuMzIgMTcuODVDMTcuNjEgMTcuODUgMTcuODQgMTcuNjIgMTcuODQgMTcuMzNWMTcuMTJDMTcuODQgMTcuMTIgMTcuODUgMTcuMDUgMTcuODcgMTcuMDJDMTcuODkgMTYuOTkgMTcuOTIgMTYuOTcgMTcuOTUgMTYuOTZDMTguMjggMTYuODQgMTguNTggMTYuNjQgMTguOCAxNi4zNkMxOS4wMiAxNi4wOCAxOS4xNyAxNS43NiAxOS4yMSAxNS40MUMxOS4yNiAxNS4wNiAxOS4yMSAxNC43IDE5LjA3IDE0LjM4QzE4LjkzIDE0LjA2IDE4LjcgMTMuNzggMTguNDIgMTMuNTdMMTguNDUgMTMuNTZaTTE3LjM0IDE4LjQ1QzE3LjIgMTguNDUgMTcuMDcgMTguNDkgMTYuOTUgMTguNTdDMTYuODMgMTguNjUgMTYuNzUgMTguNzYgMTYuNjkgMTguODhDMTYuNjQgMTkuMDEgMTYuNjIgMTkuMTUgMTYuNjUgMTkuMjhDMTYuNjggMTkuNDEgMTYuNzQgMTkuNTQgMTYuODQgMTkuNjRDMTYuOTQgMTkuNzQgMTcuMDYgMTkuOCAxNy4yIDE5LjgzQzE3LjM0IDE5Ljg2IDE3LjQ4IDE5Ljg0IDE3LjYgMTkuNzlDMTcuNzMgMTkuNzQgMTcuODQgMTkuNjUgMTcuOTEgMTkuNTNDMTcuOTkgMTkuNDEgMTguMDMgMTkuMjggMTguMDMgMTkuMTRDMTguMDMgMTguOTUgMTcuOTYgMTguNzggMTcuODMgMTguNjVDMTcuNyAxOC41MiAxNy41MiAxOC40NSAxNy4zNCAxOC40NVoiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:  extutil.Ptr("Kubernetes"),
		Category:    extutil.Ptr("Kubernetes"), //Can be removed in Q1/24 - support for backward compatibility of old sidebar
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
		Kind:        action_kit_api.Check,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Timeout",
				Description:  extutil.Ptr("Maximum time to wait for the rollout to be rolled out completely."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
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

	if config.Duration != 0 {
		duration := time.Duration(int(time.Millisecond) * config.Duration)
		state.EndOffset = time.Since(referenceTime) + duration
	}

	state.Cluster = request.Target.Attributes["k8s.cluster-name"][0]
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.Deployment = request.Target.Attributes["k8s.deployment"][0]
	return nil, nil
}

func (f CheckDeploymentRolloutStatusAction) Start(_ context.Context, _ *CheckDeploymentRolloutStatusState) (*action_kit_api.StartResult, error) {
	return nil, nil
}

func (f CheckDeploymentRolloutStatusAction) Status(_ context.Context, state *CheckDeploymentRolloutStatusState) (*action_kit_api.StatusResult, error) {
	if state.EndOffset != 0 && time.Since(referenceTime) > state.EndOffset {
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
