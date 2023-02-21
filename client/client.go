package client

import (
	"errors"
	"flag"
	"fmt"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerAppsv1 "k8s.io/client-go/listers/apps/v1"
	listerCorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

var K8S *Client

type Client struct {
	daemonsetsLister     listerAppsv1.DaemonSetLister
	daemonsetsInformer   cache.SharedIndexInformer
	deploymentsLister    listerAppsv1.DeploymentLister
	deploymentsInformer  cache.SharedIndexInformer
	podsLister           listerCorev1.PodLister
	podsInformer         cache.SharedIndexInformer
	replicaSetsLister    listerAppsv1.ReplicaSetLister
	replicaSetsInformer  cache.SharedIndexInformer
	statefulSetsLister   listerAppsv1.StatefulSetLister
	statefulSetsInformer cache.SharedIndexInformer
}

func (c Client) Pods() []*corev1.Pod {
	pods, err := c.podsLister.Pods("").List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching pods")
		return []*corev1.Pod{}
	}
	return pods
}

func (c Client) Deployments() []*appsv1.Deployment {
	deployments, err := c.deploymentsLister.Deployments("").List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching deployments")
		return []*appsv1.Deployment{}
	}
	return deployments
}

func (c Client) DaemonSetByNamespaceAndName(namespace string, name string) *appsv1.DaemonSet {
	key := fmt.Sprintf("%s/%s", namespace, name)
	item, _, err := c.daemonsetsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		log.Error().Err(err).Msgf("Error during lookup of DaemonSet %s/%s", namespace, name)
	}
	return item.(*appsv1.DaemonSet)
}
func (c Client) DeploymentByNamespaceAndName(namespace string, name string) *appsv1.Deployment {
	key := fmt.Sprintf("%s/%s", namespace, name)
	item, _, err := c.deploymentsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		log.Error().Err(err).Msgf("Error during lookup of Deployment %s/%s", namespace, name)
	}
	return item.(*appsv1.Deployment)
}
func (c Client) ReplicaSetByNamespaceAndName(namespace string, name string) *appsv1.ReplicaSet {
	key := fmt.Sprintf("%s/%s", namespace, name)
	item, _, err := c.replicaSetsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		log.Error().Err(err).Msgf("Error during lookup of ReplicaSet %s/%s", namespace, name)
	}
	return item.(*appsv1.ReplicaSet)
}
func (c Client) StatefulSetByNamespaceAndName(namespace string, name string) *appsv1.StatefulSet {
	key := fmt.Sprintf("%s/%s", namespace, name)
	item, _, err := c.statefulSetsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		log.Error().Err(err).Msgf("Error during lookup of StatefulSet %s/%s", namespace, name)
	}
	return item.(*appsv1.StatefulSet)
}

func PrepareClient() {
	clientset := createClientset()

	// stop signal for the informer
	stopper := make(chan struct{})
	defer close(stopper)

	factory := informers.NewSharedInformerFactory(clientset, 0)

	// DeploymentsInformer.SetTransform() // TODO - Check whether we could use transformers to remove stuff --> save RAM?
	daemonsets := factory.Apps().V1().DaemonSets()
	daemosetsInformer := daemonsets.Informer()
	deployments := factory.Apps().V1().Deployments()
	deploymentsInformer := deployments.Informer()
	pods := factory.Core().V1().Pods()
	podsInformer := pods.Informer()
	replicaSets := factory.Apps().V1().ReplicaSets()
	replicaSetsInformer := replicaSets.Informer()
	statefulSets := factory.Apps().V1().StatefulSets()
	statefulSetsInformer := statefulSets.Informer()

	defer runtime.HandleCrash()

	go factory.Start(stopper)

	log.Info().Msgf("Start cache sync.")
	if !cache.WaitForCacheSync(stopper,
		daemosetsInformer.HasSynced,
		deploymentsInformer.HasSynced,
		podsInformer.HasSynced,
		replicaSetsInformer.HasSynced,
		statefulSetsInformer.HasSynced,
	) {
		log.Fatal().Msg("Timed out waiting for caches to sync")
	}
	log.Info().Msgf("Caches synced.")

	K8S = &Client{
		daemonsets.Lister(),
		daemosetsInformer,
		deployments.Lister(),
		deploymentsInformer,
		pods.Lister(),
		podsInformer,
		replicaSets.Lister(),
		replicaSetsInformer,
		statefulSets.Lister(),
		statefulSetsInformer,
	}
}

func createClientset() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Info().Msgf("Extension is running inside a cluster, config found")
	} else if errors.Is(err, rest.ErrNotInCluster) {
		log.Info().Msgf("Extension is not running inside a cluster, try local .kube config")
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	if err != nil {
		log.Fatal().Err(err).Msgf("Could not find kubernetes config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not create kubernetes client")
	}

	info, err := clientset.ServerVersion()
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not fetch server version.")
	}

	log.Info().Msgf("Cluster connected! Kubernetes Server Version %+v", info)

	return clientset
}
