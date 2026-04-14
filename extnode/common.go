// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extnode

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
)

const (
	NodeTargetType         = "com.steadybit.extension_kubernetes.kubernetes-node"
	DrainNodeActionId      = "com.steadybit.extension_kubernetes.drain_node"
	TaintNodeActionId      = "com.steadybit.extension_kubernetes.taint_node"
	NodeCountCheckActionId = "com.steadybit.extension_kubernetes.node_count_check"
)

var (
	targetSelectionTemplates = action_kit_api.TargetSelection{
		TargetType: NodeTargetType,
		SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
			{
				Label:       "host name",
				Description: new("Find node by its name"),
				Query:       "host.hostname=\"\"",
			},
			{
				Label:       "kubernetes deployment",
				Description: new("Find node by cluster, namespace and deployment."),
				Query:       "k8s.cluster-name=\"\" and k8s.namespace=\"\" and k8s.deployment=\"\"",
			},
			{
				Label:       "kubernetes statefulset",
				Description: new("Find node by cluster, namespace and statefulset."),
				Query:       "k8s.cluster-name=\"\" and k8s.namespace=\"\" and k8s.statefulset=\"\"",
			},
			{
				Label:       "kubernetes daemonset",
				Description: new("Find node by cluster, namespace and daemonset."),
				Query:       "k8s.cluster-name=\"\" and k8s.namespace=\"\" and k8s.daemonset=\"\"",
			},
		}),
	}
)
