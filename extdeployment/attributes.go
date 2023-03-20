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
					One:   "container name",
					Other: "container names",
				},
			},
			{
				Attribute: "k8s.namespace",
				Label: discovery_kit_api.PluralLabel{
					One:   "namespace name",
					Other: "namespace names",
				},
			},
			{
				Attribute: "k8s.cluster-name",
				Label: discovery_kit_api.PluralLabel{
					One:   "cluster name",
					Other: "cluster names",
				},
			},
			{
				Attribute: "k8s.deployment",
				Label: discovery_kit_api.PluralLabel{
					One:   "deployment name",
					Other: "deployment names",
				},
			},
		},
	}
}
