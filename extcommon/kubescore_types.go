// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

// kube-score analyzes Kubernetes objects by wrapping them in internal structs.
// Typically, these structs are created by parsing Kubernetes manifests.
// However, since k8s structs are already available in memory, this glue code
// avoids unnecessary serialization and deserialization.
// Only Deployments, DaemonSets, and StatefulSets are scored by the extension,
// so only these types are handled here.

package extcommon

import (
	ks "github.com/zegl/kube-score/domain"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubeScoreObjects struct {
	bothMetas            []ks.BothMeta
	pods                 []ks.Pod
	podspecers           []ks.PodSpecer
	networkPolicies      []ks.NetworkPolicy
	services             []ks.Service
	podDisruptionBudgets []ks.PodDisruptionBudget
	deployments          []ks.Deployment
	statefulsets         []ks.StatefulSet
	ingresses            []ks.Ingress
	cronjobs             []ks.CronJob
	hpaTargeters         []ks.HpaTargeter
}

func NewScoreObjects(objects []kubeScoreInput) KubeScoreObjects {
	kubeObjects := KubeScoreObjects{}
	for _, obj := range objects {
		switch obj.GetObjectKind().GroupVersionKind().String() {

		case "apps/v1, Kind=Deployment":
			d := Deployment{
				deployment: obj.(*appsv1.Deployment),
			}
			kubeObjects.deployments = append(kubeObjects.deployments, d)
			kubeObjects.podspecers = append(kubeObjects.podspecers, d)
			kubeObjects.bothMetas = append(kubeObjects.bothMetas, ks.BothMeta{
				TypeMeta:       d.deployment.TypeMeta,
				ObjectMeta:     d.deployment.ObjectMeta,
				FileLocationer: FileLocation{},
			})

		case "/v1, Kind=Service":
			s := Service{
				service: obj.(*corev1.Service),
			}
			kubeObjects.services = append(kubeObjects.services, s)
			kubeObjects.bothMetas = append(kubeObjects.bothMetas, ks.BothMeta{
				TypeMeta:       s.service.TypeMeta,
				ObjectMeta:     s.service.ObjectMeta,
				FileLocationer: FileLocation{},
			})

		case "apps/v1, Kind=DaemonSet":
			d := DaemonSet{
				daemonSet: obj.(*appsv1.DaemonSet),
			}
			kubeObjects.podspecers = append(kubeObjects.podspecers, d)
			kubeObjects.bothMetas = append(kubeObjects.bothMetas, ks.BothMeta{
				TypeMeta:       d.daemonSet.TypeMeta,
				ObjectMeta:     d.daemonSet.ObjectMeta,
				FileLocationer: FileLocation{},
			})

		case "apps/v1, Kind=StatefulSet":
			s := StatefulSet{
				statefulSet: obj.(*appsv1.StatefulSet),
			}
			kubeObjects.statefulsets = append(kubeObjects.statefulsets, s)
			kubeObjects.podspecers = append(kubeObjects.podspecers, s)
			kubeObjects.bothMetas = append(kubeObjects.bothMetas, ks.BothMeta{
				TypeMeta:       s.statefulSet.TypeMeta,
				ObjectMeta:     s.statefulSet.ObjectMeta,
				FileLocationer: FileLocation{},
			})

		case "autoscaling/v2, Kind=HorizontalPodAutoscaler":
			h := HpaTargeter{
				hpa: obj.(*autoscalingv2.HorizontalPodAutoscaler),
			}
			kubeObjects.hpaTargeters = append(kubeObjects.hpaTargeters, h)
			kubeObjects.bothMetas = append(kubeObjects.bothMetas, ks.BothMeta{
				TypeMeta:       h.hpa.TypeMeta,
				ObjectMeta:     h.hpa.ObjectMeta,
				FileLocationer: FileLocation{},
			})
		}
	}
	return kubeObjects
}

func (p *KubeScoreObjects) Services() []ks.Service {
	return p.services
}

func (p *KubeScoreObjects) Pods() []ks.Pod {
	return p.pods
}

func (p *KubeScoreObjects) PodSpeccers() []ks.PodSpecer {
	return p.podspecers
}

func (p *KubeScoreObjects) Ingresses() []ks.Ingress {
	return p.ingresses
}

func (p *KubeScoreObjects) PodDisruptionBudgets() []ks.PodDisruptionBudget {
	return p.podDisruptionBudgets
}

func (p *KubeScoreObjects) CronJobs() []ks.CronJob {
	return p.cronjobs
}

func (p *KubeScoreObjects) Deployments() []ks.Deployment {
	return p.deployments
}

func (p *KubeScoreObjects) StatefulSets() []ks.StatefulSet {
	return p.statefulsets
}

func (p *KubeScoreObjects) Metas() []ks.BothMeta {
	return p.bothMetas
}

func (p *KubeScoreObjects) NetworkPolicies() []ks.NetworkPolicy {
	return p.networkPolicies
}

func (p *KubeScoreObjects) HorizontalPodAutoscalers() []ks.HpaTargeter {
	return p.hpaTargeters
}

type FileLocation struct {
}

func (f FileLocation) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}

