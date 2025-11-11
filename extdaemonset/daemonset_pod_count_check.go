// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdaemonset

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewDaemonSetPodCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.PodCountCheckState] {
	return &extcommon.PodCountCheckAction{
		Client:          k8s,
		ActionId:        DaemonSetPodCountCheckActionId,
		TargetType:      DaemonSetTargetType,
		TargetTypeLabel: "DaemonSet",
		SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "daemonset",
				Description: extutil.Ptr("Find daemonSet by cluster, namespace and daemonSet"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.daemonset=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.daemonset"][0]
		},
		GetDesiredAndCurrentPodCount: func(k8s *client.Client, namespace string, target string) (*int32, int32, error) {
			d := k8s.DaemonSetByNamespaceAndName(namespace, target)
			if d == nil {
				return nil, 0, extension_kit.ToError(fmt.Sprintf("DaemonSet %s not found.", target), nil)
			}
			return extutil.Ptr(d.Status.DesiredNumberScheduled), d.Status.NumberReady, nil
		},
	}
}
