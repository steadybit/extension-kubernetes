package extcommon

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/client"
	"reflect"
	"slices"
)

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

		forward := slices.Index(types, eventType) >= 0
		log.Warn().Type("type", event).Bool("forward", forward).Strs("types", s).Msg("resource event")
		if forward {
			out <- struct{}{}
		}
	}
}
