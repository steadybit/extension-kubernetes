package extargorollout

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extcommon"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type rolloutDiscovery struct {
	k8s *client.Client
}

var (
	_ discovery_kit_sdk.TargetDescriber          = (*rolloutDiscovery)(nil)
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*rolloutDiscovery)(nil)
)

func NewRolloutDiscovery(k8s *client.Client) discovery_kit_sdk.TargetDiscovery {
	discovery := &rolloutDiscovery{k8s: k8s}
	chRefresh := extcommon.TriggerOnKubernetesResourceChange(k8s,
		reflect.TypeOf(corev1.Pod{}),
		reflect.TypeOf(unstructured.Unstructured{}),
		reflect.TypeOf(autoscalingv2.HorizontalPodAutoscaler{}),
		reflect.TypeOf(corev1.Service{}),
	)
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsTrigger(context.Background(), chRefresh, 5*time.Second),
	)
}

func (d *rolloutDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: ArgoRolloutTargetType,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("30s"),
		},
	}
}

func (d *rolloutDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       ArgoRolloutTargetType,
		Label:    discovery_kit_api.PluralLabel{One: "Kubernetes Argo Rollout", Other: "Kubernetes Argo Rollouts"},
		Category: extutil.Ptr("Kubernetes"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(ArgoRolloutIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "k8s.argo-rollout"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "k8s.argo-rollout",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *rolloutDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	rollouts := d.k8s.ArgoRollouts()

	filteredRollouts := make([]*unstructured.Unstructured, 0, len(rollouts))
	for _, rollout := range rollouts {
		if client.IsExcludedFromDiscovery(metav1.ObjectMeta{
			Name:        rollout.GetName(),
			Namespace:   rollout.GetNamespace(),
			Annotations: rollout.GetAnnotations(),
			Labels:      rollout.GetLabels(),
		}) {
			continue
		}
		filteredRollouts = append(filteredRollouts, rollout)
	}

	targets := make([]discovery_kit_api.Target, len(filteredRollouts))
	nodes := d.k8s.Nodes()

	for i, rollout := range filteredRollouts {
		targetName := fmt.Sprintf("%s/%s/%s", extconfig.Config.ClusterName, rollout.GetNamespace(), rollout.GetName())
		attributes := map[string][]string{
			"k8s.namespace":      {rollout.GetNamespace()},
			"k8s.argo-rollout":   {rollout.GetName()},
			"k8s.workload-type":  {"argo-rollout"},
			"k8s.workload-owner": {rollout.GetName()},
			"k8s.cluster-name":   {extconfig.Config.ClusterName},
			"k8s.distribution":   {d.k8s.Distribution},
		}

		// Get replicas from spec
		if replicas, found, err := unstructured.NestedInt64(rollout.Object, "spec", "replicas"); err == nil && found {
			attributes["k8s.specification.replicas"] = []string{fmt.Sprintf("%d", replicas)}
		}

		// Get min-ready-seconds from spec
		if minReadySeconds, found, err := unstructured.NestedInt64(rollout.Object, "spec", "minReadySeconds"); err == nil && found {
			attributes["k8s.specification.min-ready-seconds"] = []string{fmt.Sprintf("%d", minReadySeconds)}
		}

		// Add labels
		for key, value := range rollout.GetLabels() {
			if !slices.Contains(extconfig.Config.LabelFilter, key) {
				attributes[fmt.Sprintf("k8s.argo-rollout.label.%v", key)] = []string{value}
				attributes[fmt.Sprintf("k8s.label.%v", key)] = []string{value}
			}
		}

		extcommon.AddNamespaceLabels(attributes, d.k8s, rollout.GetNamespace())

		// Get pod template labels
		if podTemplate, found, err := unstructured.NestedMap(rollout.Object, "spec", "template", "metadata", "labels"); err == nil && found {
			// Convert map[string]interface{} to map[string]string
			podTemplateLabels := make(map[string]string)
			for k, v := range podTemplate {
				if str, ok := v.(string); ok {
					podTemplateLabels[k] = str
				}
			}

			// Get pods by label selector
			pods := d.k8s.PodsByLabelSelector(&metav1.LabelSelector{
				MatchLabels: podTemplateLabels,
			}, rollout.GetNamespace())

			// Add pod-based attributes
			for key, value := range extcommon.GetPodBasedAttributes("argo-rollout", metav1.ObjectMeta{
				Name:        rollout.GetName(),
				Namespace:   rollout.GetNamespace(),
				Annotations: rollout.GetAnnotations(),
				Labels:      rollout.GetLabels(),
			}, pods, nodes) {
				attributes[key] = value
			}

			// Add service names
			for key, value := range extcommon.GetServiceNames(d.k8s.ServicesMatchingToPodLabels(rollout.GetNamespace(), podTemplateLabels)) {
				attributes[key] = value
			}
		}

		targets[i] = discovery_kit_api.Target{
			Id:         targetName,
			TargetType: ArgoRolloutTargetType,
			Label:      rollout.GetName(),
			Attributes: attributes,
		}
	}

	return discovery_kit_commons.ApplyAttributeExcludes(targets, extconfig.Config.DiscoveryAttributesExcludesArgoRollout), nil
}

func (d *rolloutDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getRolloutToContainerEnrichmentRule(),
	}
}

func getRolloutToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_kubernetes.kubernetes-argo-rollout-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: ArgoRolloutTargetType,
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
				Name:    "k8s.argo-rollout.label.",
			},
			{
				Matcher: discovery_kit_api.Regex,
				Name:    "^k8s\\.label\\.(?!topology).*",
			},
		},
	}
}
