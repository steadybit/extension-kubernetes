// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package client

import (
	"errors"
	"flag"
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerAppsv1 "k8s.io/client-go/listers/apps/v1"
	listerAutoscalingv2 "k8s.io/client-go/listers/autoscaling/v2"
	listerCorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var K8S *Client

type Client struct {
	Distribution string
	permissions  *PermissionCheckResult

	daemonSet struct {
		lister   listerAppsv1.DaemonSetLister
		informer cache.SharedIndexInformer
	}

	deployment struct {
		lister   listerAppsv1.DeploymentLister
		informer cache.SharedIndexInformer
	}

	pod struct {
		lister   listerCorev1.PodLister
		informer cache.SharedIndexInformer
	}

	namespace struct {
		lister   listerCorev1.NamespaceLister
		informer cache.SharedIndexInformer
	}

	replicaSet struct {
		lister   listerAppsv1.ReplicaSetLister
		informer cache.SharedIndexInformer
	}

	service struct {
		lister   listerCorev1.ServiceLister
		informer cache.SharedIndexInformer
	}

	statefulSet struct {
		lister   listerAppsv1.StatefulSetLister
		informer cache.SharedIndexInformer
	}

	event struct {
		informer cache.SharedIndexInformer
	}

	node struct {
		lister   listerCorev1.NodeLister
		informer cache.SharedIndexInformer
	}

	hpa struct {
		lister   listerAutoscalingv2.HorizontalPodAutoscalerLister
		informer cache.SharedIndexInformer
	}

	handlers struct {
		sync.Mutex
		l []chan<- interface{}
	}
	resourceEventHandler cache.ResourceEventHandlerFuncs
}

func (c *Client) Permissions() *PermissionCheckResult {
	return c.permissions
}

func (c *Client) Pods() []*corev1.Pod {
	pods, err := c.pod.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching pods")
		return []*corev1.Pod{}
	}
	return pods
}

func (c *Client) Namespaces() []*corev1.Namespace {
	namespaces, err := c.namespace.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching namespaces")
		return []*corev1.Namespace{}
	}
	return namespaces
}

func (c *Client) PodByNamespaceAndName(namespace string, name string) *corev1.Pod {
	item, err := c.pod.lister.Pods(namespace).Get(name)
	logGetError(fmt.Sprintf("pod %s/%s", namespace, name), err)
	return item
}

func (c *Client) PodsByLabelSelector(labelSelector *metav1.LabelSelector, namespace string) []*corev1.Pod {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		log.Error().Err(err).Msgf("Error while creating a selector  %s", labelSelector)
		return nil
	}
	list, err := c.pod.lister.Pods(namespace).List(selector)
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching Pods for selector %s in namespace %s", selector, namespace)
		return nil
	}
	return list
}

func (c *Client) Deployments() []*appsv1.Deployment {
	deployments, err := c.deployment.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching deployments")
		return []*appsv1.Deployment{}
	}
	return deployments
}

func (c *Client) DeploymentByNamespaceAndName(namespace string, name string) *appsv1.Deployment {
	item, err := c.deployment.lister.Deployments(namespace).Get(name)
	logGetError(fmt.Sprintf("deployment %s/%s", namespace, name), err)
	return item
}

func (c *Client) ServicesByPod(pod *corev1.Pod) []*corev1.Service {
	services, err := c.service.lister.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching services")
		return []*corev1.Service{}
	}
	var result []*corev1.Service
	for _, service := range services {
		match := service.Spec.Selector != nil
		for key, value := range service.Spec.Selector {
			if value != pod.ObjectMeta.Labels[key] {
				match = false
			}
		}
		if match {
			result = append(result, service)
		}
	}
	return result
}

func (c *Client) ServicesMatchingToPodLabels(namespace string, labelSelector map[string]string) []*corev1.Service {
	services, err := c.service.lister.Services(namespace).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching services")
		return []*corev1.Service{}
	}
	var result []*corev1.Service
	for _, service := range services {
		match := service.Spec.Selector != nil
		for key, value := range service.Spec.Selector {
			if value != labelSelector[key] {
				match = false
			}
		}
		if match {
			result = append(result, service)
		}
	}
	return result
}

func (c *Client) DaemonSets() []*appsv1.DaemonSet {
	daemonSets, err := c.daemonSet.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching DaemonSets")
		return []*appsv1.DaemonSet{}
	}
	return daemonSets
}

func (c *Client) DaemonSetByNamespaceAndName(namespace string, name string) *appsv1.DaemonSet {
	item, err := c.daemonSet.lister.DaemonSets(namespace).Get(name)
	logGetError(fmt.Sprintf("daemonset %s/%s", namespace, name), err)
	return item
}

func (c *Client) ReplicaSetByNamespaceAndName(namespace string, name string) *appsv1.ReplicaSet {
	item, err := c.replicaSet.lister.ReplicaSets(namespace).Get(name)
	logGetError(fmt.Sprintf("replicaset %s/%s", namespace, name), err)
	return item
}

