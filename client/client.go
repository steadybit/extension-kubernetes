// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package client

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	networkingv1client "k8s.io/client-go/kubernetes/typed/networking/v1"
	listerAppsv1 "k8s.io/client-go/listers/apps/v1"
	listerAutoscalingv2 "k8s.io/client-go/listers/autoscaling/v2"
	listerCorev1 "k8s.io/client-go/listers/core/v1"
	listerNetworkingv1 "k8s.io/client-go/listers/networking/v1"
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

	ingress struct {
		lister   listerNetworkingv1.IngressLister
		informer cache.SharedIndexInformer
	}

	ingressClass struct {
		lister   listerNetworkingv1.IngressClassLister
		informer cache.SharedIndexInformer
	}

	handlers struct {
		sync.Mutex
		l []chan<- interface{}
	}
	resourceEventHandler cache.ResourceEventHandlerFuncs
	networkingV1         networkingv1client.NetworkingV1Interface
}

func (c *Client) Permissions() *PermissionCheckResult {
	return c.permissions
}

func (c *Client) Pods() []*corev1.Pod {
	if extconfig.HasNamespaceFilter() {
		log.Info().Msgf("Fetching pods for namespace %s", extconfig.Config.Namespace)
		pods, err := c.pod.lister.Pods(extconfig.Config.Namespace).List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching pods")
			return []*corev1.Pod{}
		}
		return c.onlyRunningPods(pods)
	} else {
		pods, err := c.pod.lister.List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching pods")
			return []*corev1.Pod{}
		}
		return c.onlyRunningPods(pods)
	}
}

func (c *Client) Namespaces() []*corev1.Namespace {
	if extconfig.HasNamespaceFilter() {
		var namespace = &corev1.Namespace{}
		namespace.Name = extconfig.Config.Namespace
		return []*corev1.Namespace{namespace}
	} else {
		namespaces, err := c.namespace.lister.List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching namespaces")
			return []*corev1.Namespace{}
		}
		return namespaces
	}
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
	return c.onlyRunningPods(list)
}

func (c *Client) onlyRunningPods(list []*corev1.Pod) []*corev1.Pod {
	runningPods := make([]*corev1.Pod, 0)
	for _, pod := range list {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods = append(runningPods, pod)
		}
	}
	return runningPods
}

func (c *Client) Deployments() []*appsv1.Deployment {
	if extconfig.HasNamespaceFilter() {
		log.Info().Msgf("Fetching deployments for namespace %s", extconfig.Config.Namespace)
		deployments, err := c.deployment.lister.Deployments(extconfig.Config.Namespace).List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching deployments")
			return []*appsv1.Deployment{}
		}
		return deployments
	} else {
		deployments, err := c.deployment.lister.List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching deployments")
			return []*appsv1.Deployment{}
		}
		return deployments
	}
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
	if extconfig.HasNamespaceFilter() {
		log.Info().Msgf("Fetching daemonsets for namespace %s", extconfig.Config.Namespace)
		daemonSets, err := c.daemonSet.lister.DaemonSets(extconfig.Config.Namespace).List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching DaemonSets")
			return []*appsv1.DaemonSet{}
		}
		return daemonSets
	} else {
		daemonSets, err := c.daemonSet.lister.List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching DaemonSets")
			return []*appsv1.DaemonSet{}
		}
		return daemonSets
	}
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
	if extconfig.HasNamespaceFilter() {
		log.Info().Msgf("Fetching statefulsets for namespace %s", extconfig.Config.Namespace)
		statefulSets, err := c.statefulSet.lister.StatefulSets(extconfig.Config.Namespace).List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching StatefulSets")
			return []*appsv1.StatefulSet{}
		}
		return statefulSets
	} else {
		statefulSets, err := c.statefulSet.lister.List(labels.Everything())
		if err != nil {
			log.Error().Err(err).Msgf("Error while fetching StatefulSets")
			return []*appsv1.StatefulSet{}
		}
		return statefulSets
	}
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
	if extconfig.HasNamespaceFilter() {
		return []*corev1.Node{}
	}
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

func (c *Client) Ingresses() []*networkingv1.Ingress {
	if extconfig.HasNamespaceFilter() {
		return []*networkingv1.Ingress{}
	}
	ingresses, err := c.ingress.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching ingresses")
		return []*networkingv1.Ingress{}
	}
	return ingresses
}

func (c *Client) IngressClasses() []*networkingv1.IngressClass {
	if extconfig.HasNamespaceFilter() {
		return []*networkingv1.IngressClass{}
	}
	ingressClasses, err := c.ingressClass.lister.List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching IngressClasses")
		return []*networkingv1.IngressClass{}
	}
	return ingressClasses
}

func (c *Client) GetHAProxyIngressClasses() ([]string, bool) {
	ingressClasses := c.IngressClasses()
	haproxyClassNames := make([]string, 0)
	hasDefaultClass := false

	for _, ic := range ingressClasses {
		if ic.Spec.Controller == "haproxy.org/ingress-controller/haproxy" {
			haproxyClassNames = append(haproxyClassNames, ic.Name)

			// Check if this is a default class
			if ic.Annotations != nil {
				if value, ok := ic.Annotations["ingressclass.kubernetes.io/is-default-class"]; ok && value == "true" {
					hasDefaultClass = true
				}
			}
		}
	}

	return haproxyClassNames, hasDefaultClass
}

