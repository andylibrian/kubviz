package kubeallresources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/intelops/kubviz/constants"
	"github.com/intelops/kubviz/pkg/opentelemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	kubevizotel "github.com/intelops/kubviz/agent/kubviz/otel"
)

var ClusterName string = os.Getenv("CLUSTER_NAME")

func PublishAllResources(config *rest.Config) error {
	ctx := context.Background()
	tracer := otel.Tracer("kubeallresources")
	_, span := tracer.Start(opentelemetry.BuildContext(ctx), "PublishAllResources")
	span.SetAttributes(attribute.String("kubeallresources-plugin-agent", "kubeallresources-output"))
	defer span.End()

	// Create a new discovery client to discover all resources in the cluster
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)

	// Create a new dynamic client to list resources in the cluster
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	// Get a list of all available API groups and versions in the cluster
	resourceLists, err := dc.ServerPreferredResources()
	if err != nil {
		return err
	}
	gvrs, err := discovery.GroupVersionResources(resourceLists)
	if err != nil {
		return err
	}
	// Iterate over all available API groups and versions and list all resources in each group
	for gvr := range gvrs {
		// List all resources in the group
		list, err := dynamicClient.Resource(gvr).Namespace("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error listing %s: %v\n", gvr.String(), err)
			continue
		}

		for _, item := range list.Items {
			err := publishUnstructuredObject(&item)
			if err != nil {
				fmt.Println("error while publishing unstructured object", err)
				continue
			}
		}
	}

	return nil
}

func publishUnstructuredObject(obj *unstructured.Unstructured) error {
	objJson, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	kubevizotel.PublishEventLog(constants.EventSubject_kubeallresources, objJson)
	return nil
}
