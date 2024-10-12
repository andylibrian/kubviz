package kubernetes

type RelationshipDef struct {
	TargetResource   string
	RelationshipType string
}

type RelationshipInfo struct {
	TargetK8sUID     string
	TargetResource   string
	TargetName       string
	TargetNamespace  string
	RelationshipType string
}

var defaultRelationships = map[string]RelationshipDef{
	".metadata.ownerReferences[*].uid": {"", "owned-by"},
	".metadata.namespace":              {"v1/Namespace", "belongs-to"},
}

var kubernetesRelationships = map[string]map[string]map[string]map[string]RelationshipDef{
	"": { // Core API group
		"v1": {
			"Pod": {
				".spec.volumes[*].configMap.name":                  {"v1/ConfigMap", "mount-volume"},
				".spec.volumes[*].secret.secretName":               {"v1/Secret", "mount-volume"},
				".spec.containers[*].envFrom[*].configMapRef.name": {"v1/ConfigMap", "env-config"},
				".spec.containers[*].envFrom[*].secretRef.name":    {"v1/Secret", "env-config"},
				".spec.serviceAccountName":                         {"v1/ServiceAccount", "use-account"},
			},
		},
	},
	"apps": {
		"v1": {
			"Deployment": {
				".spec.template.spec.volumes[*].configMap.name":                  {"v1/ConfigMap", "mount-volume"},
				".spec.template.spec.volumes[*].secret.secretName":               {"v1/Secret", "mount-volume"},
				".spec.template.spec.containers[*].envFrom[*].configMapRef.name": {"v1/ConfigMap", "env-config"},
				".spec.template.spec.containers[*].envFrom[*].secretRef.name":    {"v1/Secret", "env-config"},
				".spec.template.spec.serviceAccountName":                         {"v1/ServiceAccount", "use-account"},
			},
			"Statefulset": {
				".spec.template.spec.volumes[*].configMap.name":                  {"v1/ConfigMap", "mount-volume"},
				".spec.template.spec.volumes[*].secret.secretName":               {"v1/Secret", "mount-volume"},
				".spec.template.spec.containers[*].envFrom[*].configMapRef.name": {"v1/ConfigMap", "env-config"},
				".spec.template.spec.containers[*].envFrom[*].secretRef.name":    {"v1/Secret", "env-config"},
				".spec.template.spec.serviceAccountName":                         {"v1/ServiceAccount", "use-account"},
			},
		},
	},
	"networking.k8s.io": {
		"v1": {
			"Ingress": {
				".spec.rules[*].http.paths[*].backend.service.name": {"v1/Service", "route-traffic"},
			},
		},
	},
	"rbac.authorization.k8s.io": {
		"v1": {
			"Role": {
				".rules[*].resourceNames[*]": {"", "grant-access"},
			},
			"RoleBinding": {
				".roleRef.name":     {"rbac.authorization.k8s.io/v1/Role", "bind-role"},
				".subjects[*].name": {"", "assign-to"},
			},
		},
	},
}
