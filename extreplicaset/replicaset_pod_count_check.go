// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extreplicaset

import (
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewReplicaSetPodCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.PodCountCheckState] {
	return &extcommon.PodCountCheckAction{
		Client:          k8s,
		ActionId:        ReplicaSetPodCountCheckActionId,
		TargetType:      ReplicaSetTargetType,
		TargetTypeLabel: "ReplicaSet",
		SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "replicaset",
				Description: extutil.Ptr("Find replicaset by cluster, namespace and replicaset"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.replicaset=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.replicaset"][0]
		},
		GetDesiredAndCurrentPodCount: func(k8s *client.Client, namespace string, target string) (*int32, int32, error) {
			rs := k8s.ReplicaSetByNamespaceAndName(namespace, target)
			if rs == nil {
				return nil, 0, extension_kit.ToError(fmt.Sprintf("ReplicaSet %s not found.", target), nil)
			}
			return rs.Spec.Replicas, rs.Status.ReadyReplicas, nil
		},
	}
}