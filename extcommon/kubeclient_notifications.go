package extcommon

import (
	"reflect"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/v2/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ArgoRolloutGVK is the GroupVersionKind for Argo Rollouts
var ArgoRolloutGVK = schema.GroupVersionKind{
	Group:   "argoproj.io",
	Version: "v1alpha1",
	Kind:    "Rollout",
}

// triggerableUnstructuredGVKs lists the dynamic (unstructured) resource kinds whose change events
// should trigger a low-latency discovery refresh. Unstructured objects all share the same Go type,
// so they cannot be matched by reflect.Type like typed resources — we match on their GVK instead.
var triggerableUnstructuredGVKs = []schema.GroupVersionKind{
	ArgoRolloutGVK,
	{Group: client.GatewayNetworkingGroup, Version: "v1", Kind: "HTTPRoute"},
	{Group: client.GatewayNetworkingGroup, Version: "v1", Kind: "Gateway"},
	{Group: client.GatewayNetworkingGroup, Version: "v1", Kind: "GatewayClass"},
}

func TriggerOnKubernetesResourceChange(k8s *client.Client, t ...reflect.Type) chan struct{} {
	chRefresh := make(chan struct{})
	chNotification := make(chan any)

	k8s.Notify(chNotification)
	go triggerNotificationsForType(chNotification, chRefresh, t...)

	return chRefresh
}

func triggerNotificationsForType(in <-chan any, out chan<- struct{}, types ...reflect.Type) {
	var s []string
	for _, r := range types {
		s = append(s, r.String())
	}

	for event := range in {
		eventType := reflect.TypeOf(event)
		if eventType.Kind() == reflect.Pointer {
			eventType = eventType.Elem()
		}

		forward := false
		if obj, ok := event.(*unstructured.Unstructured); ok {
			// Unstructured resources (CRDs) can't be matched by reflect.Type, so match on GVK.
			gvk := obj.GetObjectKind().GroupVersionKind()
			forward = slices.Contains(triggerableUnstructuredGVKs, gvk)
		} else {
			forward = slices.Index(types, eventType) >= 0
		}

		log.Trace().
			Str("eventType", eventType.String()).
			Bool("forward", forward).
			Strs("types", s).
			Msg("resource event")

		if forward {
			out <- struct{}{}
		}
	}
}
