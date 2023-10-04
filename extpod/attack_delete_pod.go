// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpod

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"os/exec"
)

type DeletePodAction struct {
}

type DeletePodActionState struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
}

func NewDeletePodAction() action_kit_sdk.Action[DeletePodActionState] {
	return DeletePodAction{}
}

var _ action_kit_sdk.Action[DeletePodActionState] = (*DeletePodAction)(nil)

func (f DeletePodAction) NewEmptyState() DeletePodActionState {
	return DeletePodActionState{}
}

func (f DeletePodAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          DeletePodActionId,
		Label:       "Delete Pod",
		Description: "Delete Pods in a Kubernetes cluster",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(deletePodActionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: PodTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find pods by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("state"),
		TimeControl: action_kit_api.TimeControlInstantaneous,
		Kind:        action_kit_api.Attack,
		Parameters:  []action_kit_api.ActionParameter{},
		Prepare: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/pod/attack/rollout-restart/prepare",
		},
		Start: action_kit_api.MutatingEndpointReference{
			Method: "POST",
			Path:   "/pod/attack/rollout-restart/start",
		},
	}
}

func (f DeletePodAction) Prepare(_ context.Context, state *DeletePodActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.Namespace = request.Target.Attributes["k8s.namespace"][0]
	state.Pod = request.Target.Attributes["k8s.pod.name"][0]
	return nil, nil
}

func (f DeletePodAction) Start(_ context.Context, state *DeletePodActionState) (*action_kit_api.StartResult, error) {
	log.Info().Msgf("Delete pod %+v", state)

	cmd := exec.Command("kubectl",
		"delete",
		"pod",
		"--namespace",
		state.Namespace,
		state.Pod)
	cmdOut, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		return nil, extension_kit.ToError(fmt.Sprintf("Failed to delete pod: %s", cmdOut), cmdErr)
	}

	return nil, nil
}
