// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extstatefulset

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	corev1 "k8s.io/api/core/v1"
)

func NewStatefulSetLogErrorCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.LogErrorCountCheckState] {
	return &extcommon.LogErrorCountCheckAction{
		Client:          k8s,
		ActionId:        StatefulSetLogErrorCountCheckActionId,
		TargetType:      StatefulSetTargetType,
		TargetTypeLabel: "StatefulSet",
		SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "statefulset",
				Description: new("Find statefulset by cluster, namespace and statefulset"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.statefulset=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.statefulset"][0]
		},
		GetPods: func(k8s *client.Client, namespace string, target string) ([]*corev1.Pod, error) {
			w := k8s.StatefulSetByNamespaceAndName(namespace, target)
			if w == nil {
				return nil, extension_kit.ToError(fmt.Sprintf("StatefulSet %s not found.", target), nil)
			}
			return k8s.PodsByLabelSelector(w.Spec.Selector, namespace), nil
		},
		MetricLabelKey: "k8s.statefulset",
	}
}
