// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extargorollout

import (
	"context"
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func NewScaleArgoRolloutAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getScaleArgoRolloutDescription(),
		OptsProvider: scaleArgoRollout(),
	}
}

type ScaleArgoRolloutConfig struct {
	ReplicaCount int
}

func getScaleArgoRolloutDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ArgoRolloutScaleActionId,
		Label:       "Scale Argo Rollout",
		Description: "Up-/ or downscale a Kubernetes Argo Rollout",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        new(ArgoRolloutIcon),
		Technology:  new("Kubernetes"),
		TargetSelection: new(action_kit_api.TargetSelection{
			TargetType: ArgoRolloutTargetType,
			SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "argo rollout",
					Description: new("Find Argo Rollout by cluster, namespace and rollout"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.argo-rollout=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  new("The duration of the action. The Argo Rollout will be scaled back to the original value after the action."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("180s"),
				Required:     new(true),
			},
			{
				Name:         "replicaCount",
				Label:        "Replica Count",
				Description:  new("The new replica count."),
				Type:         action_kit_api.ActionParameterTypeInteger,
				DefaultValue: new("1"),
				Required:     new(true),
			},
		},
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status:  &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:    &action_kit_api.MutatingEndpointReference{},
	}
}

func scaleArgoRollout() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		rollout := request.Target.Attributes["k8s.argo-rollout"][0]

		var config ScaleArgoRolloutConfig
		if err := extconversion.Convert(request.Config, &config); err != nil {
			return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
		}

		rolloutDefinition := client.K8S.ArgoRolloutByNamespaceAndName(namespace, rollout)
		if rolloutDefinition == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find Argo Rollout %s/%s.", namespace, rollout), nil)
		}

		oldReplicaCount, found, err := unstructured.NestedInt64(rolloutDefinition.Object, "spec", "replicas")
		if err != nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find current replicaCount for Argo Rollout %s/%s.", namespace, rollout), err)
		}
		if !found {
			// Replicas is optional in Argo Rollout spec, default value is 1
			// See: https://github.com/argoproj/argo-rollouts/blob/4d341b31dbd2e673c766b5f09cb6803a6ae2192e/utils/defaults/defaults.go#L111
			oldReplicaCount = 1
		}

		command := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", config.ReplicaCount),
			fmt.Sprintf("--current-replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("rollout/%s", rollout),
		}

		rollbackCommand := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("rollout/%s", rollout),
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: rollbackCommand,
			LogTargetType:   "argo-rollout",
			LogTargetName:   fmt.Sprintf("%s/%s", namespace, rollout),
			LogActionName:   "scale argo rollout",
		}, nil
	}
}
