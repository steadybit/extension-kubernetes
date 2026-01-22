package client

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-kubernetes/v2/extconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	authorizationv1 "k8s.io/api/authorization/v1"
)

type PermissionCheckResult struct {
	Permissions map[string]PermissionCheckOutcome
}

type PermissionCheckOutcome string

const (
	WARN  PermissionCheckOutcome = "warn"
	ERROR PermissionCheckOutcome = "error"
	OK    PermissionCheckOutcome = "ok"
)

type requiredPermission struct {
	verbs                []string
	group                string
	resource             string
	subresource          string
	allowGracefulFailure bool
}

func (p *requiredPermission) Key(verb string) string {
	result := ""
	if p.group != "" {
		result = p.group + "/"
	}
	result = result + p.resource + "/"
	if p.subresource != "" {
		result = result + p.subresource + "/"
	}
	result = result + verb
	return result
}

var requiredPermissions = []requiredPermission{
	{group: "apps", resource: "deployments", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "apps", resource: "replicasets", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "apps", resource: "daemonsets", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "apps", resource: "statefulsets", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "autoscaling", resource: "horizontalpodautoscalers", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: true},
	{group: "", resource: "services", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "", resource: "pods", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "", resource: "namespaces", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: true},
	{group: "", resource: "nodes", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "", resource: "events", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: false},
	{group: "apps", resource: "deployments", verbs: []string{"patch"}, allowGracefulFailure: true},
	{group: "apps", resource: "deployments", subresource: "scale", verbs: []string{"get", "update", "patch"}, allowGracefulFailure: true},
	{group: "apps", resource: "replicasets", subresource: "scale", verbs: []string{"get", "update", "patch"}, allowGracefulFailure: true},
	{group: "apps", resource: "statefulsets", subresource: "scale", verbs: []string{"get", "update", "patch"}, allowGracefulFailure: true},
	{group: "", resource: "pods", verbs: []string{"delete"}, allowGracefulFailure: true},
	{group: "", resource: "pods", subresource: "eviction", verbs: []string{"create"}, allowGracefulFailure: true},
	{group: "", resource: "nodes", verbs: []string{"patch"}, allowGracefulFailure: true},
	{group: "", resource: "pods", subresource: "exec", verbs: []string{"create"}, allowGracefulFailure: true},
	{group: "networking.k8s.io", resource: "ingresses", verbs: []string{"get", "list", "watch", "update", "patch"}, allowGracefulFailure: true},
	{group: "networking.k8s.io", resource: "ingressclasses", verbs: []string{"get", "list", "watch"}, allowGracefulFailure: true},
}

var (
	argoRolloutPermissionOnce sync.Once
)

// ensureArgoRolloutPermission adds the Argo Rollout permission to requiredPermissions if needed.
// This is safe to call multiple times - it will only add the permission once.
func ensureArgoRolloutPermission() {
	if !extconfig.Config.DiscoveryDisabledArgoRollout {
		argoRolloutPermissionOnce.Do(func() {
			requiredPermissions = append(requiredPermissions, requiredPermission{
				group: "argoproj.io", resource: "rollouts", verbs: []string{"get", "list", "watch", "patch"}, allowGracefulFailure: true,
			})
		})
	}
}

func checkPermissions(client *kubernetes.Clientset) *PermissionCheckResult {
	ensureArgoRolloutPermission()

	result := make(map[string]PermissionCheckOutcome)
	reviews := client.AuthorizationV1().SelfSubjectAccessReviews()
	errors := false

	for _, p := range requiredPermissions {
		for _, verb := range p.verbs {
			sar := authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Namespace:   extconfig.Config.Namespace,
						Verb:        verb,
						Resource:    p.resource,
						Subresource: p.subresource,
						Group:       p.group,
					},
				},
			}
			review, err := reviews.Create(context.TODO(), &sar, metav1.CreateOptions{})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to check permission %s", p.Key(verb))
			}
			if err != nil || !review.Status.Allowed {
				if p.allowGracefulFailure {
					result[p.Key(verb)] = WARN
				} else {
					result[p.Key(verb)] = ERROR
					errors = true
				}
			} else {
				result[p.Key(verb)] = OK
			}
		}
	}

	logPermissionCheckResult(result)
	if errors {
		log.Fatal().Msg("Required permissions are missing. Exit now.")
	}

	return &PermissionCheckResult{
		Permissions: result,
	}
}

