// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kubernetes/extdeployment"
)

func main() {
	extlogging.InitZeroLog()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))
	extdeployment.RegisterDeploymentRolloutRestartAttackHandlers()
	extdeployment.RegisterDeploymentRolloutStatusCheckHandlers()

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8088,
	})
}

type ExtensionListResponse struct {
	Actions []action_kit_api.DescribingEndpointReference `json:"actions"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
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
	}
}
