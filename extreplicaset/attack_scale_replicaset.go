// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extreplicaset

import (
	"context"
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extconversion"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewScaleReplicaSetAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getScaleReplicaSetDescription(),
		OptsProvider: scaleReplicaSet(),
	}
}

type ScaleReplicaSetConfig struct {
	ReplicaCount int
}

func getScaleReplicaSetDescription() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ScaleReplicaSetActionId,
		Label:       "Scale ReplicaSet",
		Description: "Up-/ or downscale a Kubernetes ReplicaSet",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTkuODQ5NjEgMTUuMzI3MUMxMC4xNTM3IDE1LjMyNzIgMTAuNDAwNCAxNS41NzM4IDEwLjQwMDQgMTUuODc3OVYyMC4zNzZDMTAuNDAwNCAyMC42ODAxIDEwLjE1MzcgMjAuOTI2OCA5Ljg0OTYxIDIwLjkyNjhIMi41NTA3OEMyLjI0NjcyIDIwLjkyNjcgMiAyMC42ODAxIDIgMjAuMzc2VjE1Ljg3NzlDMiAxNS41NzM4IDIuMjQ2NzIgMTUuMzI3MiAyLjU1MDc4IDE1LjMyNzFIOS44NDk2MVpNMTIuOTEyMSAxNy43NjQ2QzEyLjkxMTkgMTguMzc4NiAxMi40MjY2IDE4Ljg3NTkgMTEuODI4MSAxOC44NzZIMTEuMDQ4OFYxOC4wNDJIMTEuODI4MUMxMS45Nzc2IDE4LjA0MTkgMTIuMDk4NCAxNy45MTggMTIuMDk4NiAxNy43NjQ2VjE2LjY4NjVIMTIuOTEyMVYxNy43NjQ2Wk0xOS4wNzAzIDNDMjAuNDY4NCAzLjAwMDMzIDIxLjU5OTYgNC4xMzIwNyAyMS41OTk2IDUuNTMwMjdWMTIuODg5NkMyMS41OTk0IDE0LjI4NzcgMjAuNDY4MyAxNS40MTk2IDE5LjA3MDMgMTUuNDE5OUgxNy40NkMxNy4wODMgMTUuNDE5OSAxNi43NyAxNS4xMDczIDE2Ljc2OTUgMTQuNzMwNUMxNi43Njk1IDE0LjM1MzMgMTcuMDgyOCAxNC4wNCAxNy40NiAxNC4wNEgxOS4wNzAzQzE5LjcwNDcgMTQuMDM5NyAyMC4yMTk2IDEzLjUyNDEgMjAuMjE5NyAxMi44ODk2VjUuNTMwMjdDMjAuMjE5NyA0Ljg5NTY3IDE5LjcwNDggNC4zODAyMSAxOS4wNzAzIDQuMzc5ODhIOC45NTAyQzguMzE1NCA0LjM3OTg4IDcuNzk5OCA0Ljg5NTQ3IDcuNzk5OCA1LjUzMDI3VjYuNjE1MjNDNy43OTk2NCA2Ljk5MjI2IDcuNDg2NDIgNy4zMDU2IDcuMTA5MzggNy4zMDU2NkM2LjczMjQ4IDcuMzA1NDEgNi40MjAwOSA2Ljk5MjE1IDYuNDE5OTIgNi42MTUyM1Y1LjUzMDI3QzYuNDE5OTIgNC4xMzE4NyA3LjU1MTggMyA4Ljk1MDIgM0gxOS4wNzAzWk02LjM3NSAxMy4xNzY4SDUuNTk1N0M1LjQ0NjA3IDEzLjE3NjggNS4zMjUyIDEzLjMwMTUgNS4zMjUyIDEzLjQ1NTFWMTQuNTMyMkg0LjUxMTcyVjEzLjQ1NTFDNC41MTE3MiAxMi44NDA5IDQuOTk3MTEgMTIuMzQyOCA1LjU5NTcgMTIuMzQyOEg2LjM3NVYxMy4xNzY4Wk0xMS44MjgxIDEyLjM0MjhDMTIuNDI2NyAxMi4zNDI4IDEyLjkxMjEgMTIuODQwOSAxMi45MTIxIDEzLjQ1NTFWMTQuNTMyMkgxMi4wOTg2VjEzLjQ1NTFDMTIuMDk4NiAxMy4zMDE2IDExLjk3NzcgMTMuMTc2OCAxMS44MjgxIDEzLjE3NjhIMTEuMDQ4OFYxMi4zNDI4SDExLjgyODFaTTkuNDkxMjEgMTMuMTc2OEg3LjkzMjYyVjEyLjM0MjhIOS40OTEyMVYxMy4xNzY4Wk0xOC4zNzExIDUuNzU5NzdDMTguNjM3NiA1Ljc2MDA4IDE4Ljg0ODYgNS45ODA5MiAxOC44NDg2IDYuMjM4MjhWOC4yMzQzOEMxOC44NDg1IDguNTAwODQgMTguNjM3NSA4LjcxMjU5IDE4LjM3MTEgOC43MTI4OUMxOC4xMDQ0IDguNzEyODkgMTcuODkyNyA4LjQ5MTg0IDE3Ljg5MjYgOC4yMzQzOFY3LjM3MDEyTDE2LjUwMjkgOC43MzE0NVY4Ljc1TDE0LjE1NzIgMTEuMDY4NEgxNS4wMTI3QzE1LjI3OTQgMTEuMDY4NSAxNS40OTEyIDExLjI4MDEgMTUuNDkxMiAxMS41NDY5QzE1LjQ5MSAxMS44MTM0IDE1LjI3MDEgMTIuMDI1MyAxNS4wMTI3IDEyLjAyNTRIMTMuMDA2OEMxMi43NDAzIDEyLjAyNTMgMTIuNTI4NSAxMS44MDQzIDEyLjUyODMgMTEuNTQ2OVY5LjU0MTAyQzEyLjUyODQgOS4yNzQzOCAxMi43NDAyIDkuMDYyNjEgMTMuMDA2OCA5LjA2MjVDMTMuMjczNSA5LjA2MjU3IDEzLjQ4NTIgOS4yODM1NCAxMy40ODU0IDkuNTQxMDJWMTAuNDA2MkwxNS44MzExIDguMDg3ODlWOC4wNjkzNEwxNy4yMTE5IDYuNzE2OEgxNi4zNTU1QzE2LjA4ODkgNi43MTY2IDE1Ljg3ODEgNi41MDQ4MSAxNS44Nzc5IDYuMjM4MjhDMTUuODc4IDUuOTcxNjcgMTYuMDk4MSA1Ljc1OTk3IDE2LjM1NTUgNS43NTk3N0gxOC4zNzExWiIgZmlsbD0iY3VycmVudENvbG9yIi8+Cjwvc3ZnPgo="),
		Technology:  extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: ReplicaSetTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "replicaset",
					Description: extutil.Ptr("Find replicaset by cluster, namespace and replicaset"),
					Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.replicaset=\"\"",
				},
			}),
		}),
		TimeControl: action_kit_api.TimeControlExternal,
		Kind:        action_kit_api.Attack,
		Parameters: []action_kit_api.ActionParameter{
			{
				Label:        "Duration",
				Description:  extutil.Ptr("The duration of the action. The replicaset will be scaled back to the original value after the action."),
				Name:         "duration",
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("180s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "replicaCount",
				Label:        "Replica Count",
				Description:  extutil.Ptr("The new replica count."),
				Type:         action_kit_api.ActionParameterTypeInteger,
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

func scaleReplicaSet() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		replicaset := request.Target.Attributes["k8s.replicaset"][0]

		if workloadTypes, ok := request.Target.Attributes["k8s.workload-type"]; ok && len(workloadTypes) > 0 {
			if workloadTypes[0] == "deployment" {
				return nil, extension_kit.ToError("Scaling replicaSets controlled by deployments will have no effect. Please use the 'Scale Deployment' action instead.", nil)
			}
		}

		var config ScaleReplicaSetConfig
		if err := extconversion.Convert(request.Config, &config); err != nil {
			return nil, extension_kit.ToError("Failed to unmarshal the config.", err)
		}

		replicasetDefinition := client.K8S.ReplicaSetByNamespaceAndName(namespace, replicaset)
		if replicasetDefinition == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find replicaset %s/%s.", namespace, replicaset), nil)
		}
		if replicasetDefinition.Spec.Replicas == nil {
			return nil, extension_kit.ToError(fmt.Sprintf("Failed to find current replicaCount for replicaset %s/%s.", namespace, replicaset), nil)
		}

		oldReplicaCount := *replicasetDefinition.Spec.Replicas

		command := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", config.ReplicaCount),
			fmt.Sprintf("--current-replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("replicaset/%s", replicaset),
		}

		rollbackCommand := []string{"kubectl",
			"scale",
			fmt.Sprintf("--replicas=%d", oldReplicaCount),
			fmt.Sprintf("--namespace=%s", namespace),
			fmt.Sprintf("replicaset/%s", replicaset),
		}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: rollbackCommand,
			LogTargetType:   "replicaset",
			LogTargetName:   fmt.Sprintf("%s/%s", namespace, replicaset),
			LogActionName:   "scale replicaset",
		}, nil
	}
}
