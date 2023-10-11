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
	"github.com/steadybit/extension-kubernetes/extcommon"
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
		Id:          TaintNodeActionId,
		Label:       "Taint Node",
		Description: "Taint a Kubernetes node",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(taintNodeIcon),
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
				Description:  extutil.Ptr("The duration of the attack. The taint will be removed after the attack."),
				Advanced:     extutil.Ptr(false),
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("180s"),
				Order:        extutil.Ptr(0),
			},
			{
				Label:       "Key",
				Name:        "key",
				Type:        action_kit_api.String,
				Description: extutil.Ptr("The key of the taint."),
				Advanced:    extutil.Ptr(false),
				Required:    extutil.Ptr(true),
				Order:       extutil.Ptr(1),
			},
			{
				Label:       "Value",
				Name:        "value",
				Type:        action_kit_api.String,
				Description: extutil.Ptr("The optional value of the taint."),
				Advanced:    extutil.Ptr(false),
				Required:    extutil.Ptr(false),
				Order:       extutil.Ptr(1),
			},
			{
				Label:        "Effect",
				Name:         "effect",
				Type:         action_kit_api.String,
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