func logPermissionCheckResult(permissions map[string]PermissionCheckOutcome) {
	log.Info().Msg("Permission check results:")
	allGood := true
	for k, v := range permissions {
		if v == OK {
			log.Debug().Str("permission", k).Str("result", string(v)).Msg("Permission granted.")
		} else if v == WARN {
			log.Warn().Str("permission", k).Str("result", string(v)).Msg("Permission missing, but not required. Some features may not work - see documentation for details.")
			allGood = false
		} else if v == ERROR {
			log.Error().Str("permission", k).Str("result", string(v)).Msg("Permission missing.")
			allGood = false
		}
	}
	if allGood {
		log.Info().Msg("All permissions granted.")
	}
}

func (p *PermissionCheckResult) hasPermissions(requiredPermissions []string) bool {
	for _, rp := range requiredPermissions {
		outcome, ok := p.Permissions[rp]
		if !ok || outcome != OK {
			return false
		}
	}
	return true
}

func (p *PermissionCheckResult) CanReadHorizontalPodAutoscalers() bool {
	return p.hasPermissions([]string{
		"autoscaling/horizontalpodautoscalers/get",
		"autoscaling/horizontalpodautoscalers/list",
		"autoscaling/horizontalpodautoscalers/watch"})
}

func (p *PermissionCheckResult) CanReadNamespaces() bool {
	return p.hasPermissions([]string{
		"namespaces/get",
		"namespaces/list",
		"namespaces/watch"})
}

func (p *PermissionCheckResult) IsRolloutRestartPermitted() bool {
	return p.hasPermissions([]string{
		"apps/deployments/patch",
	})
}

func (p *PermissionCheckResult) IsScaleDeploymentPermitted() bool {
	return p.hasPermissions([]string{
		"apps/deployments/scale/get",
		"apps/deployments/scale/update",
		"apps/deployments/scale/patch",
	})
}

func (p *PermissionCheckResult) IsScaleReplicaSetPermitted() bool {
	return p.hasPermissions([]string{
		"apps/replicasets/scale/get",
		"apps/replicasets/scale/update",
		"apps/replicasets/scale/patch",
	})
}

func (p *PermissionCheckResult) IsScaleStatefulSetPermitted() bool {
	return p.hasPermissions([]string{
		"apps/statefulsets/scale/get",
		"apps/statefulsets/scale/update",
		"apps/statefulsets/scale/patch",
	})
}

func (p *PermissionCheckResult) IsDeletePodPermitted() bool {
	return p.hasPermissions([]string{
		"pods/delete",
	})
}

func (p *PermissionCheckResult) IsDrainNodePermitted() bool {
	return p.hasPermissions([]string{
		"pods/eviction/create",
		"nodes/patch",
	})
}

func (p *PermissionCheckResult) IsListIngressPermitted() bool {
	return p.hasPermissions([]string{
		"networking.k8s.io/ingresses/get",
		"networking.k8s.io/ingresses/list",
		"networking.k8s.io/ingresses/watch",
	})
}

func (p *PermissionCheckResult) IsListIngressClassesPermitted() bool {
	return p.hasPermissions([]string{
		"networking.k8s.io/ingressclasses/get",
		"networking.k8s.io/ingressclasses/list",
		"networking.k8s.io/ingressclasses/watch",
	})
}

func (p *PermissionCheckResult) IsModifyIngressPermitted() bool {
	return p.hasPermissions([]string{
		"networking.k8s.io/ingresses/update",
		"networking.k8s.io/ingresses/patch",
	})
}

func (p *PermissionCheckResult) IsTaintNodePermitted() bool {
	return p.hasPermissions([]string{
		"pods/eviction/create",
		"nodes/patch",
	})
}

func (p *PermissionCheckResult) IsCrashLoopPodPermitted() bool {
	return p.hasPermissions([]string{
		"pods/exec/create",
	})
}

func (p *PermissionCheckResult) IsSetImageDeploymentPermitted() bool {
	return p.hasPermissions([]string{
		"apps/deployments/get",
		"apps/deployments/patch",
	})
}

func (p *PermissionCheckResult) IsArgoRolloutRestartPermitted() bool {
	return p.hasPermissions([]string{
		"argoproj.io/rollouts/patch",
	})
}

func MockAllPermitted() *PermissionCheckResult {
	ensureArgoRolloutPermission()

	result := make(map[string]PermissionCheckOutcome)

	for _, p := range requiredPermissions {
		for _, verb := range p.verbs {
			result[p.Key(verb)] = OK
		}
	}
	return &PermissionCheckResult{
		Permissions: result,
	}
}
