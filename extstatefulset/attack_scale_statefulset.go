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
		Icon:        extutil.Ptr("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0yMC4xMiAxNS41SDE4LjM3QzE3Ljk2IDE1LjUgMTcuNjIgMTUuMTYgMTcuNjIgMTQuNzVDMTcuNjIgMTQuMzQgMTcuOTYgMTQgMTguMzcgMTRIMjAuMTJDMjAuODEgMTQgMjEuMzcgMTMuNDQgMjEuMzcgMTIuNzVWNC43NUMyMS4zNyA0LjA2IDIwLjgxIDMuNSAyMC4xMiAzLjVIOS4xMkM4LjQzIDMuNSA3Ljg3IDQuMDYgNy44NyA0Ljc1VjUuOTNDNy44NyA2LjM0IDcuNTI5OTkgNi42OCA3LjEyIDYuNjhDNi43MSA2LjY4IDYuMzcgNi4zNCA2LjM3IDUuOTNWNC43NUM2LjM3IDMuMjMgNy42IDIgOS4xMiAySDIwLjEyQzIxLjY0IDIgMjIuODcgMy4yMyAyMi44NyA0Ljc1VjEyLjc1QzIyLjg3IDE0LjI3IDIxLjY0IDE1LjUgMjAuMTIgMTUuNVpNMTEuMDEzOSAxOC4wNTA2VjE3LjE0OTRIMTEuODQ1N0MxMi4wMDY5IDE3LjE0OTQgMTIuMTM0MyAxNy4wMTY5IDEyLjEzNDMgMTYuODQ5VjE1LjY4MjdIMTNWMTYuODQ5QzEzIDE3LjUxMTYgMTIuNDgyMyAxOC4wNTA2IDExLjg0NTcgMTguMDUwNkgxMS4wMTM5Wk0xMi45OTE1IDEzLjM2NzlIMTIuMTI1OFYxMi4yMDE2QzEyLjEyNTggMTIuMDMzNyAxMS45OTg1IDExLjkwMTIgMTEuODM3MiAxMS45MDEySDExLjAwNTRWMTFIMTEuODM3MkMxMi40NzM4IDExIDEyLjk5MTUgMTEuNTM5IDEyLjk5MTUgMTIuMjAxNlYxMy4zNjc5Wk05LjM1ODggMTEuMDA4OFYxMS45MUg3LjcwMzcxVjExLjAwODhIOS4zNTg4Wk02LjA0ODYxIDExLjAwODhWMTEuOTFINS4yMTY4MkM1LjA1NTU2IDExLjkxIDQuOTI4MjQgMTIuMDQyNiA0LjkyODI0IDEyLjIxMDRWMTMuMzc2N0g0LjA2MjVWMTIuMjEwNEM0LjA2MjUgMTEuNTQ3OCA0LjU4MDI1IDExLjAwODggNS4yMTY4MiAxMS4wMDg4SDYuMDQ4NjFaTTEwLjY0MDQgMTYuMDI3M1YxNS4xNTI2QzEwLjY0MDQgMTQuMzgzOSA4LjcwNTI1IDEzLjc0NzggNi4zMjAyMiAxMy43NDc4QzMuOTM1MTkgMTMuNzQ3OCAyIDE0LjM3NTEgMiAxNS4xNTI2VjE2LjAyNzNDMiAxNi43OTYgMy45MzUxOSAxNy40MzIxIDYuMzIwMjIgMTcuNDMyMUM4LjcwNTI1IDE3LjQzMjEgMTAuNjQwNCAxNi44MDQ4IDEwLjY0MDQgMTYuMDI3M1pNMTAuNjQwNCAyMC41OTUyVjE3Ljg3MzlDOS43MTUyOCAxOC41MSA4LjAwOTI2IDE4LjgwMTYgNi4zMjAyMiAxOC44MDE2QzQuNjMxMTcgMTguODAxNiAyLjkyNTE1IDE4LjUxIDIgMTcuODczOVYyMC41OTUyQzIgMjEuMzYzOSAzLjkzNTE5IDIyIDYuMzIwMjIgMjJDOC43MDUyNSAyMiAxMC42NDA0IDIxLjM3MjcgMTAuNjQwNCAyMC41OTUyWk0xNy4xNyA1SDE5LjM1SDE5LjM2QzE5LjY1IDUgMTkuODggNS4yNCAxOS44OCA1LjUyVjcuNjlDMTkuODggNy45OCAxOS42NSA4LjIxIDE5LjM2IDguMjFDMTkuMDcgOC4yMSAxOC44NCA3Ljk3IDE4Ljg0IDcuNjlWNi43NUwxNy4zMyA4LjIzVjguMjVMMTQuNzggMTAuNzdIMTUuNzFDMTYgMTAuNzcgMTYuMjMgMTEgMTYuMjMgMTEuMjlDMTYuMjMgMTEuNTggMTUuOTkgMTEuODEgMTUuNzEgMTEuODFIMTMuNTNDMTMuMjQgMTEuODEgMTMuMDEgMTEuNTcgMTMuMDEgMTEuMjlWOS4xMUMxMy4wMSA4LjgyIDEzLjI0IDguNTkgMTMuNTMgOC41OUMxMy44MiA4LjU5IDE0LjA1IDguODMgMTQuMDUgOS4xMVYxMC4wNUwxNi42IDcuNTNWNy41MUwxOC4xIDYuMDRIMTcuMTdDMTYuODggNi4wNCAxNi42NSA1LjgxIDE2LjY1IDUuNTJDMTYuNjUgNS4yMyAxNi44OSA1IDE3LjE3IDVaIiBmaWxsPSIjMUQyNjMyIi8+Cjwvc3ZnPgo="),
		Technology:  extutil.Ptr("Kubernetes"),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType: StatefulSetTargetType,
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label:       "statefulSet",
					Description: extutil.Ptr("Find statefulSet by cluster, namespace and statefulSet"),
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
