// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/extcommon"
)

func NewDrainNodeAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getDrainNodeDescription(),
		OptsProvider: drainNode(),
	}
}

func getDrainNodeDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          DrainNodeActionId,
		Label:       "Drain Node",
		Description: "Drain a Kubernetes node",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(drainNodeIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: NodeTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find node by its name"),
					Query:       "host.hostname=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Name:         "duration",
				Type:         action_kit_api.Duration,
				Description:  extutil.Ptr("The duration of the action. The node will be uncordoned after the action."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("180s"),
				Order:        extutil.Ptr(0),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:    &action_kit_api.MutatingEndpointReference{},
	}
}

func drainNode() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		nodeName := request.Target.Attributes["host.hostname"][0]

		command := []string{
			"kubectl",
			"drain",
			nodeName,
			"--pod-selector=steadybit.com/extension!=true,steadybit.com/outpost!=true,steadybit.com/agent!=true",
			"--delete-emptydir-data",
			"--ignore-daemonsets",
			"--force"}

		rollbackCommand := []string{
			"kubectl",
			"uncordon",
			nodeName,
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: &rollbackCommand,
			LogTargetType:   "node",
			LogTargetName:   nodeName,
			LogActionName:   "drain node",
		}, nil
	}
}
