package dgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dgraphModel "github.com/intelops/kubviz/model/dgraph"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
)

var ErrResourceNotFound = errors.New("resource not found")

// DgraphKubernetesResourceRepository handles Dgraph operations for KubernetesResource
type DgraphKubernetesResourceRepository struct {
	client *dgo.Dgraph
}

// NewDgraphKubernetesResourceRepository creates a new DgraphKubernetesResourceRepository instance
func NewDgraphKubernetesResourceRepository(client *dgo.Dgraph) *DgraphKubernetesResourceRepository {
	return &DgraphKubernetesResourceRepository{client: client}
}

func (repo *DgraphKubernetesResourceRepository) Save(ctx context.Context, resource *dgraphModel.KubernetesResource) error {
	txn := repo.client.NewTxn()
	defer txn.Discard(ctx)

	existingResource, err := repo.FindByKubernetesUID(ctx, resource.MetadataUID)
	if err != nil && !errors.Is(err, ErrResourceNotFound) {
		return fmt.Errorf("error checking existing resource: %v", err)
	}

	var mutations []*api.Mutation

	if existingResource != nil {
		resource.UID = existingResource.UID

		// Handle upsert labels

		// Compute label diffs
		labelsToAdd, labelsToRemove, _ := diffLabels(existingResource.MetadataLabels, resource.MetadataLabels)

		// Prepare label removal mutation if needed
		if len(labelsToRemove) > 0 {
			deleteLabelsNquads := generateLabelDeleteNquads(resource.UID, labelsToRemove)
			mutations = append(mutations, &api.Mutation{
				DelNquads: []byte(strings.Join(deleteLabelsNquads, "\n")),
			})
		}

		// Prepare label addition mutations if needed
		if len(labelsToAdd) > 0 {
			addLabelsNquads := generateLabelAddNquads(resource.UID, labelsToAdd)
			mutations = append(mutations, &api.Mutation{
				SetNquads: []byte(strings.Join(addLabelsNquads, "\n")),
			})
		}
	}

	resourceCopy := *resource
	resourceCopy.DgraphType = []string{"KubernetesResource"}

	if existingResource != nil {
		// Unsetting labels here to be handled in a separate mutation
		resourceCopy.MetadataLabels = nil
	}

	resourceJSON, err := json.Marshal(resourceCopy)
	if err != nil {
		return fmt.Errorf("error json marshaling resource: %v", err)
	}

	mutations = append(mutations, &api.Mutation{
		SetJson: resourceJSON,
	})

	// Perform all mutations in a single request
	request := &api.Request{
		Mutations: mutations,
		CommitNow: true,
	}

	_, err = txn.Do(ctx, request)
	if err != nil {
		return fmt.Errorf("error performing mutations: %v", err)
	}

	return nil
}

func diffLabels(oldLabels, newLabels []dgraphModel.Label) (toAdd, toRemove, toUpdate []dgraphModel.Label) {
	oldMap := make(map[string]string)
	for _, label := range oldLabels {
		oldMap[label.Key] = label.Value
	}

	newMap := make(map[string]string)
	for _, label := range newLabels {
		newMap[label.Key] = label.Value
	}

	toAdd = []dgraphModel.Label{}
	toRemove = []dgraphModel.Label{}
	toUpdate = []dgraphModel.Label{}

	for _, newLabel := range newLabels {
		if oldValue, exists := oldMap[newLabel.Key]; !exists {
			toAdd = append(toAdd, newLabel)
		} else if oldValue != newLabel.Value {
			toUpdate = append(toUpdate, newLabel)
		}
	}

	for _, oldLabel := range oldLabels {
		if _, exists := newMap[oldLabel.Key]; !exists {
			toRemove = append(toRemove, oldLabel)
		}
	}

	return toAdd, toRemove, toUpdate
}

func generateLabelDeleteNquads(uid string, labels []dgraphModel.Label) []string {
	var nquads []string
	for _, label := range labels {
		nquad := fmt.Sprintf("<%s> <metadata_labels> * (key = %q) .", uid, label.Key)
		nquads = append(nquads, nquad)
	}
	return nquads
}

func generateLabelAddNquads(uid string, labelsToAdd []dgraphModel.Label) []string {
	var nquads []string
	for _, label := range labelsToAdd {
		labelUID := fmt.Sprintf("_:label_%s", label.Key)
		nquads = append(nquads, fmt.Sprintf("<%s> <metadata_labels> %s .", uid, labelUID))
		nquads = append(nquads, fmt.Sprintf("%s <dgraph.type> \"Label\" .", labelUID))
		nquads = append(nquads, fmt.Sprintf("%s <key> %q .", labelUID, label.Key))
		nquads = append(nquads, fmt.Sprintf("%s <value> %q .", labelUID, label.Value))
	}
	return nquads
}

// getResourceQueryFields returns the common query fields for KubernetesResource
func (repo *DgraphKubernetesResourceRepository) getResourceQueryFields() string {
	return `
		uid
		dgraph.type
		group
		apiVersion
		kind
		metadata_name
		metadata_namespace
		metadata_uid
		metadata_labels {
			key
			value
		}
		status_phase
		spec_nodeName
		isCurrent
		lastUpdated
		additionalFields
		relationships {
			uid
			dgraph.type
			targetResource
			relationshipType
			targetUID
			target {
				uid
				metadata_name
				metadata_namespace
				metadata_uid
			}
		}
	`
}

// FindByDgraphUID retrieves a KubernetesResource by its Dgraph UID
func (repo *DgraphKubernetesResourceRepository) FindByDgraphUID(ctx context.Context, dgraphUID string) (*dgraphModel.KubernetesResource, error) {
	txn := repo.client.NewTxn()
	defer txn.Discard(ctx)

	q := fmt.Sprintf(`
		query findResource($dgraphUID: string) {
			resources(func: uid($dgraphUID)) @filter(eq(dgraph.type, "KubernetesResource")) {
				%s
			}
		}
	`, repo.getResourceQueryFields())

	vars := map[string]string{"$dgraphUID": dgraphUID}
	resp, err := txn.QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, fmt.Errorf("error querying Dgraph: %v", err)
	}

	var result struct {
		Resources []dgraphModel.KubernetesResource `json:"resources"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(result.Resources) == 0 {
		return nil, ErrResourceNotFound
	}

	return &result.Resources[0], nil
}

// FindByKubernetesUID retrieves a KubernetesResource by its Kubernetes UID
func (repo *DgraphKubernetesResourceRepository) FindByKubernetesUID(ctx context.Context, k8sUID string) (*dgraphModel.KubernetesResource, error) {
	txn := repo.client.NewTxn()
	defer txn.Discard(ctx)

	q := fmt.Sprintf(`
		query findResource($k8sUID: string) {
			resources(func: eq(metadata_uid, $k8sUID)) @filter(eq(dgraph.type, "KubernetesResource")) {
				%s
			}
		}
	`, repo.getResourceQueryFields())

	vars := map[string]string{"$k8sUID": k8sUID}
	resp, err := txn.QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, fmt.Errorf("error querying Dgraph: %v", err)
	}

	var result struct {
		Resources []dgraphModel.KubernetesResource `json:"resources"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(result.Resources) == 0 {
		return nil, ErrResourceNotFound
	}

	return &result.Resources[0], nil
}
