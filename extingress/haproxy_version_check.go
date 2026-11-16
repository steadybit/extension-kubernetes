// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extingress

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/v2/client"
	corev1 "k8s.io/api/core/v1"
)

// HAProxy Kubernetes Ingress Controller 3.4 dropped support for config snippets, which the HAProxy actions rely on.
// See https://github.com/haproxytech/kubernetes-ingress/commit/82c3b72c81e241a10bd612c29ee06879ea0a718a
const (
	haProxyFirstUnsupportedMajor = 3
	haProxyFirstUnsupportedMinor = 4
)

var haProxyImageTagVersionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)`)

// checkHAProxyControllerSupportsConfigSnippets returns an error if the HAProxy ingress controller serving the given
// ingress class runs a version that no longer supports config snippets. Controllers whose version cannot be determined
// (e.g. custom tags or digest-only image references) are allowed.
// The image is read from the container statuses, as the pod informer strips the image from the pod spec.
func checkHAProxyControllerSupportsConfigSnippets(ingressClassName string) error {
	for _, pod := range findHAProxyControllerPods(client.K8S.Pods(), ingressClassName) {
		for _, container := range pod.Status.ContainerStatuses {
			if !isHAProxyIngressControllerImage(container.Image) {
				continue
			}
			major, minor, ok := parseHAProxyImageVersion(container.Image)
			if !ok {
				log.Warn().Msgf("Could not determine HAProxy ingress controller version from image %s of pod %s/%s, skipping version check", container.Image, pod.Namespace, pod.Name)
				continue
			}
			if major > haProxyFirstUnsupportedMajor || (major == haProxyFirstUnsupportedMajor && minor >= haProxyFirstUnsupportedMinor) {
				return fmt.Errorf("HAProxy ingress controller %d.%d (pod %s/%s) is not supported yet: HAProxy %d.%d removed support for config snippets, which this action relies on. Supported versions are up to 3.2.x",
					major, minor, pod.Namespace, pod.Name, haProxyFirstUnsupportedMajor, haProxyFirstUnsupportedMinor)
			}
		}
	}
	return nil
}

// findHAProxyControllerPods returns the HAProxy ingress controller pods serving the given ingress class. If no pod
// explicitly declares the class, all HAProxy ingress controller pods are returned.
func findHAProxyControllerPods(pods []*corev1.Pod, ingressClassName string) []*corev1.Pod {
	var controllerPods, classPods []*corev1.Pod
	for _, pod := range pods {
		if !isHAProxyIngressControllerPod(pod) {
			continue
		}
		controllerPods = append(controllerPods, pod)
		if ingressClassName != "" && haProxyPodServesIngressClass(pod, ingressClassName) {
			classPods = append(classPods, pod)
		}
	}
	if len(classPods) > 0 {
		return classPods
	}
	return controllerPods
}

func isHAProxyIngressControllerPod(pod *corev1.Pod) bool {
	for _, container := range pod.Status.ContainerStatuses {
		if isHAProxyIngressControllerImage(container.Image) {
			return true
		}
	}
	return false
}

func isHAProxyIngressControllerImage(image string) bool {
	return strings.Contains(imageRepository(image), "haproxytech/kubernetes-ingress")
}

func haProxyPodServesIngressClass(pod *corev1.Pod, ingressClassName string) bool {
	for _, container := range pod.Spec.Containers {
		for i, arg := range container.Args {
			if arg == "--ingress.class" && i+1 < len(container.Args) && container.Args[i+1] == ingressClassName {
				return true
			}
			if arg == "--ingress.class="+ingressClassName {
				return true
			}
		}
	}
	return false
}

// parseHAProxyImageVersion extracts major and minor version from an image tag like "haproxytech/kubernetes-ingress:3.4.0"
func parseHAProxyImageVersion(image string) (int, int, bool) {
	tag := imageTag(image)
	matches := haProxyImageTagVersionRegex.FindStringSubmatch(tag)
	if matches == nil {
		return 0, 0, false
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

func imageRepository(image string) string {
	image, _, _ = strings.Cut(image, "@")
	if i := strings.LastIndex(image, ":"); i > strings.LastIndex(image, "/") {
		return image[:i]
	}
	return image
}

func imageTag(image string) string {
	image, _, _ = strings.Cut(image, "@")
	if i := strings.LastIndex(image, ":"); i > strings.LastIndex(image, "/") {
		return image[i+1:]
	}
	return ""
}