func (c *Client) StatefulSets() []*appsv1.StatefulSet {
	statefulSets, err := c.statefulSet.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching StatefulSets")
		return []*appsv1.StatefulSet{}
	}
	return statefulSets
}

func (c *Client) StatefulSetByNamespaceAndName(namespace string, name string) *appsv1.StatefulSet {
	item, err := c.statefulSet.lister.StatefulSets(namespace).Get(name)
	logGetError(fmt.Sprintf("statefulset %s/%s", namespace, name), err)
	return item
}

func (c *Client) NodesReadyCount() int {
	nodes := c.Nodes()
	nodeCountReady := 0
	for _, node := range nodes {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				nodeCountReady = nodeCountReady + 1
			}
		}
	}
	return nodeCountReady
}

func (c *Client) Nodes() []*corev1.Node {
	nodes, err := c.node.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching nodes")
		return []*corev1.Node{}
	}
	return nodes
}

func (c *Client) Events(since time.Time) *[]corev1.Event {
	events := c.event.informer.GetIndexer().List()
	//filter events by time
	result := filterEvents(events, since)
	//sort events by time
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastTimestamp.Time.Before(result[j].LastTimestamp.Time)
	})
	return &result
}

func (c *Client) HorizontalPodAutoscalerByNamespaceAndDeployment(namespace string, reference string) *autoscalingv2.HorizontalPodAutoscaler {
	hpas, err := c.hpa.lister.HorizontalPodAutoscalers(namespace).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching horizontal pod autoscalers")
		return nil
	}
	for _, hpa := range hpas {
		if hpa.Spec.ScaleTargetRef.Kind == "Deployment" && hpa.Spec.ScaleTargetRef.Name == reference {
			return hpa
		}
	}
	return nil
}

func logGetError(resource string, err error) {
	if err != nil {
		var t *k8sErrors.StatusError
		if !errors.As(err, &t) || t.ErrStatus.Reason != metav1.StatusReasonNotFound {
			log.Error().Err(err).Msgf("Error while getting %s", resource)
		}
	}
}

func filterEvents(events []interface{}, since time.Time) []corev1.Event {
	var filtered []corev1.Event
	for _, event := range events {
		if event.(*corev1.Event).LastTimestamp.Time.After(since) {
			filtered = append(filtered, *event.(*corev1.Event))
		}
	}
	return filtered
}

func PrepareClient(stopCh <-chan struct{}) {
	clientset, rootApiPath := createClientset()
	permissions := checkPermissions(clientset)
	K8S = CreateClient(clientset, stopCh, rootApiPath, permissions)
}

