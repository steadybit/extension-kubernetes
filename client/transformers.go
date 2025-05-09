package client

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func transformDaemonSet(i interface{}) (interface{}, error) {
	if d, ok := i.(*appsv1.DaemonSet); ok {
		d.ObjectMeta.Annotations = nil
		d.ObjectMeta.ManagedFields = nil
		d.Status = appsv1.DaemonSetStatus{
			NumberReady:            d.Status.NumberReady,
			DesiredNumberScheduled: d.Status.DesiredNumberScheduled,
		}
		return d, nil
	}
	return i, nil
}

func transformDeployment(i interface{}) (interface{}, error) {
	if d, ok := i.(*appsv1.Deployment); ok {
		d.ObjectMeta.Annotations = nil
		d.ObjectMeta.ManagedFields = nil
		d.Status.Conditions = nil
		return d, nil
	}
	return i, nil
}

func transformPod(i interface{}) (interface{}, error) {
	if pod, ok := i.(*corev1.Pod); ok {
		pod.ObjectMeta.Annotations = nil
		pod.ObjectMeta.ManagedFields = nil

		newPodSpec := corev1.PodSpec{
			NodeName:   pod.Spec.NodeName,
			HostPID:    pod.Spec.HostPID,
			Containers: make([]corev1.Container, 0, len(pod.Spec.Containers)),
		}
		for _, container := range pod.Spec.Containers {
			newPodSpec.Containers = append(newPodSpec.Containers, corev1.Container{
				Name:            container.Name,
				ImagePullPolicy: container.ImagePullPolicy,
				LivenessProbe:   container.LivenessProbe,
				ReadinessProbe:  container.ReadinessProbe,
				Resources: corev1.ResourceRequirements{
					Limits:   container.Resources.Limits,
					Requests: container.Resources.Requests,
				},
			})
		}
		pod.Spec = newPodSpec
		pod.Status = corev1.PodStatus{
			Phase:             pod.Status.Phase,
			ContainerStatuses: pod.Status.ContainerStatuses,
		}
		return pod, nil
	}
	return i, nil
}

func transformNamespace(i interface{}) (interface{}, error) {
	if namespace, ok := i.(*corev1.Namespace); ok {
		namespace.ObjectMeta.Annotations = nil
		namespace.ObjectMeta.ManagedFields = nil

		newNamespaceSpec := corev1.NamespaceSpec{
			Finalizers: nil,
		}
		namespace.Spec = newNamespaceSpec
		return namespace, nil
	}
	return i, nil
}

func transformReplicaSet(i interface{}) (interface{}, error) {
	if rs, ok := i.(*appsv1.ReplicaSet); ok {
		rs.ObjectMeta.Annotations = nil
		rs.ObjectMeta.ManagedFields = nil
		rs.Spec = appsv1.ReplicaSetSpec{}
		rs.Status = appsv1.ReplicaSetStatus{}
		return rs, nil
	}
	return i, nil
}

func transformService(i interface{}) (interface{}, error) {
	if s, ok := i.(*corev1.Service); ok {
		s.ObjectMeta.Labels = nil
		s.ObjectMeta.Annotations = nil
		s.ObjectMeta.ManagedFields = nil
		s.Spec = corev1.ServiceSpec{
			Selector: s.Spec.Selector,
		}
		s.Status = corev1.ServiceStatus{}
		return s, nil
	}
	return i, nil
}

func transformStatefulSet(i interface{}) (interface{}, error) {
	if s, ok := i.(*appsv1.StatefulSet); ok {
		s.ObjectMeta.Annotations = nil
		s.ObjectMeta.ManagedFields = nil
		s.Status = appsv1.StatefulSetStatus{
			ReadyReplicas: s.Status.ReadyReplicas,
		}
		return s, nil
	}
	return i, nil
}

func transformEvents(i interface{}) (interface{}, error) {
	if event, ok := i.(*corev1.Event); ok {
		event.ObjectMeta.ManagedFields = nil
		return event, nil
	}
	return i, nil
}

func transformNodes(i interface{}) (interface{}, error) {
	if node, ok := i.(*corev1.Node); ok {
		node.ObjectMeta.Annotations = nil
		node.ObjectMeta.ManagedFields = nil
		node.Spec = corev1.NodeSpec{}
		node.Status = corev1.NodeStatus{
			Conditions: node.Status.Conditions,
			Addresses:  node.Status.Addresses,
		}
		return node, nil
	}
	return i, nil
}

func transformHPA(i interface{}) (interface{}, error) {
	if hpa, ok := i.(*autoscalingv1.HorizontalPodAutoscaler); ok {
		hpa.ObjectMeta.Annotations = nil
		hpa.ObjectMeta.ManagedFields = nil
		return hpa, nil
	}
	return i, nil
}

func transformIngressClass(i interface{}) (interface{}, error) {
	if ic, ok := i.(*networkingv1.IngressClass); ok {
		// Keep only needed annotations, particularly is-default-class
		defaultClassAnnotation := ""
		if ic.Annotations != nil {
			defaultClassAnnotation = ic.Annotations["ingressclass.kubernetes.io/is-default-class"]
		}

		if defaultClassAnnotation != "" {
			ic.Annotations = map[string]string{
				"ingressclass.kubernetes.io/is-default-class": defaultClassAnnotation,
			}
		} else {
			ic.Annotations = nil
		}

		ic.ObjectMeta.ManagedFields = nil

		return ic, nil
	}
	return i, nil
}

func transformIngress(i interface{}) (interface{}, error) {
	if d, ok := i.(*networkingv1.Ingress); ok {
		// Preserve ingressClassName and the class annotation if present
		ingressClassName := d.Spec.IngressClassName

		// Preserve only the specific annotation we need
		classAnnotation := ""
		if d.ObjectMeta.Annotations != nil {
			classAnnotation = d.ObjectMeta.Annotations["kubernetes.io/ingress.class"]
		}

		// Clear annotations but keep what we need in a minimal map
		if classAnnotation != "" {
			d.ObjectMeta.Annotations = map[string]string{
				"kubernetes.io/ingress.class": classAnnotation,
			}
		} else {
			d.ObjectMeta.Annotations = nil
		}

		d.ObjectMeta.ManagedFields = nil

		// Keep the rules and TLS configuration for discovery
		rules := d.Spec.Rules
		tls := d.Spec.TLS

		// Create minimal spec with only what we need
		d.Spec = networkingv1.IngressSpec{
			IngressClassName: ingressClassName,
			Rules:           rules,
			TLS:             tls,
		}

		return d, nil
	}
	return i, nil
}

