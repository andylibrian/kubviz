package kubernetes

import (
	"fmt"
	"strings"
	"time"

	dgraphModel "github.com/intelops/kubviz/model/dgraph"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/jsonpath"
)

func ConvertKubernetesObjectToUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	objUnst := &unstructured.Unstructured{}

	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	objUnst.SetUnstructuredContent(objMap)
	objUnst.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	return objUnst, nil
}

func ConvertUnstructuredToDgraphResource(obj *unstructured.Unstructured) (*dgraphModel.KubernetesResource, error) {
	resource := &dgraphModel.KubernetesResource{
		DgraphType: []string{"KubernetesResource"},
		Group:      obj.GroupVersionKind().Group,
		APIVersion: obj.GroupVersionKind().Version,
		Kind:       obj.GroupVersionKind().Kind,
	}

	resource.MetadataName = obj.GetName()
	resource.MetadataNamespace = obj.GetNamespace()
	resource.MetadataUID = string(obj.GetUID())

	// Extract labels
	for k, v := range obj.GetLabels() {
		resource.MetadataLabels = append(resource.MetadataLabels, dgraphModel.Label{
			Key:   k,
			Value: v,
		})
	}

	resource.IsCurrent = true
	resource.LastUpdated = time.Now()

	// Example for specific fields
	if obj.GetKind() == "Pod" {
		resource.StatusPhase = getNestedString(obj.Object, "status", "phase")
		resource.SpecNodeName = getNestedString(obj.Object, "spec", "nodeName")
	}

	// Extract additional fields
	resource.AdditionalFields = make(map[string]string)
	if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
		for k, v := range spec {
			resource.AdditionalFields[fmt.Sprintf("spec_%s", k)] = fmt.Sprintf("%v", v)
		}
	}

	return resource, nil
}

func getNestedString(obj map[string]interface{}, fields ...string) string {
	val, found, err := unstructured.NestedString(obj, fields...)
	if !found || err != nil {
		return ""
	}
	return val
}

func BuildRelationships(obj *unstructured.Unstructured) []RelationshipInfo {
	relationships := []RelationshipInfo{}

	// Always check default paths
	for jsonPath, relDef := range defaultRelationships {
		relationships = append(relationships, buildRelationshipsFromPath(obj, jsonPath, relDef)...)
	}

	// Check type-specific relationships
	group := obj.GetObjectKind().GroupVersionKind().Group
	version := obj.GetObjectKind().GroupVersionKind().Version
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	if groupRelationships, ok := kubernetesRelationships[group]; ok {
		if versionRelationships, ok := groupRelationships[version]; ok {
			if kindRelationships, ok := versionRelationships[kind]; ok {
				for jsonPath, relDef := range kindRelationships {
					relationships = append(relationships, buildRelationshipsFromPath(obj, jsonPath, relDef)...)
				}
			}
		}
	}

	return relationships
}

func buildRelationshipsFromPath(obj *unstructured.Unstructured, jsonPath string, relDef RelationshipDef) []RelationshipInfo {
	relationships := []RelationshipInfo{}

	values, err := getValuesFromJSONPath(obj.Object, "{"+jsonPath+"}")
	if err != nil {
		// TODO: log
		// fmt.Printf("Error getting values from JSONPath %s: %v\n", jsonPath, err)
		return relationships
	}

	lastPart := getJSONPathLastPart(jsonPath)

	for _, value := range values {
		if value == "" {
			continue
		}

		valueStr := value.(string)
		targetNamespace := obj.GetNamespace()
		if relDef.TargetResource == "v1/Namespace" {
			targetNamespace = ""
		}

		relInfo := RelationshipInfo{
			TargetResource:   relDef.TargetResource,
			TargetNamespace:  targetNamespace,
			RelationshipType: relDef.RelationshipType,
		}

		if lastPart == "uid" {
			relInfo.TargetK8sUID = valueStr
		} else {
			relInfo.TargetName = valueStr
		}

		relationships = append(relationships, relInfo)
	}

	return relationships
}

func getValuesFromJSONPath(obj interface{}, jsonPathStr string) ([]interface{}, error) {
	j := jsonpath.New("jsonpath")
	j.AllowMissingKeys(true)

	err := j.Parse(jsonPathStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing jsonpath %s: %v", jsonPathStr, err)
	}

	results, err := j.FindResults(obj)
	if err != nil {
		return nil, fmt.Errorf("error finding results for jsonpath %s: %v", jsonPathStr, err)
	}

	var values []interface{}
	for _, result := range results {
		for _, r := range result {
			values = append(values, r.Interface())
		}
	}

	if values == nil {
		values = []interface{}{}
	}

	return values, nil
}

func getJSONPathLastPart(jsonPath string) string {
	parts := strings.Split(jsonPath, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
