// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package main

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"net/http"
)

func main() {
	extlogging.InitZeroLog()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))
	extdeployment.RegisterDeploymentRolloutRestartAttackHandlers()

	port := 8088
	log.Log().Msgf("Starting extension-kubernetes server on port %d. Get started via /", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start extension-kubernetes server on port %d", port)
	}
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
		},
	}
}
