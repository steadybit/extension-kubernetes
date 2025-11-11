// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Steadybit GmbH

package extingress

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/v2/client"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateNginxSteadybitModule checks if the ngx_steadybit_sleep_module.so is loaded by directly searching for NGINX controller pods
func validateNginxSteadybitModule(targetAttributes map[string][]string) error {
	if extconfig.Config.NginxDelaySkipImageCheck {
		log.Info().Msg("Skipping NGINX module validation as per configuration")
		return nil
	}

	// Get the ingress class from target attributes
	var ingressClassName string
	if ingressClass, exists := targetAttributes["k8s.ingress.class"]; exists && len(ingressClass) > 0 {
		ingressClassName = ingressClass[0]
	}

	if ingressClassName == "" {
		return fmt.Errorf("could not determine ingress class name to search for NGINX controller pods")
	}

	// Find the IngressClass to get controller deployment information
	ingressClasses := client.K8S.IngressClasses()
	var targetIngressClass *networkingv1.IngressClass
	for _, ic := range ingressClasses {
		if ic.Name == ingressClassName {
			if isNginxController(ic.Spec.Controller) {
				targetIngressClass = ic
				break
			} else {
				return fmt.Errorf("IngressClass %s is not an NGINX controller (controller: %s)", ingressClassName, ic.Spec.Controller)
			}
		}
	}

	if targetIngressClass == nil {
		return fmt.Errorf("IngressClass %s not found", ingressClassName)
	}

	var nginxPods []*corev1.Pod

	// Use IngressClass annotations to find the specific deployment/namespace
	if targetIngressClass.Annotations != nil {
		// For UBI NGINX: use "operator-sdk/primary-resource" annotation
		// Format: "namespace/deployment-name"
		if primaryResource, exists := targetIngressClass.Annotations["operator-sdk/primary-resource"]; exists {
			parts := strings.Split(primaryResource, "/")
			if len(parts) >= 2 {
				namespace := parts[0]
				deploymentName := parts[1]

				// Find pods by deployment labels in the specific namespace
				// For UBI NGINX, try multiple label selector patterns
				labelSelectors := []map[string]string{
					// Primary pattern using deployment name
					{"app": deploymentName},
					// Alternative patterns for UBI NGINX
					{"app.kubernetes.io/name": deploymentName},
					{"app.kubernetes.io/instance": deploymentName},
				}
				log.Debug().Msgf("Searching for NGINX pods in namespace %s with label selectors: %v", namespace, labelSelectors)
				nginxPods = findPodsWithLabelSelectors(labelSelectors, namespace, ingressClassName)
			}
		}

		// For community NGINX: use "meta.helm.sh/release-namespace" annotation
		if len(nginxPods) == 0 {
			if releaseNamespace, exists := targetIngressClass.Annotations["meta.helm.sh/release-namespace"]; exists {
				log.Debug().Msgf("Using meta.helm.sh/release-namespace annotation to find NGINX pods in namespace %s", releaseNamespace)
				// Common label selectors for community NGINX ingress controller
				releaseName := "nginx-ingress"
				if releaseNamAnno, exists := targetIngressClass.Annotations["meta.helm.sh/release-name"]; exists {
					log.Debug().Msgf("Using meta.helm.sh/release-name annotation to find NGINX pods with release name %s", releaseName)
					releaseName = releaseNamAnno
				}
				log.Debug().Msgf("Using release name %s to find NGINX pods", releaseName)
				labelSelectors := []map[string]string{
					{"app.kubernetes.io/instance": releaseName},
					{"app.kubernetes.io/name": releaseName, "app.kubernetes.io/component": "controller"},
					{"app.kubernetes.io/name": releaseName},
					{"app": releaseName},
				}
				log.Debug().Msgf("Searching for NGINX pods in release namespace %s with label selectors: %v", releaseNamespace, labelSelectors)
				nginxPods = findPodsWithLabelSelectors(labelSelectors, releaseNamespace, ingressClassName)
			}
		}
	} else {
		return fmt.Errorf("IngressClass %s has no Annotations", ingressClassName)
	}

	if len(nginxPods) == 0 {
		return fmt.Errorf("no NGINX ingress controller pods found for IngressClass %s", ingressClassName)
	}

	// Check all available pods for the module - we need at least one with the steadybit module
	// This handles cases where multiple nginx controllers exist but only some have the module
	var lastError error
	var checkedPods []string

	for _, pod := range nginxPods {
		checkedPods = append(checkedPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))

		containerName := "controller" // Default container name for NGINX ingress controller

		// Try to find the correct container name
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Name, "nginx") || strings.Contains(container.Name, "controller") {
				containerName = container.Name
				break
			}
		}

		// Check multiple possible nginx.conf locations
		configPaths := []string{
			"/etc/nginx/nginx.conf",
			"/usr/local/nginx/conf/nginx.conf",
			"/opt/nginx/conf/nginx.conf",
		}

		var configContent string
		var configPath string

		for _, path := range configPaths {
			output, err := client.K8S.ExecInPod(context.Background(), pod.Namespace, pod.Name, containerName, []string{"cat", path})
			if err == nil {
				configContent = output
				configPath = path
				break
			}
		}

		if configContent == "" {
			lastError = fmt.Errorf("failed to read nginx.conf from pod %s/%s: could not find configuration at any of the expected paths %v", pod.Namespace, pod.Name, configPaths)
			log.Debug().Msgf("Could not read nginx.conf from pod %s/%s, trying next pod", pod.Namespace, pod.Name)
			continue
		}

		// Check if the steadybit sleep module is loaded via load_module directive
		if strings.Contains(configContent, "ngx_steadybit_sleep_module") {
			log.Debug().Msgf("NGINX steadybit sleep module is loaded via load_module directive in %s in pod %s/%s", configPath, pod.Namespace, pod.Name)
			return nil // Found a controller with the module - success!
		}

		// If not found in main config, check if module file exists in common module directories
		modulePaths := []string{
			"/etc/nginx/modules/ngx_steadybit_sleep_module.so",
			"/usr/local/nginx/modules/ngx_steadybit_sleep_module.so",
			"/opt/nginx/modules/ngx_steadybit_sleep_module.so",
			"/usr/lib/nginx/modules/ngx_steadybit_sleep_module.so",
		}

		for _, modulePath := range modulePaths {
			exists, err := client.K8S.FileExistsInPod(context.Background(), pod.Namespace, pod.Name, containerName, modulePath)
			if err == nil && exists {
				log.Debug().Msgf("Found ngx_steadybit_sleep_module.so at %s in pod %s/%s, but it's not loaded in nginx.conf, trying next pod", modulePath, pod.Namespace, pod.Name)
				lastError = fmt.Errorf("ngx_steadybit_sleep_module.so exists at %s but is not loaded. Please add 'load_module %s;' to the nginx configuration", modulePath, modulePath)
				break // Move to next pod
			}
		}

		if lastError == nil {
			lastError = fmt.Errorf("ngx_steadybit_sleep_module is not loaded in NGINX ingress controller pod %s/%s. Please ensure the module is installed and loaded with 'load_module /path/to/ngx_steadybit_sleep_module.so;' in the nginx configuration at %s", pod.Namespace, pod.Name, configPath)
		}

		log.Debug().Msgf("Pod %s/%s does not have steadybit module, trying next pod", pod.Namespace, pod.Name)
	}

	// If we get here, none of the pods had the steadybit module
	return fmt.Errorf("ngx_steadybit_sleep_module is not loaded in any of the NGINX ingress controller pods for IngressClass %s (checked pods: %s). Please ensure at least one controller has the module installed and loaded with 'load_module /path/to/ngx_steadybit_sleep_module.so;'. Last error: %v", ingressClassName, strings.Join(checkedPods, ", "), lastError)
}

