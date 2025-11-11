// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
)

func NewDeploymentPodCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.PodCountCheckState] {
	return &extcommon.PodCountCheckAction{
		Client:          k8s,
		ActionId:        DeploymentPodCountCheckActionId,
		TargetType:      DeploymentTargetType,
		TargetTypeLabel: "Deployment",
		SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "deployment",
				Description: extutil.Ptr("Find deployment by cluster, namespace and deployment"),
				Query:       "k8s.cluster-name=\"\" AND k8s.namespace=\"\" AND k8s.deployment=\"\"",
			},
		}),
		GetTarget: func(request action_kit_api.PrepareActionRequestBody) string {
			return request.Target.Attributes["k8s.deployment"][0]
		},
		GetDesiredAndCurrentPodCount: func(k8s *client.Client, namespace string, target string) (*int32, int32, error) {
			d := k8s.DeploymentByNamespaceAndName(namespace, target)
			if d == nil {
				return nil, 0, extension_kit.ToError(fmt.Sprintf("Deployment %s not found.", target), nil)
			}
			return d.Spec.Replicas, d.Status.ReadyReplicas, nil
		},
	}
}
