// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kubernetes/extdeployment"
)

func main() {
	extlogging.InitZeroLog()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))
	extdeployment.RegisterDeploymentRolloutRestartAttackHandlers()
	extdeployment.RegisterDeploymentRolloutStatusCheckHandlers()
	extdeployment.RegisterDeploymentDiscoveryHandlers()

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8088,
	})
}

type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		ActionList: action_kit_api.ActionList{
			Actions: []action_kit_api.DescribingEndpointReference{
				{
					"GET",
					"/deployment/attack/rollout-restart",
				},
				{
					"GET",
					"/deployment/check/rollout-status",
				},
			},
		},
		DiscoveryList: discovery_kit_api.DiscoveryList{
			Discoveries: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery",
				},
			},
			TargetTypes: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery/target-description",
				},
			},
			TargetAttributes: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery/attribute-descriptions",
				},
			},
		},
	}
}