func (c *Client) GetNginxIngressClasses() ([]string, bool) {
	ingressClasses := c.IngressClasses()
	nginxClassNames := make([]string, 0)
	hasDefaultClass := false

	// Controller names for both open source and enterprise NGINX ingress controllers
	nginxControllers := []string{
		"k8s.io/ingress-nginx",         // Open source NGINX Ingress Controller
		"nginx.org/ingress-controller", // NGINX Enterprise Ingress Controller
	}

	for _, ic := range ingressClasses {
		// Check if this IngressClass uses an NGINX controller
		for _, controller := range nginxControllers {
			if ic.Spec.Controller == controller {
				nginxClassNames = append(nginxClassNames, ic.Name)

				// Check if this is a default class
				if ic.Annotations != nil {
					if value, ok := ic.Annotations["ingressclass.kubernetes.io/is-default-class"]; ok && value == "true" {
						hasDefaultClass = true
					}
				}
				break // No need to check other controllers for this class
			}
		}
	}

	// Also include the classic "nginx" class name for backward compatibility
	if !slices.Contains(nginxClassNames, "nginx") {
		nginxClassNames = append(nginxClassNames, "nginx")
	}

	return nginxClassNames, hasDefaultClass
}

func (c *Client) GetIngressControllerByClassName(className string) string {
	ingressClasses := c.IngressClasses()
	for _, ic := range ingressClasses {
		if ic.Name == className {
			return ic.Spec.Controller
		}
	}
	return ""
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

	informerResyncDuration := time.Duration(extconfig.Config.DiscoveryInformerResync) * time.Second
	var factory informers.SharedInformerFactory
	if extconfig.HasNamespaceFilter() {
		factory = informers.NewSharedInformerFactoryWithOptions(clientset, informerResyncDuration, informers.WithNamespace(extconfig.Config.Namespace))
	} else {
		factory = informers.NewSharedInformerFactory(clientset, informerResyncDuration)
	}
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

	if permissions.CanReadNamespaces() && !extconfig.HasNamespaceFilter() {
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

	if !extconfig.HasNamespaceFilter() {
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

	// Add ingress informer
	if !extconfig.HasNamespaceFilter() && permissions.IsListIngressPermitted() {
		ingresses := factory.Networking().V1().Ingresses()
		client.ingress.informer = ingresses.Informer()
		client.ingress.lister = ingresses.Lister()
		client.networkingV1 = clientset.NetworkingV1()
		informerSyncList = append(informerSyncList, client.ingress.informer.HasSynced)
		if err := client.ingress.informer.SetTransform(transformIngress); err != nil {
			log.Fatal().Err(err).Msg("Failed to add ingress transformer")
		}
		if _, err := client.ingress.informer.AddEventHandler(client.resourceEventHandler); err != nil {
			log.Fatal().Msg("failed to add ingress event handler")
		}
	}

	// Add ingressClasses informer
	if !extconfig.HasNamespaceFilter() && permissions.IsListIngressClassesPermitted() {
		ingressClasses := factory.Networking().V1().IngressClasses()
		client.ingressClass.informer = ingressClasses.Informer()
		client.ingressClass.lister = ingressClasses.Lister()
		informerSyncList = append(informerSyncList, client.ingressClass.informer.HasSynced)
		if err := client.ingressClass.informer.SetTransform(transformIngressClass); err != nil {
			log.Fatal().Err(err).Msg("Failed to add ingressClass transformer")
		}
		if _, err := client.ingressClass.informer.AddEventHandler(client.resourceEventHandler); err != nil {
			log.Fatal().Msg("failed to add ingressClass event handler")
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
func (c *Client) IngressByNamespaceAndName(namespace string, name string, forceUpdate ...bool) (*networkingv1.Ingress, error) {
	// Check if we should bypass the cache
	if len(forceUpdate) > 0 && forceUpdate[0] {
		// Get directly from the API server instead of the cache
		ingress, err := c.networkingV1.Ingresses(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				return nil, fmt.Errorf("ingress %s/%s not found", namespace, name)
			}
			return nil, fmt.Errorf("error fetching ingress %s/%s directly from API: %w", namespace, name, err)
		}
		return ingress, nil
	}

	// Use the cache as before
	ingress, err := c.ingress.lister.Ingresses(namespace).Get(name)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("ingress %s/%s not found", namespace, name)
		}
		return nil, fmt.Errorf("error fetching ingress %s/%s: %w", namespace, name, err)
	}
	return ingress, nil
}

// GetConfig returns the kubernetes config used by the client
func (c *Client) GetConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to get in-cluster config, using default config")
		config = &rest.Config{}
	}
	return config
}

