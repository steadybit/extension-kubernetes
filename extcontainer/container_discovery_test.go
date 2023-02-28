// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extcontainer

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/client"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"testing"
	"time"
)

// TODO remove
func Test_informerStrategy(t *testing.T) {
	// Given
	_, clientset := getTestClient()

	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Pods()
	lister := informer.Lister()
	stopper := make(chan struct{})
	defer runtime.HandleCrash()
	defer close(stopper)
	go factory.Start(stopper)
	if !cache.WaitForCacheSync(stopper,
		informer.Informer().HasSynced,
	) {
		log.Fatal().Msg("Timed out waiting for caches to sync")
	}

	// When
	_, err := clientset.CoreV1().
		Pods("default").
		Create(context.Background(), &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shop",
				Namespace: "default",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "Always",
					},
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	// Then
	time.Sleep(time.Second * 2)
	pods, err := lister.Pods("").List(labels.Everything())
	require.NoError(t, err)
	require.Len(t, pods, 1)
}

func Test_getDiscoveredContainer(t *testing.T) {
	// Given
	client, clientset := getTestClient()
	_, err := clientset.CoreV1().
		Pods("default").
		Create(context.Background(), &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shop",
				Namespace: "default",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:            "nginx",
						Image:           "nginx",
						ImagePullPolicy: "Always",
					},
				},
			},
		}, metav1.CreateOptions{})
	require.NoError(t, err)

	// When
	time.Sleep(time.Second * 2)
	targets := getDiscoveredContainerTargets(client)

	// Then
	require.Len(t, targets, 1)
}

func getTestClient() (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	return client.CreateClient(clientset), clientset
}
