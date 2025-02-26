// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extdeployment

import (
	"context"
	"fmt"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/steadybit/extension-kubernetes/extcommon"
	"github.com/steadybit/extension-kubernetes/extconfig"
	"github.com/steadybit/extension-kubernetes/extnamespace"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
	"reflect"
	"time"
)

type deploymentDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber          = (*deploymentDiscovery)(nil)
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*deploymentDiscovery)(nil)
)

func NewDeploymentDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &deploymentDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s,
		reflect.TypeOf(corev1.Pod{}),
		reflect.TypeOf(appsv1.Deployment{}),
		reflect.TypeOf(autoscalingv2.HorizontalPodAutoscaler{}),
		reflect.TypeOf(corev1.Service{}),
	)
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, 5*time.Second),
	)
}

func (d *deploymentDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: DeploymentTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *deploymentDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       DeploymentTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Deployment", Other: "Kubernetes Deployments"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr("data:image/svg+xml,%3Csvg%20width%3D%2224%22%20height%3D%2224%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%3E%0A%3Cpath%20d%3D%22M10.4478%202.65625C11.2739%202.24209%2012.2447%202.23174%2013.0794%202.62821L19.2871%205.57666C20.3333%206.07356%2021%207.12832%2021%208.28652V15.7134C21%2016.8717%2020.3333%2017.9264%2019.2871%2018.4233L13.0794%2021.3718C12.2447%2021.7682%2011.2739%2021.7579%2010.4478%2021.3437L4.65545%2018.4397L5.55182%2016.6518L11.3441%2019.5558C11.6195%2019.6939%2011.9431%2019.6973%2012.2214%2019.5652L18.429%2016.6167C18.7778%2016.4511%2019%2016.0995%2019%2015.7134V8.28652C19%207.90045%2018.7778%207.54887%2018.429%207.38323L12.2214%204.43479C11.9431%204.30263%2011.6195%204.30608%2011.3441%204.44413L5.55182%207.34814C5.21357%207.51773%205%207.8637%205%208.24208V15.7579C5%2016.1363%205.21357%2016.4822%205.55182%2016.6518L4.65545%2018.4397C3.6407%2017.931%203%2016.893%203%2015.7579V8.24208C3%207.10694%203.6407%206.06901%204.65545%205.56026L10.4478%202.65625Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3Cpath%20d%3D%22M11.1377%207.16465C11.5966%206.95033%2012.1359%206.94497%2012.5997%207.15014L16.0484%208.67595C16.6296%208.9331%2017%209.47893%2017%2010.0783V13.9217C17%2014.5211%2016.6296%2015.0669%2016.0484%2015.324L12.5997%2016.8499C12.1359%2017.055%2011.5966%2017.0497%2011.1377%2016.8353L7.9197%2015.3325C7.35594%2015.0693%207%2014.5321%207%2013.9447V10.0553C7%209.46787%207.35594%208.93074%207.9197%208.66747L11.1377%207.16465Z%22%20fill%3D%22%231D2632%22%2F%3E%0A%3C%2Fsvg%3E%0A"),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.deployment"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.deployment",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *deploymentDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	deployments := d.k8s.Deployments()

	filteredDeployments := make([]*appsv1.Deployment, 0, len(deployments))
	for _, deployment := range deployments {
		if client.IsExcludedFromDiscovery(deployment.ObjectMeta) {
			continue
		}
		filteredDeployments = append(filteredDeployments, deployment)
	}

	targets := make([]discovery_kit_api.Target, len(filteredDeployments))

	nodes := d.k8s.Nodes()
	for i, deployment := range filteredDeployments {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, deployment.Namespace, deployment.Name)
		attributes := map[string][]string{
			"k8s.namespace":                    {deployment.Namespace},
			"k8s.deployment":                   {deployment.Name},
			"k8s.workload-type":                {"deployment"},
			"k8s.workload-owner":               {deployment.Name},
			"k8s.cluster-name":                 {extconfig.Config.ClusterName},
			"k8s.distribution":                 {d.k8s.Distribution},
			"k8s.deployment.min-ready-seconds": {fmt.Sprintf("%d", deployment.Spec.MinReadySeconds)},
		}
		if deployment.Spec.Replicas != nil {
			attributes["k8s.specification.replicas"] = []string{fmt.Sprintf("%d", *deployment.Spec.Replicas)}
		}
		for key, value := range deployment.ObjectMeta.Labels {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.deployment.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}
		extnamespace.AddNamespaceLabels(d.k8s, deployment.Namespace, attributes)

		for key, value := range extcommon.GetPodBasedAttributes("deployment", deployment.ObjectMeta, d.k8s.PodsByLabelSelector(deployment.Spec.Selector, deployment.Namespace), nodes) {
			attributes[key] = value
		}
		for key, value := range extcommon.GetServiceNames(d.k8s.ServicesMatchingToPodLabels(deployment.Namespace, deployment.Spec.Template.Labels)) {
			attributes[key] = value
		}

		var hpa *autoscalingv2.HorizontalPodAutoscaler
		if d.k8s.Permissions().CanReadHorizontalPodAutoscalers() {
			hpa = d.k8s.HorizontalPodAutoscalerByNamespaceAndDeployment(deployment.Namespace, deployment.Name)
		}
		for key, value := range extcommon.GetKubeScoreForDeployment(deployment, d.k8s.ServicesMatchingToPodLabels(deployment.Namespace, deployment.Spec.Template.Labels), hpa) {
			attributes[key] = value
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: DeploymentTargetType,
			Label:      deployment.Name,
			Attributes: attributes,
		}
	}
	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesDeployment), nil
}

func (d *deploymentDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getDeploymentToContainerEnrichmentRule(),
	}
}

func getDeploymentToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-deployment-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: DeploymentTargetType,
			Selector: map[string]string{
				"k8s.container.id.stripped": "${dest.container.id.stripped}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.extension_container.container",
			Selector: map[string]string{
				"container.id.stripped": "${src.k8s.container.id.stripped}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.deployment.label.",
			},
			{
				Matcher: discovery_kit_api.Regex,
				Name:    "^k8s\\.label\\.(?!topology).*",
			},
		},
	}
}