func (c *Client) UpdateIngressAnnotation(ctx context.Context, namespace string, ingressName string, annotationKey string, newAnnotationSuffix string) error {
	maxRetries := 10

	for attempt := 0; attempt < maxRetries; attempt++ {
		ingress, err := c.IngressByNamespaceAndName(namespace, ingressName, true)
		if err != nil {
			return fmt.Errorf("failed to fetch ingress: %w", err)
		}

		newConfig := ""
		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
			newConfig = newAnnotationSuffix
		} else {
			// Prepend the new configuration
			if existingValue := ingress.Annotations[annotationKey]; existingValue == "" {
				newConfig = newAnnotationSuffix
			} else {
				newConfig = newAnnotationSuffix + "\n" + existingValue
			}
		}
		// Update the annotation
		ingress.Annotations[annotationKey] = newConfig

		_, err = c.networkingV1.Ingresses(namespace).Update(
			ctx,
			ingress,
			metav1.UpdateOptions{},
		)

		if err == nil {
			log.Debug().Msgf("Updated ingress %s/%s annotation %s with new config: %s", namespace, ingressName, annotationKey, newConfig)
			return nil // Update successful
		}

		// If it's not a conflict error, return the error immediately
		if !k8sErrors.IsConflict(err) {
			log.Error().Err(err).Msgf("Failed to update ingress %s/%s annotation %s: %v", namespace, ingressName, annotationKey, err)
			return fmt.Errorf("failed to update ingress annotation: %w", err)
		}

		// If it's a conflict error, we'll retry with the latest version of the resource
		log.Debug().Msgf("Conflict detected while updating ingress %s/%s, retrying (attempt %d/%d)",
			namespace, ingressName, attempt+1, maxRetries)
	}
	log.Error().Msgf("Failed to update ingress %s/%s annotation %s after %d attempts", namespace, ingressName, annotationKey, maxRetries)
	return fmt.Errorf("failed to update ingress annotation after %d attempts due to concurrent modifications", maxRetries)
}

func (c *Client) RemoveAnnotationBlock(ctx context.Context, namespace string, ingressName string, annotationKey string, executionId uuid.UUID) error {
	log.Debug().Msgf("Removing annotation block from ingress %s/%s with execution ID %s", namespace, ingressName, executionId)
	maxRetries := 10

	startMarker := fmt.Sprintf("# BEGIN STEADYBIT - %s", executionId)
	endMarker := fmt.Sprintf("# END STEADYBIT - %s", executionId)

	// Use the clientset to update the ingress

	for attempt := 0; attempt < maxRetries; attempt++ {
		ingress, err := c.IngressByNamespaceAndName(namespace, ingressName, true)
		if err != nil {
			return fmt.Errorf("failed to fetch ingress: %w", err)
		}

		if ingress.Annotations == nil || ingress.Annotations[annotationKey] == "" {
			return nil // Nothing to remove
		}

		// Remove the configuration block between markers
		existingConfig := ingress.Annotations[annotationKey]
		updatedConfig := removeAnnotationBlock(existingConfig, startMarker, endMarker)

		// If nothing was removed, no need to update
		if existingConfig == updatedConfig {
			return nil
		}

		// Update the annotation with the block removed
		ingress.Annotations[annotationKey] = updatedConfig

		_, err = c.networkingV1.Ingresses(namespace).Update(
			ctx,
			ingress,
			metav1.UpdateOptions{},
		)

		if err == nil {
			return nil // Update successful
		}

		// If it's not a conflict error, return the error immediately
		if !k8sErrors.IsConflict(err) {
			return fmt.Errorf("failed to update ingress annotation: %w", err)
		}

		// If it's a conflict error, we'll retry with the latest version of the resource
		log.Debug().Msgf("Conflict detected while removing annotation block in ingress %s/%s, retrying (attempt %d/%d)",
			namespace, ingressName, attempt+1, maxRetries)
	}

	return fmt.Errorf("failed to update ingress annotation after %d attempts due to concurrent modifications", maxRetries)
}

// removeAnnotationBlock removes the text between startMarker and endMarker (inclusive)
// and all consecutive newlines that follow the block
func removeAnnotationBlock(config, startMarker, endMarker string) string {
	startIndex := strings.Index(config, startMarker)
	endIndex := strings.Index(config, endMarker)

	if startIndex == -1 || endIndex == -1 {
		return config // Markers not found
	}

	// Calculate end of marker position
	endOfMarker := endIndex + len(endMarker)

	// Skip all consecutive newlines after the end marker
	for endOfMarker < len(config) && config[endOfMarker] == '\n' {
		endOfMarker++
	}

	// Remove the block including the markers and all trailing newlines
	return config[:startIndex] + config[endOfMarker:]
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
	if extconfig.Config.DisableDiscoveryExcludes {
		return false
	}

	if label, ok := objectMeta.Labels["steadybit.com/discovery-disabled"]; ok && strings.ToLower(label) == "true" {
		return true
	}

	if label, ok := objectMeta.Labels["steadybit.com.discovery-disabled"]; ok && strings.ToLower(label) == "true" {
		return true
	}

	if label, ok := objectMeta.Labels["com.steadybit.agent"]; ok && strings.ToLower(label) == "true" {
		return true
	}
	return false
}
