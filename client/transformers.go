package client

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
)

func transformDaemonSet(i interface{}) (interface{}, error) {
	d, ok := i.(*appsv1.DaemonSet)
	if ok {
		d.ObjectMeta.Annotations = nil
		d.ObjectMeta.ManagedFields = nil
		d.Status = appsv1.DaemonSetStatus{}
		return d, nil
	}
	return i, nil
}

func transformDeployment(i interface{}) (interface{}, error) {
	d, ok := i.(*appsv1.Deployment)
	if ok {
		d.ObjectMeta.Annotations = nil
		d.ObjectMeta.ManagedFields = nil
		d.Status.Conditions = nil
		return d, nil
	}
	return i, nil
}

func transformPod(i interface{}) (interface{}, error) {
	pod, ok := i.(*corev1.Pod)
	if ok {
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
			newContainer.Resources.Claims = nil
			newPodSpec.Containers[index] = newContainer
		}
		pod.Spec = newPodSpec
		newPodStatus := corev1.PodStatus{}
		newPodStatus.ContainerStatuses = pod.Status.ContainerStatuses
		pod.Status = newPodStatus
		return pod, nil
	}
	return i, nil
}

func transformReplicaSet(i interface{}) (interface{}, error) {
	rs, ok := i.(*appsv1.ReplicaSet)
	if ok {
		rs.ObjectMeta.Annotations = nil
		rs.ObjectMeta.ManagedFields = nil
		rs.Spec = appsv1.ReplicaSetSpec{}
		rs.Status = appsv1.ReplicaSetStatus{}
		return rs, nil
	}
	return i, nil
}

func transformService(i interface{}) (interface{}, error) {
	s, ok := i.(*corev1.Service)
	if ok {
		s.ObjectMeta.Labels = nil
		s.ObjectMeta.Annotations = nil
		s.ObjectMeta.ManagedFields = nil
		newServiceSpec := corev1.ServiceSpec{}
		newServiceSpec.Selector = s.Spec.Selector
		s.Spec = newServiceSpec
		s.Status = corev1.ServiceStatus{}
		return s, nil
	}
	return i, nil
}

func transformStatefulSet(i interface{}) (interface{}, error) {
	s, ok := i.(*appsv1.StatefulSet)
	if ok {
		s.ObjectMeta.Annotations = nil
		s.ObjectMeta.ManagedFields = nil
		s.Status = appsv1.StatefulSetStatus{}
		return s, nil
	}
	return i, nil
}

func transformEvents(i interface{}) (interface{}, error) {
	event, ok := i.(*corev1.Event)
	if ok {
		event.ObjectMeta.ManagedFields = nil
		return event, nil
	}
	return i, nil
}

func transformNodes(i interface{}) (interface{}, error) {
	node, ok := i.(*corev1.Node)
	if ok {
		node.ObjectMeta.Annotations = nil
		node.ObjectMeta.ManagedFields = nil
		node.Spec = corev1.NodeSpec{}
		newNodeStatus := corev1.NodeStatus{}
		newNodeStatus.Conditions = node.Status.Conditions
		node.Status = newNodeStatus
		return node, nil
	}
	return i, nil
}

func transformHPA(i interface{}) (interface{}, error) {
	hpa, ok := i.(*autoscalingv1.HorizontalPodAutoscaler)
	if ok {
		hpa.ObjectMeta.Annotations = nil
		hpa.ObjectMeta.ManagedFields = nil
		return hpa, nil
	}
	return i, nil
}
