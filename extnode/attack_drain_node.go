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
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0yMi45ODIgMTUuNzhDMjIuNjIyIDE1LjQyIDIyLjAzMiAxNS40MiAyMS42NjIgMTUuNzhMMTkuMDYyIDE4LjM4VjExLjE4QzE5LjA2MiAxMC43NCAxOC42NDIgMTAuMzggMTguMTMyIDEwLjM4QzE3LjYyMiAxMC4zOCAxNy4yMDIgMTAuNzQgMTcuMjAyIDExLjE4VjE4LjM4TDE0LjYwMiAxNS43OEMxNC4yNDIgMTUuNDIgMTMuNjUyIDE1LjQyIDEzLjI4MiAxNS43OEMxMi45MjIgMTYuMTQgMTIuOTIyIDE2LjczIDEzLjI4MiAxNy4xTDE3LjMzMiAyMS4xNUMxNy40OTIgMjEuMzkgMTcuNzgyIDIxLjU3IDE4LjEzMiAyMS41N0MxOC40ODIgMjEuNTcgMTguNzcyIDIxLjQgMTguOTMyIDIxLjE1TDIyLjk4MiAxNy4xQzIzLjM0MiAxNi43NCAyMy4zNDIgMTYuMTUgMjIuOTgyIDE1Ljc4Wk0xLjg4MjAyIDEzLjM1TDEwLjQ3MiAxNi41M0MxMC45MTIgMTYuNjkgMTEuNDEyIDE2LjQ3IDExLjU3MiAxNi4wMkMxMS43MzIgMTUuNTcgMTEuNTEyIDE1LjA4IDExLjA2MiAxNC45MkwyLjQ4MjAyIDExLjc0QzIuMDQyMDIgMTEuNTggMS41NDIwMiAxMS44IDEuMzgyMDIgMTIuMjVDMS4yMjIwMiAxMi43IDEuNDQyMDIgMTMuMTkgMS44OTIwMiAxMy4zNUgxLjg4MjAyWk0yLjA2MjAyIDguMTZMMTEuMDcyIDExLjU2QzExLjE3MiAxMS42IDExLjI4MiAxMS42MiAxMS4zOTIgMTEuNjJDMTEuNTAyIDExLjYyIDExLjYwMiAxMS42IDExLjcxMiAxMS41NkwyMC43MjIgOC4xNkMyMS4wNzIgOC4wMyAyMS4zMTIgNy42OSAyMS4zMTIgNy4zMUMyMS4zMTIgNi45MyAyMS4wODIgNi41OSAyMC43MjIgNi40NkwxMS43MDIgMy4wNkMxMS42MDIgMy4wMiAxMS40OTIgMyAxMS4zODIgM0MxMS4yNzIgMyAxMS4xNzIgMy4wMiAxMS4wNjIgMy4wNkwyLjA2MjAyIDYuNDZDMS43MTIwMiA2LjU5IDEuNDcyMDIgNi45MyAxLjQ3MjAyIDcuMzFDMS40NzIwMiA3LjY5IDEuNzAyMDIgOC4wMyAyLjA2MjAyIDguMTZaTTExLjM5MiA0LjY1TDE4LjQ0MiA3LjMxTDExLjM5MiA5Ljk3TDQuMzQyMDIgNy4zMUwxMS4zOTIgNC42NVpNMTIuMjUyIDE5LjE3TDEwLjY3MiAxOS43NEwyLjE1MjAyIDE2LjU4QzEuNzEyMDIgMTYuNDIgMS4yMTIwMiAxNi42NCAxLjA1MjAyIDE3LjA5QzAuODkyMDIxIDE3LjU0IDEuMTEyMDIgMTguMDMgMS41NjIwMiAxOC4xOUwxMC4xNTIgMjEuMzdDMTAuMzQyIDIxLjQ0IDEwLjUzMiAyMS40MyAxMC43MTIgMjEuMzdDMTAuNzQyIDIxLjM3IDEwLjc2MiAyMS4zNyAxMC43OTIgMjEuMzZMMTIuNzgyIDIwLjY1QzEzLjE5MiAyMC41IDEzLjQwMiAyMC4wNSAxMy4yNjIgMTkuNjRDMTMuMTIyIDE5LjIzIDEyLjY2MiAxOS4wMiAxMi4yNTIgMTkuMTdaIiBmaWxsPSIjMUQyNjMyIi8+Cjwvc3ZnPgo="),
		Technology:  extutil.Ptr("Kubernetes"),
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
			"--pod-selector=steadybit.com/extension!=true,steadybit.com/agent!=true",
			"--delete-emptydir-data",
			"--ignore-daemonsets",
			"--force"}

		rollbackPreconditionCommand := []string{
			"kubectl",
			"get",
			"node",
			nodeName,
		}

		rollbackCommand := []string{
			"kubectl",
			"uncordon",
			nodeName,
		}

		return &extcommon.KubectlOpts{
			Command:                     command,
			RollbackPreconditionCommand: &rollbackPreconditionCommand,
			RollbackCommand:             &rollbackCommand,
			LogTargetType:               "node",
			LogTargetName:               nodeName,
			LogActionName:               "drain node",
		}, nil
	}
}