// findPodsWithLabelSelectors tries to find pods with the given label selectors in the specified namespace
func findPodsWithLabelSelectors(labelSelectors []map[string]string, namespace string, ingressClassName string) []*corev1.Pod {
	var pods []*corev1.Pod

	for _, selector := range labelSelectors {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: selector,
		}
		foundPods := client.K8S.PodsByLabelSelector(labelSelector, namespace)

		// Filter pods that serve the specific ingress class, or if we can't determine it specifically,
		// accept any NGINX pods in the right namespace (for cases where the ingress class isn't in the pod args)
		for _, pod := range foundPods {
			if podServesIngressClass(pod, ingressClassName) || isNginxControllerPod(pod) {
				pods = append(pods, pod)
			}
		}

		if len(pods) > 0 {
			break // Found pods, no need to try other selectors
		}
	}

	return pods
}

// isNginxControllerPod checks if a pod is likely an NGINX controller pod based on container names and images
func isNginxControllerPod(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		// Check container name patterns
		if strings.Contains(container.Name, "nginx") || strings.Contains(container.Name, "controller") {
			// Check image patterns
			if strings.Contains(container.Image, "nginx") ||
				strings.Contains(container.Image, "ingress-nginx") ||
				strings.Contains(container.Image, "nginx-ingress") ||
				strings.Contains(container.Image, "steadybit") {
				return true
			}
		}
	}
	return false
}

