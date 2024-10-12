package dgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"github.com/intelops/kubviz/client/pkg/kubernetes"
)

type DgraphRelationshipRepository struct {
	client *dgo.Dgraph
}

func NewDgraphRelationshipRepository(client *dgo.Dgraph) *DgraphRelationshipRepository {
	return &DgraphRelationshipRepository{client: client}
}

func (repo *DgraphRelationshipRepository) Save(ctx context.Context, sourceUID string, relationships []kubernetes.RelationshipInfo) error {
	txn := repo.client.NewTxn()
	defer txn.Discard(ctx)

	// Find the source Dgraph UID
	sourceDgraphUID, err := FindDgraphUIDByKubernetesUID(ctx, repo.client, sourceUID)
	if err != nil {
		return fmt.Errorf("failed to find source Dgraph UID: %v", err)
	}

	// Prepare a mutation to delete existing relationships
	deleteNquads := []string{
		fmt.Sprintf("<%s> <relationships> * .", sourceDgraphUID),
	}

	var setNquadsAll []string

	i := 0
	for _, info := range relationships {
		var setNquads []string
		spew.Dump(info)
		var targetDgraphUID string
		if info.TargetK8sUID == "" {
			// Resolve by name
			targetDgraphUID, err = getDgraphResourceUID(ctx, repo.client, info.TargetName, info.TargetNamespace, info.TargetResource)
			if err != nil {
				// TODO: logs
				// fmt.Printf("error while finding Dgraph UID for target: %s\n", err.Error())
				continue
			}
		} else {
			// Resolve by UID
			targetDgraphUID, err = FindDgraphUIDByKubernetesUID(ctx, repo.client, info.TargetK8sUID)
			if err != nil {
				// TODO: logs
				// fmt.Printf("error while finding Dgraph UID for target: %s\n", err.Error())
				continue
			}
		}

		// Create a new relationship node
		relationshipUID := fmt.Sprintf("_:rel%d", i) // Create a blank node
		setNquads = append(setNquads, fmt.Sprintf("%s <dgraph.type> \"Relationship\" .", relationshipUID))
		setNquads = append(setNquads, fmt.Sprintf("%s <targetResource> %q .", relationshipUID, info.TargetResource))
		setNquads = append(setNquads, fmt.Sprintf("%s <relationshipType> %q .", relationshipUID, info.RelationshipType))

		if info.TargetK8sUID != "" {
			setNquads = append(setNquads, fmt.Sprintf("%s <targetUID> %q .", relationshipUID, info.TargetK8sUID))
		}

		// Link the new relationship to the target object:
		// Relationship -> Target
		setNquads = append(setNquads, fmt.Sprintf("%s <target> <%s> .", relationshipUID, targetDgraphUID))

		// Link the new relationship to the source object:
		// Source -> Relationship -> Target
		setNquads = append(setNquads, fmt.Sprintf("<%s> <relationships> %s .", sourceDgraphUID, relationshipUID))

		setNquadsAll = append(setNquadsAll, setNquads...)

		i++
	}

	// Combine delete and set mutations
	mutation := &api.Mutation{
		DelNquads: []byte(strings.Join(deleteNquads, "\n")),
		SetNquads: []byte(strings.Join(setNquadsAll, "\n")),
	}

	// Perform the mutation
	_, err = txn.Mutate(ctx, mutation)
	if err != nil {
		return fmt.Errorf("failed to update relationships: %v", err)
	}

	// Commit the transaction
	return txn.Commit(ctx)
}

// FindDgraphUIDByKubernetesUID retrieves the Dgraph UID for a given Kubernetes UID
func FindDgraphUIDByKubernetesUID(ctx context.Context, client *dgo.Dgraph, k8sUID string) (string, error) {
	txn := client.NewTxn()
	defer txn.Discard(ctx)

	query := `
    query findDgraphUID($k8sUID: string) {
        resource(func: eq(metadata_uid, $k8sUID)) {
            uid
        }
    }
    `
	vars := map[string]string{"$k8sUID": k8sUID}
	resp, err := txn.QueryWithVars(ctx, query, vars)
	if err != nil {
		return "", fmt.Errorf("failed to query Dgraph: %v", err)
	}

	var result struct {
		Resource []struct {
			UID string `json:"uid"`
		} `json:"resource"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal Dgraph response: %v", err)
	}

	if len(result.Resource) == 0 {
		return "", fmt.Errorf("resource not found for UID: %s", k8sUID)
	}

	return result.Resource[0].UID, nil
}

func getDgraphResourceUID(ctx context.Context, dgraphClient *dgo.Dgraph, name, namespace, targetResource string) (string, error) {
	parts := strings.SplitN(targetResource, "/", 3)

	group := ""
	kind := ""

	if len(parts) == 1 {
		kind = parts[0]
	} else if len(parts) == 2 {
		kind = parts[1]
	} else {
		group = parts[0]
		kind = parts[2]
	}

	var query string
	var groupFilter string

	if group == "" {
		groupFilter = "(not(has(group)) OR eq(group, \"\"))"
	} else {
		groupFilter = "eq(group, $group)"
	}

	if namespace != "" {
		// Query for namespaced resources
		query = fmt.Sprintf(`query resource($name: string, $namespace: string, $group: string, $kind: string) {
            resource(func: eq(dgraph.type, "KubernetesResource"))
            @filter(eq(metadata_name, $name) AND
                    eq(metadata_namespace, $namespace) AND
                    %s AND
                    eq(kind, $kind)) {
                uid
            }
        }`, groupFilter)
	} else {
		// Query for cluster-scoped resources
		query = fmt.Sprintf(`query resource($name: string, $group: string, $kind: string) {
            resource(func: eq(dgraph.type, "KubernetesResource"))
            @filter(eq(metadata_name, $name) AND
                    %s AND
                    eq(kind, $kind)) {
                uid
            }
        }`, groupFilter)
	}

	vars := map[string]string{
		"$name":  name,
		"$group": group,
		"$kind":  kind,
	}
	if namespace != "" {
		vars["$namespace"] = namespace
	}

	resp, err := dgraphClient.NewTxn().QueryWithVars(ctx, query, vars)
	if err != nil {
		return "", fmt.Errorf("failed to query Dgraph: %v", err)
	}

	var result struct {
		Resource []struct {
			UID string `json:"uid"`
		} `json:"resource"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal Dgraph response: %v", err)
	}

	if len(result.Resource) == 0 {
		return "", fmt.Errorf("resource not found: ns: %s , name: %s (group: %s, kind: %s)", namespace, name, group, kind)
	}

	return result.Resource[0].UID, nil
}
