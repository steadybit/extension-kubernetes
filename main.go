// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
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
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)

	exthealth.SetReady(false)
	exthealth.StartProbes(8089)

	client.PrepareClient(stopCh)

	extconfig.ParseConfiguration()
	extconfig.ValidateConfiguration()

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))

	action_kit_sdk.RegisterAction(extdeployment.NewDeploymentRolloutRestartAction())
	action_kit_sdk.RegisterAction(extdeployment.NewCheckDeploymentRolloutStatusAction())
	action_kit_sdk.RegisterAction(extdeployment.NewPodCountCheckAction())
	action_kit_sdk.RegisterAction(extdeployment.NewPodCountMetricsAction())
	action_kit_sdk.RegisterAction(extnode.NewNodeCountCheckAction())
	action_kit_sdk.RegisterAction(extevents.NewK8sEventsAction())

	extdeployment.RegisterAttributeDescriptionHandlers()
	extdeployment.RegisterDeploymentDiscoveryHandlers()
	extcontainer.RegisterContainerDiscoveryHandlers()
	extcluster.RegisterClusterDiscoveryHandlers()

	action_kit_sdk.InstallSignalHandler()

	action_kit_sdk.RegisterCoverageEndpoints()

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
		ActionList: action_kit_sdk.GetActionList(),
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
			TargetEnrichmentRules: []discovery_kit_api.DescribingEndpointReference{
				{
					Method: "GET",
					Path:   "/container/discovery/rules/k8s-container-to-container",
				},
				{
					Method: "GET",
					Path:   "/container/discovery/rules/k8s-container-to-host",
				},
				{
					Method: "GET",
					Path:   "/deployment/discovery/rules/k8s-deployment-to-container",
				},
				{
					Method: "GET",
					Path:   "/deployment/discovery/rules/container-to-k8s-deployment",
				},
			},
		},
	}
}
