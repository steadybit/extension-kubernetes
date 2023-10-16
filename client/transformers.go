package client

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func transformDaemonset(i interface{}) (interface{}, error) {
	d := i.(*appsv1.DaemonSet)
	d.ObjectMeta.Annotations = nil
	d.ObjectMeta.ManagedFields = nil
	newDaemonSetSpec := appsv1.DaemonSetSpec{}
	newDaemonSetSpec.Selector = d.Spec.Selector
	d.Spec = newDaemonSetSpec
	d.Status = appsv1.DaemonSetStatus{}
	return d, nil
}

func transformDeployment(i interface{}) (interface{}, error) {
	d := i.(*appsv1.Deployment)
	d.ObjectMeta.Annotations = nil
	d.ObjectMeta.ManagedFields = nil
	d.Status.Conditions = nil
	return d, nil
}

func transformPod(i interface{}) (interface{}, error) {
	pod := i.(*corev1.Pod)
	pod.ObjectMeta.Annotations = nil
	pod.ObjectMeta.ManagedFields = nil
	newPodSpec := corev1.PodSpec{}
	newPodSpec.NodeName = pod.Spec.NodeName
	newPodSpec.HostPID = pod.Spec.HostPID
	newPodSpec.Containers = make([]corev1.Container, len(pod.Spec.Containers))
	for index, container := range pod.Spec.Containers {
		newContainer := corev1.Container{}
		newContainer.Name = container.Name
		newContainer.ImagePullPolicy = container.ImagePullPolicy
		newContainer.LivenessProbe = container.LivenessProbe
		newContainer.ReadinessProbe = container.ReadinessProbe
		newContainer.Resources = container.Resources
		newContainer.Resources.Requests = nil
		newContainer.Resources.Claims = nil
		newPodSpec.Containers[index] = newContainer
	}
	pod.Spec = newPodSpec
	newPodStatus := corev1.PodStatus{}
	newPodStatus.ContainerStatuses = pod.Status.ContainerStatuses
	pod.Status = newPodStatus
	return pod, nil
}

func transformReplicaSet(i interface{}) (interface{}, error) {
	rs := i.(*appsv1.ReplicaSet)
	rs.ObjectMeta.Annotations = nil
	rs.ObjectMeta.ManagedFields = nil
	rs.Spec = appsv1.ReplicaSetSpec{}
	rs.Status = appsv1.ReplicaSetStatus{}
	return rs, nil
}

func transformService(i interface{}) (interface{}, error) {
	s := i.(*corev1.Service)
	s.ObjectMeta.Labels = nil
	s.ObjectMeta.Annotations = nil
	s.ObjectMeta.ManagedFields = nil
	newServiceSpec := corev1.ServiceSpec{}
	newServiceSpec.Selector = s.Spec.Selector
	s.Spec = newServiceSpec
	s.Status = corev1.ServiceStatus{}
	return s, nil
}

func transformStatefulSet(i interface{}) (interface{}, error) {
	s := i.(*appsv1.StatefulSet)
	s.ObjectMeta.Annotations = nil
	s.ObjectMeta.ManagedFields = nil
	newStatefulSetSpec := appsv1.StatefulSetSpec{}
	newStatefulSetSpec.Replicas = s.Spec.Replicas
	newStatefulSetSpec.Selector = s.Spec.Selector
	s.Spec = newStatefulSetSpec
	s.Status = appsv1.StatefulSetStatus{}
	return s, nil
}

func transformEvents(i interface{}) (interface{}, error) {
	event := i.(*corev1.Event)
	event.ObjectMeta.ManagedFields = nil
	return event, nil
}

func transformNodes(i interface{}) (interface{}, error) {
	node := i.(*corev1.Node)
	node.ObjectMeta.Annotations = nil
	node.ObjectMeta.ManagedFields = nil
	node.Spec = corev1.NodeSpec{}
	newNodeStatus := corev1.NodeStatus{}
	newNodeStatus.Conditions = node.Status.Conditions
	node.Status = newNodeStatus
	return node, nil
}
