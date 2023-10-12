// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extstatefulset

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
)

func NewScaleStatefulSetAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getScaleStatefulSetDescription(),
		OptsProvider: scaleStatefulSet(),
	}
}

type ScaleStatefulSetConfig struct {
	ReplicaCount int
}

func getScaleStatefulSetDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ScaleStatefulSetActionId,
		Label:       "Scale StatefulSet",
		Description: "Up-/ or downscale a Kubernetes StatefulSet",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(scaleStatefulSetIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: StatefulSetTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "default",
					Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.statefulset=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  extutil.Ptr("The duration of the action. The statefulSet will be scaled back to the original value after the action."),
				Name:         "duration",
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("180s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "replicaCount",
				Label:        "Replica Count",
				Description:  extutil.Ptr("The new replica count."),
				Type:         action_kit_api.Integer,
				DefaultValue: extutil.Ptr("1"),
				Required:     extutil.Ptr(true),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:    &action_kit_api.MutatingEndpointReference{},
	}
}

func scaleStatefulSet() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		statefulSet := request.Target.Attributes["k8s.statefulset"][0]

		var config ScaleStatefulSetConfig
		if err := extconversion.Convert(request.Config, &config); err != nil {
			return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
		}

		statefulSetDefinition := client.K8S.StatefulSetByNamespaceAndName(namespace, statefulSet)
		if statefulSetDefinition == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find statefulSet %s/%s.", namespace, statefulSet), nil)
		}
		if statefulSetDefinition.Spec.Replicas == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find current replicaCount for statefulSet %s/%s.", namespace, statefulSet), nil)
		}

		oldReplicaCount := *statefulSetDefinition.Spec.Replicas

		command := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", config.ReplicaCount),
			fmt.Sprintf("--current-replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("statefulset/%s", statefulSet),
		}

		rollbackCommand := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("statefulset/%s", statefulSet),
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: &rollbackCommand,
			LogTargetType:   "statefulSet",
			LogTargetName:   fmt.Sprintf("%s/%s", namespace, statefulSet),
			LogActionName:   "scale statefulSet",
		}, nil
	}
}
