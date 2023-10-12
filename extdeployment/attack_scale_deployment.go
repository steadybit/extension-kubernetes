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
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
)

func NewScaleDeploymentAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getScaleDeploymentDescription(),
		OptsProvider: scaleDeployment(),
	}
}

type ScaleDeploymentConfig struct {
	ReplicaCount int
}

func getScaleDeploymentDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ScaleDeploymentActionId,
		Label:       "Scale Deployment",
		Description: "Up-/ or downscale a Kubernetes Deployment",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(scaleDeploymentIcon),
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
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  extutil.Ptr("The duration of the action. The deployment will be scaled back to the original value after the action."),
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

func scaleDeployment() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		deployment := request.Target.Attributes["k8s.deployment"][0]

		var config ScaleDeploymentConfig
		if err := extconversion.Convert(request.Config, &config); err != nil {
			return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
		}

		deploymentDefinition := client.K8S.DeploymentByNamespaceAndName(namespace, deployment)
		if deploymentDefinition == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find deployment %s/%s.", namespace, deployment), nil)
		}
		if deploymentDefinition.Spec.Replicas == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find current replicaCount for deployment %s/%s.", namespace, deployment), nil)
		}

		oldReplicaCount := *deploymentDefinition.Spec.Replicas

		command := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", config.ReplicaCount),
			fmt.Sprintf("--current-replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("deployment/%s", deployment),
		}

		rollbackCommand := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("deployment/%s", deployment),
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: &rollbackCommand,
			LogTargetType:   "deployment",
			LogTargetName:   fmt.Sprintf("%s/%s", namespace, deployment),
			LogActionName:   "scale deployment",
		}, nil
	}
}
