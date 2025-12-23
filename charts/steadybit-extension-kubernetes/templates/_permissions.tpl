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
  {{- if not .Values.discovery.disabled.deployment }}
  {{/* Required for Rollout Restart Attack */}}
  - apiGroups:
      - apps
    resources:
      - deployments
    verbs:
      - patch
  {{- end }}
  {{- if not .Values.discovery.disabled.deployment }}
  {{/* Required for Scale Deployments Attack */}}
  - apiGroups:
      - apps
    resources:
      - deployments/scale
    verbs:
      - get
      - update
      - patch
  {{- end }}
  {{- if not .Values.discovery.disabled.replicaSet }}
  {{/* Required for Scale ReplicaSet Attack */}}
  - apiGroups:
      - apps
    resources:
      - replicasets/scale
    verbs:
      - get
      - update
      - patch
  {{- end }}
  {{- if not .Values.discovery.disabled.statefulSet }}
  {{/* Required for Scale StatefulSets Attack */}}
  - apiGroups:
      - apps
    resources:
      - statefulsets/scale
    verbs:
      - get
      - update
      - patch
  {{- end }}
  {{- if not .Values.discovery.disabled.pod }}
  {{/* Required for Delete Pod Attack */}}
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - delete
  {{- end }}
  {{- if not .Values.discovery.disabled.pod }}
  {{/* Required for Crash Loop Pod Attack */}}
  - apiGroups: [""]
    resources:
      - pods/exec
    verbs:
      - create
  {{- end }}
  {{- if not .Values.discovery.disabled.argoRollout }}
  {{/* Required for Argo Discovery */}}
  - apiGroups: ["argoproj.io"]
    resources:
      - rollouts
    verbs:
      - get
      - list
      - watch
  {{- end }}
{{- end -}}