type Deployment struct {
	deployment *appsv1.Deployment
}

func (d Deployment) Deployment() appsv1.Deployment {
	return *d.deployment
}
func (d Deployment) GetTypeMeta() metav1.TypeMeta {
	return d.deployment.TypeMeta
}
func (d Deployment) GetObjectMeta() metav1.ObjectMeta {
	return d.deployment.ObjectMeta
}
func (d Deployment) GetPodTemplateSpec() corev1.PodTemplateSpec {
	d.deployment.Spec.Template.ObjectMeta.Namespace = d.deployment.ObjectMeta.Namespace
	return d.deployment.Spec.Template
}
func (d Deployment) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}

type Service struct {
	service *corev1.Service
}

func (d Service) Service() corev1.Service {
	return *d.service
}
func (d Service) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}

type StatefulSet struct {
	statefulSet *appsv1.StatefulSet
}

func (s StatefulSet) StatefulSet() appsv1.StatefulSet {
	return *s.statefulSet
}
func (s StatefulSet) GetTypeMeta() metav1.TypeMeta {
	return s.statefulSet.TypeMeta
}
func (s StatefulSet) GetObjectMeta() metav1.ObjectMeta {
	return s.statefulSet.ObjectMeta
}
func (s StatefulSet) GetPodTemplateSpec() corev1.PodTemplateSpec {
	s.statefulSet.Spec.Template.ObjectMeta.Namespace = s.statefulSet.ObjectMeta.Namespace
	return s.statefulSet.Spec.Template
}
func (s StatefulSet) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}

type HpaTargeter struct {
	hpa *autoscalingv2.HorizontalPodAutoscaler
}

func (h HpaTargeter) GetTypeMeta() metav1.TypeMeta {
	return h.hpa.TypeMeta
}
func (h HpaTargeter) GetObjectMeta() metav1.ObjectMeta {
	return h.hpa.ObjectMeta
}
func (h HpaTargeter) MinReplicas() *int32 {
	return h.hpa.Spec.MinReplicas
}
func (h HpaTargeter) HpaTarget() autoscalingv1.CrossVersionObjectReference {
	return autoscalingv1.CrossVersionObjectReference(h.hpa.Spec.ScaleTargetRef)
}
func (h HpaTargeter) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}

type DaemonSet struct {
	daemonSet *appsv1.DaemonSet
}

func (d DaemonSet) GetTypeMeta() metav1.TypeMeta {
	return d.daemonSet.TypeMeta
}
func (d DaemonSet) GetObjectMeta() metav1.ObjectMeta {
	return d.daemonSet.ObjectMeta
}
func (d DaemonSet) GetPodTemplateSpec() corev1.PodTemplateSpec {
	d.daemonSet.Spec.Template.ObjectMeta.Namespace = d.daemonSet.ObjectMeta.Namespace
	return d.daemonSet.Spec.Template
}
func (d DaemonSet) FileLocation() ks.FileLocation {
	return ks.FileLocation{}
}
