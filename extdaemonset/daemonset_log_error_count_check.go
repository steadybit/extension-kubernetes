// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extdaemonset

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	corev1 "k8s.io/api/core/v1"
)

func NewDaemonSetLogErrorCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.LogErrorCountCheckState] {
	return &extcommon.LogErrorCountCheckAction{
		Client:          k8s,
		ActionId:        DaemonSetLogErrorCountCheckActionId,
		TargetType:      DaemonSetTargetType,
		TargetTypeLabel: "DaemonSet",
		SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "daemonset",
				Description: new("Find daemonset by cluster, namespace and daemonset"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.daemonset=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.daemonset"][0]
		},
		GetPods: func(k8s *client.Client, namespace string, target string) ([]*corev1.Pod, error) {
			w := k8s.DaemonSetByNamespaceAndName(namespace, target)
			if w == nil {
				return nil, extension_kit.ToError(fmt.Sprintf("DaemonSet %s not found.", target), nil)
			}
			return k8s.PodsByLabelSelector(w.Spec.Selector, namespace), nil
		},
		MetricLabelKey: "k8s.daemonset",
	}
}
