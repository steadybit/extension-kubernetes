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

func TriggerOnKubernetesResourceChange(k8s *client.Client, t ...reflect.Type) chan struct{} {
	chRefresh := make(chan struct{})
	chNotification := make(chan interface{})

	k8s.Notify(chNotification)
	go triggerNotificationsForType(chNotification, chRefresh, t...)

	return chRefresh
}

func triggerNotificationsForType(in <-chan interface{}, out chan<- struct{}, types ...reflect.Type) {
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
			// For unstructured resources, check if it's an Argo Rollout
			gvk := obj.GetObjectKind().GroupVersionKind()
			if gvk == ArgoRolloutGVK {
				forward = true
			}
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
