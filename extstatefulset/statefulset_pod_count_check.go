// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extstatefulset

import (
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
)

func NewStatefulSetPodCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.PodCountCheckState] {
	return &extcommon.PodCountCheckAction{
		Client:     k8s,
		ActionId:   StatefulSetPodCountCheckActionId,
		TargetType: StatefulSetTargetType,
		SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "default",
				Description: extutil.Ptr("Find statefulSet by cluster, namespace and statefulSet"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.statefulset=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.statefulset"][0]
		},
		GetDesiredAndCurrentPodCount: func(k8s *client.Client, namespace string, target string) (*int32, int32, error) {
			d := k8s.StatefulSetByNamespaceAndName(namespace, target)
			if d == nil {
				return nil, 0, extension_kit.ToError(fmt.Sprintf("StatefulSet %s not found.", target), nil)
			}
			return d.Spec.Replicas, d.Status.ReadyReplicas, nil
		},
	}
}
