// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extstatefulset

import (
	"fmt"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	appsv1 "k8s.io/api/apps/v1"
)

func NewStatefulSetPodCountCheckAction(k8s *client.Client) action_kit_sdk.Action[extcommon.PodCountCheckState] {
	return &extcommon.PodCountCheckAction{
		Client:          k8s,
		ActionId:        StatefulSetPodCountCheckActionId,
		TargetType:      StatefulSetTargetType,
		TargetTypeLabel: "StatefulSet",
		SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "statefulSet",
				Description: new("Find statefulSet by cluster, namespace and statefulSet"),
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
		MetricLabelKey: "k8s.statefulset",
		GetPodCountMetrics: func(k8s *client.Client, namespace string, target string) (*extcommon.PodCountMetrics, error) {
			d := k8s.StatefulSetByNamespaceAndName(namespace, target)
			if d == nil {
				return nil, nil
			}
			return statefulSetPodCountMetrics(d), nil
		},
		Widget: action_kit_api.PredefinedWidget{
			Type:               action_kit_api.ComSteadybitWidgetPredefined,
			PredefinedWidgetId: "com.steadybit.widget.predefined.DeploymentReadinessWidget",
		},
	}
}

func statefulSetPodCountMetrics(d *appsv1.StatefulSet) *extcommon.PodCountMetrics {
	var desired int32
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	return &extcommon.PodCountMetrics{
		Desired:   desired,
		Current:   d.Status.Replicas,
		Ready:     d.Status.ReadyReplicas,
		Available: d.Status.AvailableReplicas,
	}
}
