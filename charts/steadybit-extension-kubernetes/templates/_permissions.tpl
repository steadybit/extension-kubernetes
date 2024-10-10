{{/*
permissions for clusterrole or role
*/}}
{{- define "defaultPermissions" -}}
{{/* Required for Discoveries */}}
  - apiGroups:
      - apps
    resources:
      - deployments
      - replicasets
      - daemonsets
      - statefulsets
    verbs:
      - get
      - list
      - watch
  {{/* Required for Discoveries */}}
  - apiGroups: [""]
    resources:
      - services
      - pods
      - nodes
      - events
      - namespaces
    verbs:
      - get
      - list
      - watch
  {{/* Required for Single-Replica-Advice */}}
  - apiGroups:
      - autoscaling
    resources:
      - horizontalpodautoscalers
    verbs:
      - get
      - list
      - watch
  {{/* Required for Rollout Restart Attack */}}
  - apiGroups:
      - apps
    resources:
      - deployments
    verbs:
      - patch
  {{/* Required for Scale Deployments Attack */}}
  - apiGroups:
      - apps
    resources:
      - deployments/scale
    verbs:
      - get
      - update
      - patch
  {{/* Required for Scale StatefulSets Attack */}}
  - apiGroups:
      - apps
    resources:
      - statefulsets/scale
    verbs:
      - get
      - update
      - patch
  {{/* Required for Delete Pod Attack */}}
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - delete
  {{/* Required for Crash Loop Pod Attack */}}
  - apiGroups: [""]
    resources:
      - pods/exec
    verbs:
      - create
{{- end -}}
