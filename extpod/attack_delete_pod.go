// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extpod

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/extcommon"
)

func NewDeletePodAction() action_kit_sdk.Action[extcommon.KubectlActionState] {
	return &extcommon.KubectlAction{
		Description:  getDeletePodDescription(),
		OptsProvider: deletePod(),
	}
}

func getDeletePodDescription() action_kit_api.ActionDescription {
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
		TimeControl: action_kit_api.TimeControlInternal,
		Kind:        action_kit_api.Attack,
		Parameters:  []action_kit_api.ActionParameter{},
		Prepare:     action_kit_api.MutatingEndpointReference{},
		Start:       action_kit_api.MutatingEndpointReference{},
		Status:      &action_kit_api.MutatingEndpointReferenceWithCallInterval{},
		Stop:        &action_kit_api.MutatingEndpointReference{},
	}
}

func deletePod() extcommon.KubectlOptsProvider {
	return func(ctx context.Context, request action_kit_api.PrepareActionRequestBody) (*extcommon.KubectlOpts, error) {
		namespace := request.Target.Attributes["k8s.namespace"][0]
		pod := request.Target.Attributes["k8s.pod.name"][0]

		command := []string{"kubectl",
			"delete",
			"pod",
			"--namespace",
			namespace,
			pod}

		return &extcommon.KubectlOpts{
			Command:         command,
			RollbackCommand: nil,
			LogTargetType:   "pod",
			LogTargetName:   pod,
			LogActionName:   "delete pod",
		}, nil
	}
}
