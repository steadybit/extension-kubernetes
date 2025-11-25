package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubernetesClient creates a kubernetes.Interface using either:
//   - the in-cluster configuration (if running inside Kubernetes), or
//   - the default kubeconfig (if running locally)
//
// It automatically selects the correct configuration.
func NewKubernetesClient() (kubernetes.Interface, error) {
	config, err := loadKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kube config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// loadKubeConfig returns the in-cluster config when available; otherwise it falls back to ~/.kube/config.
func loadKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	inCluster, err := rest.InClusterConfig()
	if err == nil {
		return inCluster, nil
	}

	// Fallback to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find home directory for kubeconfig: %w", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("cannot build config from kubeconfig: %w", err)
	}

	return config, nil
}

// GetWorkloadJSON fetches a specific Kubernetes workload (e.g., Deployment, StatefulSet)
// by kind, namespace, and name, and returns a sanitized JSON representation.
//
// The JSON omits short‑lived and noisy attributes such as status, resourceVersion,
// creationTimestamp, managedFields, and labels. The goal is to keep only fields
// that are relevant for reliability analysis.
func GetWorkloadJSON(ctx context.Context, client kubernetes.Interface, kind, namespace, name string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("kubernetes client is nil")
	}

	kindLower := strings.ToLower(kind)

	var obj any
	var err error

	switch kindLower {
	case "deployment", "deployments":
		obj, err = client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "statefulset", "statefulsets":
		obj, err = client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	default:
		return "", fmt.Errorf("unsupported kind '%s', supported kinds are Deployment and StatefulSet", kind)
	}

	if err != nil {
		return "", fmt.Errorf("failed to fetch %s %s/%s: %w", kind, namespace, name, err)
	}

	// Marshal to JSON, then unmarshal into a generic map for sanitization.
	raw, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to marshal %s %s/%s to JSON: %w", kind, namespace, name, err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("failed to unmarshal %s %s/%s JSON for sanitization: %w", kind, namespace, name, err)
	}

	sanitizeKubernetesObject(m)

	cleaned, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sanitized %s %s/%s to JSON: %w", kind, namespace, name, err)
	}

	return string(cleaned), nil
}

// sanitizeKubernetesObject removes short‑lived or noisy fields from a Kubernetes object
// representation to focus on attributes that are relevant for reliability analysis.
func sanitizeKubernetesObject(obj map[string]any) {
	if obj == nil {
		return
	}

	// Drop status entirely – highly volatile runtime state.
	delete(obj, "status")

	// Top-level metadata cleanup.
	if meta, ok := obj["metadata"].(map[string]any); ok {
		cleanMetadata(meta)
		obj["metadata"] = meta
	}

	// Clean nested spec/template metadata and pod-level noise.
	if spec, ok := obj["spec"].(map[string]any); ok {
		cleanSpec(spec)
		obj["spec"] = spec
	}

	// Recursively drop empty maps/slices to shrink further.
	pruneEmpty(obj)
}

// cleanMetadata removes identity / lifecycle / noisy fields from metadata.
func cleanMetadata(meta map[string]any) {
	if meta == nil {
		return
	}

	// Identity & lifecycle fields.
	delete(meta, "creationTimestamp")
	delete(meta, "resourceVersion")
	delete(meta, "uid")
	delete(meta, "managedFields")
	delete(meta, "generation")
	delete(meta, "selfLink")
	delete(meta, "ownerReferences")
	delete(meta, "finalizers")

	// Often huge and rarely needed for reliability reasoning.
	delete(meta, "annotations")
	delete(meta, "labels")
}

// cleanSpec focuses spec on the parts that matter and trims nested metadata.
func cleanSpec(spec map[string]any) {
	if spec == nil {
		return
	}

	// Handle spec.template.metadata and spec.template.spec
	if tmpl, ok := spec["template"].(map[string]any); ok {
		if meta, ok := tmpl["metadata"].(map[string]any); ok {
			// keep only name/namespace if present, drop everything else
			name, _ := meta["name"]
			namespace, _ := meta["namespace"]
			newMeta := map[string]any{}
			if name != nil {
				newMeta["name"] = name
			}
			if namespace != nil {
				newMeta["namespace"] = namespace
			}
			tmpl["metadata"] = newMeta
		}

		if podSpec, ok := tmpl["spec"].(map[string]any); ok {
			cleanPodSpec(podSpec)
			tmpl["spec"] = podSpec
		}

		spec["template"] = tmpl
	}
}

// cleanPodSpec removes fields that are typically noisy and not essential for reliability checks.
func cleanPodSpec(podSpec map[string]any) {
	if podSpec == nil {
		return
	}

	// Drop scheduler/runtime noise that tends to be long and rarely relevant to reliability logic.
	delete(podSpec, "nodeName")
	delete(podSpec, "serviceAccount")
	delete(podSpec, "serviceAccountName")
	delete(podSpec, "automountServiceAccountToken")
	delete(podSpec, "priorityClassName")
	delete(podSpec, "schedulerName")
	delete(podSpec, "enableServiceLinks")
	delete(podSpec, "dnsConfig")

	// Keep containers, resources, probes, env, tolerations, affinity, nodeSelector, etc.
	// These can be important for reliability reasoning, so we do NOT delete them here.
}

// pruneEmpty recursively removes empty maps and slices to shrink token count further.
func pruneEmpty(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		for k, val := range typed {
			if val == nil {
				delete(typed, k)
				continue
			}
			newVal := pruneEmpty(val)
			switch nv := newVal.(type) {
			case nil:
				delete(typed, k)
			case map[string]any:
				if len(nv) == 0 {
					delete(typed, k)
				} else {
					typed[k] = nv
				}
			case []any:
				if len(nv) == 0 {
					delete(typed, k)
				} else {
					typed[k] = nv
				}
			default:
				typed[k] = newVal
			}
		}
		return typed
	case []any:
		out := make([]any, 0, len(typed))
		for _, el := range typed {
			newEl := pruneEmpty(el)
			switch ne := newEl.(type) {
			case nil:
				continue
			case map[string]any:
				if len(ne) == 0 {
					continue
				}
			case []any:
				if len(ne) == 0 {
					continue
				}
			}
			out = append(out, newEl)
		}
		return out
	default:
		return v
	}
}