// findNginxControllerNamespace finds the NGINX controller namespace for the given ingress class
func findNginxControllerNamespace(ingressClassName string) string {
	if ingressClassName == "" {
		return ""
	}

	// First verify this IngressClass exists and is an NGINX controller
	ingressClasses := client.K8S.IngressClasses()

	var targetIngressClass *networkingv1.IngressClass
	for _, ic := range ingressClasses {
		if ic.Name == ingressClassName {
			if isNginxController(ic.Spec.Controller) {
				targetIngressClass = ic
				break
			} else {
				return ""
			}
		}
	}

	if targetIngressClass == nil {
		return ""
	}

	// Try to find namespace using IngressClass annotations
	if targetIngressClass.Annotations != nil {
		// For UBI NGINX: use "operator-sdk/primary-resource" annotation
		// Format: "namespace/deployment-name"
		if primaryResource, exists := targetIngressClass.Annotations["operator-sdk/primary-resource"]; exists {
			parts := strings.Split(primaryResource, "/")
			if len(parts) >= 1 {
				namespace := parts[0]
				if hasNginxControllerPodsForIngressClass(namespace, ingressClassName) {
					return namespace
				}
			}
		}

		// For community NGINX: use "meta.helm.sh/release-namespace" annotation
		if releaseNamespace, exists := targetIngressClass.Annotations["meta.helm.sh/release-namespace"]; exists {
			if hasNginxControllerPodsForIngressClass(releaseNamespace, ingressClassName) {
				return releaseNamespace
			}
		}
	}
	//
	//// Fallback to searching common namespaces
	//// Priority order: specific patterns first, then common namespaces
	//possibleNamespaces := []string{
	//	// Namespace patterns based on IngressClass name
	//	ingressClassName,
	//	"nginx-ingress-" + ingressClassName,
	//	ingressClassName + "-nginx-ingress",
	//
	//	// Common NGINX controller namespaces
	//	"ingress-nginx",           // Community NGINX
	//	"nginx-ingress",           // Enterprise/UBI NGINX
	//	"nginx-ingress-steadybit", // Custom deployments
	//	"nginx-system",
	//	"kube-system",
	//}
	//
	//// Search each possible namespace for NGINX controller pods
	//for _, ns := range possibleNamespaces {
	//	if hasNginxControllerPodsForIngressClass(ns, ingressClassName) {
	//		return ns
	//	}
	//}

	return ""
}

// isNginxController checks if the controller string indicates an NGINX controller
func isNginxController(controller string) bool {
	nginxControllers := []string{
		"k8s.io/ingress-nginx",         // Community NGINX
		"nginx.org/ingress-controller", // Enterprise/UBI NGINX
	}

	for _, nc := range nginxControllers {
		if controller == nc {
			return true
		}
	}
	return false
}

// hasNginxControllerPodsForIngressClass checks if there are NGINX controller pods for the specific ingress class
func hasNginxControllerPodsForIngressClass(namespace string, ingressClassName string) bool {
	labelSelectors := []map[string]string{
		// Community NGINX
		{"app.kubernetes.io/name": "ingress-nginx"},
		{"app.kubernetes.io/component": "controller", "app.kubernetes.io/name": "ingress-nginx"},

		// Enterprise/UBI NGINX
		{"app": "nginx-ingress"},
		{"app.kubernetes.io/name": "nginx-ingress"},

		// Additional patterns
		{"app.kubernetes.io/component": "controller"},
		{"k8s-app": "nginx-ingress-controller"},
		{"name": "nginx-ingress-controller"},
		{"app": "nginx-ingress-controller"},
		{"component": "nginx-ingress-controller"},
	}

	for _, selector := range labelSelectors {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: selector,
		}
		pods := client.K8S.PodsByLabelSelector(labelSelector, namespace)

		// Check each pod to see if it has the correct ingress class in container args
		for _, pod := range pods {
			if podServesIngressClass(pod, ingressClassName) {
				return true
			}
		}
	}
	return false
}

// hasNginxControllerPods checks if there are NGINX controller pods in the given namespace (legacy function)
func hasNginxControllerPods(namespace string) bool {
	labelSelectors := []map[string]string{
		// Community NGINX
		{"app.kubernetes.io/name": "ingress-nginx"},
		{"app.kubernetes.io/component": "controller", "app.kubernetes.io/name": "ingress-nginx"},

		// Enterprise/UBI NGINX
		{"app": "nginx-ingress"},
		{"app.kubernetes.io/name": "nginx-ingress"},

		// Additional patterns
		{"app.kubernetes.io/component": "controller"},
		{"k8s-app": "nginx-ingress-controller"},
		{"name": "nginx-ingress-controller"},
		{"app": "nginx-ingress-controller"},
		{"component": "nginx-ingress-controller"},
	}

	for _, selector := range labelSelectors {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: selector,
		}
		pods := client.K8S.PodsByLabelSelector(labelSelector, namespace)
		if len(pods) > 0 {
			return true
		}
	}
	return false
}

// podServesIngressClass checks if a pod serves the specified ingress class by examining container args
func podServesIngressClass(pod *corev1.Pod, ingressClassName string) bool {
	for _, container := range pod.Spec.Containers {
		// Check container arguments for -ingress-class flag
		for i, arg := range container.Args {
			// Handle formats: -ingress-class=value or -ingress-class value
			if arg == "-ingress-class" || arg == "--ingress-class" {
				// Next argument should be the class name
				if i+1 < len(container.Args) && container.Args[i+1] == ingressClassName {
					return true
				}
			} else if strings.HasPrefix(arg, "-ingress-class=") || strings.HasPrefix(arg, "--ingress-class=") {
				// Extract value after equals sign
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 && parts[1] == ingressClassName {
					return true
				}
			}
		}
	}
	return false
}
