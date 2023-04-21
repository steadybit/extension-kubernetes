// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcluster"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/steadybit/extension-kubernetes/extcontainer"
	"github.com/steadybit/extension-kubernetes/extdeployment"
	"github.com/steadybit/extension-kubernetes/extevents"
	"github.com/steadybit/extension-kubernetes/extnode"
)

func main() {
	stopCh := make(chan struct{})
	defer close(stopCh)
	extlogging.InitZeroLog()
	extbuild.PrintBuildInformation()

	exthealth.SetReady(false)
	exthealth.StartProbes(8089)

	client.PrepareClient(stopCh)

	extconfig.ParseConfiguration()
	extconfig.ValidateConfiguration()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))
	extdeployment.RegisterDeploymentRolloutRestartAttackHandlers()
	extdeployment.RegisterDeploymentRolloutStatusCheckHandlers()
	extdeployment.RegisterAttributeDescriptionHandlers()
	extdeployment.RegisterDeploymentDiscoveryHandlers()
	extcontainer.RegisterContainerDiscoveryHandlers()
	extevents.RegisterK8sEventsHandlers()
	extdeployment.RegisterPodCountMetricsHandlers()
	extdeployment.RegisterPodCountCheckHandlers()
	extnode.RegisterNodeCountCheckHandlers()
	extcluster.RegisterClusterDiscoveryHandlers()

	action_kit_sdk.InstallSignalHandler()

	exthealth.SetReady(true)

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
					Method: "GET",
					Path:   "/deployment/attack/rollout-restart",
				},
				{
					Method: "GET",
					Path:   "/deployment/check/rollout-status",
				},
				{
					Method: "GET",
					Path:   "/events",
				},
				{
					Method: "GET",
					Path:   "/pod-count/metrics",
				},
				{
					Method: "GET",
					Path:   "/pod-count/check",
				},
				{
					Method: "GET",
					Path:   "/node-count/check",
				},
			},
		},
		DiscoveryList: discovery_kit_api.DiscoveryList{
			Discoveries: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery",
				},
				{
					Method: "GET",
					Path:   "/container/discovery",
				},
				{
					Method: "GET",
					Path:   "/cluster/discovery",
				},
			},
			TargetTypes: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/deployment/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/container/discovery/target-description",
				},
				{
					Method: "GET",
					Path:   "/cluster/discovery/target-description",
				},
			},
			TargetAttributes: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/attribute-descriptions",
				},
			},
		},
	}
}
