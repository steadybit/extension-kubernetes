// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewTaintNodeAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getTaintNodeDescription(),
		OptsProvider: taintNode(),
	}
}

type TaintNodeConfig struct {
	Key    string
	Value  string
	Effect string
}

func getTaintNodeDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:              TaintNodeActionId,
		Label:           "Taint Node",
		Description:     "Taint a Kubernetes node",
		Version:         extbuild.GetSemverVersionStringOrUnknown(),
		Icon:            extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0yMy41MjIgMTQuNzY3NUMyMy4yNjIgMTQuMzc3NSAyMi44NDIgMTQuMTE3NSAyMi4zODIgMTQuMDQ3NUMyMS45MjIgMTMuOTc3NSAyMS40NDIgMTQuMTA3NSAyMS4wODIgMTQuMzk3NUwyMC43ODIgMTQuNjM3NVYxMC40Mjc1QzIwLjc4MiAxMC4wNzc1IDIwLjY3MiA5LjczNzUgMjAuNDYyIDkuNDY3NUMyMC4yNTIgOS4xOTc1IDE5Ljk0MiA5LjAwNzUgMTkuNjAyIDguOTI3NUMxOS40MjIgOC44OTc1IDE5LjI1MiA4Ljg4NzUgMTkuMDcyIDguOTI3NUMxOS4wNDIgOC42Mjc1IDE4LjkzMiA4LjM0NzUgMTguNzQyIDguMTA3NUMxOC41MjIgNy44Mjc1IDE4LjIwMiA3LjYzNzUgMTcuODQyIDcuNTY3NUMxNy40OTIgNy41MTc1IDE3LjEzMiA3LjU3NzUgMTYuODMyIDcuNzU3NUMxNi42MTIgNy44ODc1IDE2LjQ0MiA4LjA3NzUgMTYuMzEyIDguMjg3NUMxNi4xMDIgOC4yMjc1IDE1Ljg3MiA4LjIwNzUgMTUuNjQyIDguMjQ3NUMxNS4yOTIgOC4zMTc1IDE0Ljk3MiA4LjUwNzUgMTQuNzQyIDguNzg3NUMxNC41NTIgOS4wMjc1IDE0LjQ0MiA5LjMxNzUgMTQuNDEyIDkuNjA3NUMxNC4yNDIgOS41Nzc1IDE0LjA2MiA5LjU3NzUgMTMuODgyIDkuNjA3NUMxMy41NDIgOS42ODc1IDEzLjI0MiA5Ljg3NzUgMTMuMDIyIDEwLjE0NzVDMTIuODEyIDEwLjQxNzUgMTIuNjkyIDEwLjc2NzUgMTIuNzAyIDExLjEwNzVWMTcuMzc3NUMxMi43MDIgMTguMzY3NSAxMy4wODIgMTkuMjk3NSAxMy43ODIgMTkuOTg3NUMxNC40ODIgMjAuNjg3NSAxNS40MTIgMjEuMDY3NSAxNi4zOTIgMjEuMDY3NUgxNy45NzJDMTkuMTAyIDIxLjA2NzUgMjAuMTcyIDIwLjU3NzUgMjAuOTAyIDE5LjcxNzVMMjMuMzUyIDE2Ljg1NzVDMjMuNjAyIDE2LjU3NzUgMjMuNzUyIDE2LjIxNzUgMjMuNzgyIDE1Ljg0NzVDMjMuODEyIDE1LjQ2NzUgMjMuNzEyIDE1LjA4NzUgMjMuNTEyIDE0Ljc2NzVIMjMuNTIyWk0yMi40NjIgMTUuOTk3NUwxOS45NzIgMTguOTA3NUMxOS40NzIgMTkuNDg3NSAxOC43NTIgMTkuODE3NSAxNy45ODIgMTkuODE3NUgxNi40MDJDMTUuNzUyIDE5LjgxNzUgMTUuMTIyIDE5LjU1NzUgMTQuNjcyIDE5LjA5NzVDMTQuMjEyIDE4LjYzNzUgMTMuOTUyIDE4LjAxNzUgMTMuOTUyIDE3LjM2NzVWMTEuMDQ3NUMxMy45NTIgMTAuOTY3NSAxMy45OTIgMTAuODg3NSAxNC4wNzIgMTAuODQ3NUMxNC4xMTIgMTAuODI3NSAxNC4xNTIgMTAuODE3NSAxNC4xODIgMTAuODE3NUMxNC4yMjIgMTAuODE3NSAxNC4yNjIgMTAuODI3NSAxNC4zMDIgMTAuODQ3NUMxNC4zNzIgMTAuODg3NSAxNC40MTIgMTAuOTY3NSAxNC40MTIgMTEuMDQ3NVYxMy45NDc1QzE0LjQxMiAxNC4yODc1IDE0LjY5MiAxNC41Njc1IDE1LjAzMiAxNC41Njc1QzE1LjM3MiAxNC41Njc1IDE1LjY1MiAxNC4yODc1IDE1LjY1MiAxMy45NDc1VjkuNjc3NUMxNS42NTIgOS41OTc1IDE1LjY5MiA5LjUxNzUgMTUuNzcyIDkuNDc3NUMxNS44NDIgOS40Mzc1IDE1LjkzMiA5LjQzNzUgMTYuMDAyIDkuNDc3NUMxNi4wNzIgOS41MTc1IDE2LjExMiA5LjU5NzUgMTYuMTEyIDkuNjc3NVYxMy45NDc1QzE2LjExMiAxNC4yODc1IDE2LjM5MiAxNC41Njc1IDE2LjczMiAxNC41Njc1QzE3LjA3MiAxNC41Njc1IDE3LjM1MiAxNC4yODc1IDE3LjM1MiAxMy45NDc1VjguOTk3NUMxNy4zNTIgOC45MTc1IDE3LjM5MiA4LjgzNzUgMTcuNDYyIDguNzk3NUMxNy41MzIgOC43NTc1IDE3LjYyMiA4Ljc1NzUgMTcuNjkyIDguNzk3NUMxNy43NjIgOC44Mzc1IDE3LjgxMiA4LjkxNzUgMTcuODEyIDguOTk3NVYxMy45NDc1QzE3LjgxMiAxNC4yODc1IDE4LjA5MiAxNC41Njc1IDE4LjQzMiAxNC41Njc1QzE4Ljc3MiAxNC41Njc1IDE5LjA1MiAxNC4yODc1IDE5LjA1MiAxMy45NDc1VjEwLjM1NzVDMTkuMDUyIDEwLjI3NzUgMTkuMDkyIDEwLjE5NzUgMTkuMTYyIDEwLjE1NzVDMTkuMjMyIDEwLjExNzUgMTkuMzIyIDEwLjExNzUgMTkuMzkyIDEwLjE1NzVDMTkuNDYyIDEwLjE5NzUgMTkuNTEyIDEwLjI3NzUgMTkuNTEyIDEwLjM1NzVWMTUuNjQ3NUMxOS41MTIgMTUuODE3NSAxOS41ODIgMTUuOTc3NSAxOS42OTIgMTYuMDg3NUMxOS44MTIgMTYuMjA3NSAxOS45NzIgMTYuMjY3NSAyMC4xMzIgMTYuMjY3NUgyMC40NzJDMjAuNjEyIDE2LjI2NzUgMjAuNzUyIDE2LjIxNzUgMjAuODYyIDE2LjEyNzVMMjEuODMyIDE1LjM1NzVDMjEuOTkyIDE1LjIyNzUgMjIuMjUyIDE1LjIzNzUgMjIuNDAyIDE1LjM4NzVDMjIuNDgyIDE1LjQ2NzUgMjIuNTMyIDE1LjU2NzUgMjIuNTQyIDE1LjY3NzVDMjIuNTUyIDE1Ljc4NzUgMjIuNTEyIDE1Ljg5NzUgMjIuNDQyIDE1Ljk4NzVMMjIuNDYyIDE1Ljk5NzVaTTEwLjc0MiAxOS4zNTc1TDIuMTUyMDIgMTYuMTc3NUMxLjcxMjAyIDE2LjAxNzUgMS4yMTIwMiAxNi4yMzc1IDEuMDUyMDIgMTYuNjg3NUMwLjg5MjAyMSAxNy4xMjc1IDEuMTEyMDIgMTcuNjI3NSAxLjU2MjAyIDE3Ljc4NzVMMTAuMTUyIDIwLjk2NzVDMTAuNTkyIDIxLjEyNzUgMTEuMDkyIDIwLjkwNzUgMTEuMjUyIDIwLjQ1NzVDMTEuNDEyIDIwLjAxNzUgMTEuMTkyIDE5LjUxNzUgMTAuNzQyIDE5LjM1NzVaTTExLjE1MiAxNC41MDc1TDIuNTYyMDIgMTEuMzI3NUMyLjEyMjAyIDExLjE2NzUgMS42MjIwMiAxMS4zODc1IDEuNDYyMDIgMTEuODM3NUMxLjMwMjAyIDEyLjI4NzUgMS41MjIwMiAxMi43Nzc1IDEuOTcyMDIgMTIuOTM3NUwxMC41NjIgMTYuMTE3NUMxMS4wMDIgMTYuMjc3NSAxMS41MDIgMTYuMDU3NSAxMS42NjIgMTUuNjA3NUMxMS44MjIgMTUuMTU3NSAxMS42MDIgMTQuNjY3NSAxMS4xNTIgMTQuNTA3NVpNMTEuMjQyIDkuNTY3NUwxMS4xNDIgOS41Mjc1TDEwLjU5MiA5LjMyNzVMMTAuMDIyIDkuMTE3NUwzLjA3MjAyIDYuNTU3NUwxMS4xNDIgMy41Nzc1SDExLjE2MkwxOS4yMzIgNi41NTc1QzE5LjY0MiA2LjcwNzUgMjAuMDkyIDYuNDk3NSAyMC4yNDIgNi4wODc1QzIwLjM5MiA1LjY3NzUgMjAuMTgyIDUuMjI3NSAxOS43NzIgNS4wNzc1TDExLjcwMiAyLjA5NzVDMTEuMzQyIDEuOTY3NSAxMC45NTIgMS45Njc1IDEwLjU5MiAyLjA5NzVMMi41MjIwMiA1LjA3NzVDMS45MDIwMiA1LjMwNzUgMS40ODIwMiA1Ljg5NzUgMS40ODIwMiA2LjU1NzVDMS40ODIwMiA3LjIyNzUgMS45MDIwMiA3LjgwNzUgMi41MjIwMiA4LjAzNzVMMTAuMDcyIDEwLjgxNzVMMTAuNjkyIDExLjA0NzVDMTEuMTAyIDExLjE5NzUgMTEuNTUyIDEwLjk4NzUgMTEuNzAyIDEwLjU4NzVDMTEuODUyIDEwLjE3NzUgMTEuNjQyIDkuNzI3NSAxMS4yNDIgOS41Nzc1VjkuNTY3NVoiIGZpbGw9IiMxRDI2MzIiLz4KPC9zdmc+Cg=="),
		Technology:      extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(targetSelectionTemplates),
		TimeControl:     action_kit_api.TimeControlExternal,
		Kind:            action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				Description:  extutil.Ptr("The duration of the action. The taint will be removed after the action."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("180s"),
				Order:        extutil.Ptr(0),
			},
			{
				Label:       "Key",
				Name:        "key",
				Type:        action_kit_api.ActionParameterTypeString,
				Description: extutil.Ptr("The key of the taint."),
				Advanced:    extutil.Ptr(false),
				Required:    extutil.Ptr(true),
				Order:       extutil.Ptr(1),
			},
			{
				Label:       "Value",
				Name:        "value",
				Type:        action_kit_api.ActionParameterTypeString,
				Description: extutil.Ptr("The optional value of the taint."),
				Advanced:    extutil.Ptr(false),
				Required:    extutil.Ptr(false),
				Order:       extutil.Ptr(1),
			},
			{
				Label:        "Effect",
				Name:         "effect",
				Type:         action_kit_api.ActionParameterTypeString,
				Description:  extutil.Ptr("The effect of the taint."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("NoSchedule"),
				Order:        extutil.Ptr(2),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "NoSchedule",
						Value: "NoSchedule",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "PreferNoSchedule",
						Value: "PreferNoSchedule",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "NoExecute",
						Value: "NoExecute",
					},
				}),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:    &action_kit_api.MutatingEndpointReference{},
	}
}

func taintNode() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		nodeName := request.Target.Attributes["host.hostname"][0]
		var config TaintNodeConfig
		if err := extconversion.Convert(request.Config, &config); err != nil {
			return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
		}

		taint := config.Key
		if config.Value != "" {
			taint = taint + "=" + config.Value
		}
		taint = taint + ":" + config.Effect

		command := []string{"kubectl",
			"taint",
			"node",
			nodeName,
			taint}

		rollbackCommand := []string{
			"kubectl",
			"taint",
			"node",
			nodeName,
			taint + "-",
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: &rollbackCommand,
			LogTargetType:   "node",
			LogTargetName:   nodeName,
			LogActionName:   "taint node",
		}, nil
	}
}
