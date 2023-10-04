// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extdeployment

import (
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
)

func RegisterAttributeDescriptionHandlers() {
	exthttp.RegisterHttpHandler("/attribute-descriptions", exthttp.GetterAsHandler(getAttributeDescriptions))
}

func getAttributeDescriptions() discovery_kit_api.AttributeDescriptions {
	return discovery_kit_api.AttributeDescriptions{
		Attributes: []discovery_kit_api.AttributeDescription{
			{
				Attribute: "k8s.container.name",
				Label: discovery_kit_api.PluralLabel{
					One:   "Container name",
					Other: "Container names",
				},
			},
			{
				Attribute: "k8s.namespace",
				Label: discovery_kit_api.PluralLabel{
					One:   "Namespace name",
					Other: "Namespace names",
				},
			},
			{
				Attribute: "k8s.cluster-name",
				Label: discovery_kit_api.PluralLabel{
					One:   "Cluster name",
					Other: "Cluster names",
				},
			},
			{
				Attribute: "k8s.deployment",
				Label: discovery_kit_api.PluralLabel{
					One:   "Deployment name",
					Other: "Deployment names",
				},
			},
			{
				Attribute: "k8s.statefulset",
				Label: discovery_kit_api.PluralLabel{
					One:   "StatefulSet name",
					Other: "StatefulSet names",
				},
			},
			{
				Attribute: "k8s.daemonset",
				Label: discovery_kit_api.PluralLabel{
					One:   "DaemonSet name",
					Other: "DaemonSet names",
				},
			},
		},
	}
}
