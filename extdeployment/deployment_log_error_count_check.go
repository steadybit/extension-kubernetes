// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extdeployment

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	corev1 "k8s.io/api/core/v1"
)

func NewDeploymentLogErrorCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.LogErrorCountCheckState] {
	return &extcommon.LogErrorCountCheckAction{
		Client:          k8s,
		ActionId:        DeploymentLogErrorCountCheckActionId,
		TargetType:      DeploymentTargetType,
		TargetTypeLabel: "Deployment",
		SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "deployment",
				Description: new("Find deployment by cluster, namespace and deployment"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.deployment"][0]
		},
		GetPods: func(k8s *client.Client, namespace string, target string) ([]*corev1.Pod, error) {
			w := k8s.DeploymentByNamespaceAndName(namespace, target)
			if w == nil {
				return nil, extension_kit.ToError(fmt.Sprintf("Deployment %s not found.", target), nil)
			}
			return k8s.PodsByLabelSelector(w.Spec.Selector, namespace), nil
		},
		MetricLabelKey: "k8s.deployment",
	}
}
