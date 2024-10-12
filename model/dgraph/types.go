package dgraph

import "time"

// KubernetesResource represents a Kubernetes resource in Dgraph
type KubernetesResource struct {
	UID               string            `json:"uid,omitempty"`
	DgraphType        []string          `json:"dgraph.type,omitempty"`
	Group             string            `json:"group,omitempty"`
	APIVersion        string            `json:"apiVersion,omitempty"`
	Kind              string            `json:"kind,omitempty"`
	MetadataName      string            `json:"metadata_name,omitempty"`
	MetadataNamespace string            `json:"metadata_namespace,omitempty"`
	MetadataUID       string            `json:"metadata_uid,omitempty"`
	MetadataLabels    []Label           `json:"metadata_labels,omitempty"`
	StatusPhase       string            `json:"status_phase,omitempty"`
	SpecNodeName      string            `json:"spec_nodeName,omitempty"`
	IsCurrent         bool              `json:"isCurrent,omitempty"`
	LastUpdated       time.Time         `json:"lastUpdated,omitempty"`
	AdditionalFields  map[string]string `json:"additionalFields,omitempty"`
	Relationships     []Relationship    `json:"relationships,omitempty"`
}

// Label represents a Kubernetes label in Dgraph
type Label struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// DgraphRelationship represents a relationship between Kubernetes resources in Dgraph
type Relationship struct {
	UID              string              `json:"uid,omitempty"`
	DgraphType       []string            `json:"dgraph.type,omitempty"`
	TargetResource   string              `json:"targetResource,omitempty"`
	RelationshipType string              `json:"relationshipType,omitempty"`
	TargetUID        string              `json:"targetUID,omitempty"`
	Target           *KubernetesResource `json:"target,omitempty"`
}
