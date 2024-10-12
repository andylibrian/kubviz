package dgraph

import (
	"context"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
)

// Dgraph schema for Kubernetes resources
const dgraphSchema = `
group: string @index(exact) .
apiVersion: string @index(exact) .
kind: string @index(exact) .
metadata_name: string @index(exact) .
metadata_namespace: string @index(exact) .
metadata_uid: string @index(exact) .
metadata_labels: [uid] . 
status_phase: string @index(exact) .
spec_nodeName: string @index(exact) .
isCurrent: bool @index(bool) .
lastUpdated: datetime .

key: string @index(exact) .
value: string .

relationships: [uid] @reverse .
targetResource: string @index(exact) .
relationshipType: string @index(exact) .
targetUID: string @index(exact) .
target: uid .

type KubernetesResource {
	dgraph.type: string
	group: string
	apiVersion: string
	kind: string

	metadata_name: string
	metadata_namespace: string
	metadata_uid: string!
	metadata_labels: [Label]

	isCurrent: bool
	lastUpdated: DateTime

	status_phase: string
	spec_nodeName: string
	relationships: [Relationship]
}

type Label {
	key: string
	value: string
}

type Relationship {
	dgraph.type: string
	targetResource: string
	relationshipType: string
	targetUID: string
	target: KubernetesResource
}
`

// SetupDgraphSchema sets up the Dgraph schema for Kubernetes resources
func SetupDgraphSchema(ctx context.Context, dgraphClient *dgo.Dgraph) error {
	op := &api.Operation{
		Schema: dgraphSchema,
	}
	return dgraphClient.Alter(ctx, op)
}