// CreateClient is visible for testing
func CreateClient(clientset kubernetes.Interface, stopCh <-chan struct{}, rootApiPath string, permissions *PermissionCheckResult) *Client {
	client := &Client{
		Distribution: "kubernetes",
		permissions:  permissions,
	}
	if isOpenShift(rootApiPath) {
		client.Distribution = "openshift"
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)
	client.resourceEventHandler = cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			client.doNotify(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			client.doNotify(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			client.doNotify(obj)
		},
	}

	var informerSyncList []cache.InformerSynced

	daemonSets := factory.Apps().V1().DaemonSets()
	client.daemonSet.informer = daemonSets.Informer()
	client.daemonSet.lister = daemonSets.Lister()
	informerSyncList = append(informerSyncList, client.daemonSet.informer.HasSynced)
	if err := client.daemonSet.informer.SetTransform(transformDaemonSet); err != nil {
		log.Fatal().Err(err).Msg("Failed to add daemonSet transformer")
	}
	if _, err := client.daemonSet.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add daemonSet event handler")
	}

	deployments := factory.Apps().V1().Deployments()
	client.deployment.informer = deployments.Informer()
	client.deployment.lister = deployments.Lister()
	informerSyncList = append(informerSyncList, client.deployment.informer.HasSynced)
	if err := client.deployment.informer.SetTransform(transformDeployment); err != nil {
		log.Fatal().Err(err).Msg("Failed to add deployment transformer")
	}
	if _, err := client.deployment.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add deployment event handler")
	}

	pods := factory.Core().V1().Pods()
	client.pod.informer = pods.Informer()
	client.pod.lister = pods.Lister()
	informerSyncList = append(informerSyncList, client.pod.informer.HasSynced)
	if err := client.pod.informer.SetTransform(transformPod); err != nil {
		log.Fatal().Err(err).Msg("Failed to add pod transformer")
	}
	if _, err := client.pod.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add pod event handler")
	}

	namespaces := factory.Core().V1().Namespaces()
	client.namespace.informer = namespaces.Informer()
	client.namespace.lister = namespaces.Lister()
	informerSyncList = append(informerSyncList, client.namespace.informer.HasSynced)
	if err := client.namespace.informer.SetTransform(transformNamespace); err != nil {
		log.Fatal().Err(err).Msg("Failed to add namespace transformer")
	}
	if _, err := client.namespace.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add namespace event handler")
	}

	replicaSets := factory.Apps().V1().ReplicaSets()
	client.replicaSet.informer = replicaSets.Informer()
	client.replicaSet.lister = replicaSets.Lister()
	informerSyncList = append(informerSyncList, client.replicaSet.informer.HasSynced)
	if err := client.replicaSet.informer.SetTransform(transformReplicaSet); err != nil {
		log.Fatal().Err(err).Msg("Failed to add replicaSet transformer")
	}
	if _, err := client.replicaSet.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add replicaSet event handler")
	}

	services := factory.Core().V1().Services()
	client.service.informer = services.Informer()
	client.service.lister = services.Lister()
	informerSyncList = append(informerSyncList, client.service.informer.HasSynced)
	if err := client.service.informer.SetTransform(transformService); err != nil {
		log.Fatal().Err(err).Msg("Failed to add service transformer")
	}
	if _, err := client.service.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add service event handler")
	}

	statefulSets := factory.Apps().V1().StatefulSets()
	client.statefulSet.informer = statefulSets.Informer()
	client.statefulSet.lister = statefulSets.Lister()
	informerSyncList = append(informerSyncList, client.statefulSet.informer.HasSynced)
	if err := client.statefulSet.informer.SetTransform(transformStatefulSet); err != nil {
		log.Fatal().Err(err).Msg("Failed to add statefulSet transformer")
	}
	if _, err := client.statefulSet.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add statefulSet event handler")
	}

	nodes := factory.Core().V1().Nodes()
	client.node.informer = nodes.Informer()
	client.node.lister = nodes.Lister()
	informerSyncList = append(informerSyncList, client.node.informer.HasSynced)
	if err := client.node.informer.SetTransform(transformNodes); err != nil {
		log.Fatal().Err(err).Msg("Failed to add nodes transformer")
	}
	if _, err := client.node.informer.AddEventHandler(client.resourceEventHandler); err != nil {
		log.Fatal().Msg("failed to add node event handler")
	}

	if permissions.CanReadHorizontalPodAutoscalers() {
		hpa := factory.Autoscaling().V2().HorizontalPodAutoscalers()
		client.hpa.informer = hpa.Informer()
		client.hpa.lister = hpa.Lister()
		informerSyncList = append(informerSyncList, client.hpa.informer.HasSynced)
		if err := client.hpa.informer.SetTransform(transformHPA); err != nil {
			log.Fatal().Err(err).Msg("Failed to add hpa transformer")
		}
		if _, err := client.hpa.informer.AddEventHandler(client.resourceEventHandler); err != nil {
			log.Fatal().Msg("failed to add hpa event handler")
		}
	}

	events := factory.Core().V1().Events()
	client.event.informer = events.Informer()
	informerSyncList = append(informerSyncList, client.event.informer.HasSynced)
	if err := client.event.informer.SetTransform(transformEvents); err != nil {
		log.Fatal().Err(err).Msg("Failed to add events transformer")
	}

	defer runtime.HandleCrash()
	go factory.Start(stopCh)

	log.Info().Msgf("Start Kubernetes cache sync.")
	if !cache.WaitForCacheSync(stopCh, informerSyncList...) {
		log.Fatal().Msg("Timed out waiting for caches to sync")
	}
	log.Info().Msgf("Kubernetes caches synced.")

	return client
}

func (c *Client) doNotify(event interface{}) {
	c.handlers.Lock()
	defer c.handlers.Unlock()
	for _, ch := range c.handlers.l {
		ch <- event
	}
}

func (c *Client) Notify(ch chan<- interface{}) {
	c.handlers.Lock()
	defer c.handlers.Unlock()
	if !slices.Contains(c.handlers.l, ch) {
		c.handlers.l = append(c.handlers.l, ch)
	}
}

func (c *Client) StopNotify(ch chan<- interface{}) {
	c.handlers.Lock()
	defer c.handlers.Unlock()
	c.handlers.l = slices.DeleteFunc(c.handlers.l, func(e chan<- interface{}) bool {
		return e == ch
	})
}

func isOpenShift(rootApiPath string) bool {
	return rootApiPath == "/oapi" || rootApiPath == "oapi"
}

func createClientset() (*kubernetes.Clientset, string) {
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

	config.UserAgent = "steadybit-extension-kubernetes"
	config.Timeout = time.Second * 10
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not create kubernetes client")
	}

	info, err := clientset.ServerVersion()
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not fetch server version.")
	}

	log.Info().Msgf("Cluster connected! Kubernetes Server Version %+v", info)

	return clientset, config.APIPath
}

func IsExcludedFromDiscovery(objectMeta metav1.ObjectMeta) bool {
	discoveryEnabled, keyExists := objectMeta.Labels["steadybit.com/discovery-disabled"]
	if keyExists && strings.ToLower(discoveryEnabled) == "true" {
		return true
	}
	discoveryEnabled, keyExists = objectMeta.Labels["steadybit.com.discovery-disabled"]
	if keyExists && strings.ToLower(discoveryEnabled) == "true" {
		return true
	}
	steadybitAgent, steadybitAgentKeyExists := objectMeta.Labels["com.steadybit.agent"]
	if steadybitAgentKeyExists && strings.ToLower(steadybitAgent) == "true" {
		return true
	}
	return false
}
