package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestConvertUnstructuredToDgraphResource(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
				"uid":       "12345",
				"labels": map[string]interface{}{
					"app": "test",
				},
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
			"spec": map[string]interface{}{
				"nodeName": "node1",
			},
		},
	}

	result, err := ConvertUnstructuredToDgraphResource(obj)

	assert.NoError(t, err)
	assert.Equal(t, "v1", result.APIVersion)
	assert.Equal(t, "Pod", result.Kind)
	assert.Equal(t, "test-pod", result.MetadataName)
	assert.Equal(t, "default", result.MetadataNamespace)
	assert.Equal(t, "12345", result.MetadataUID)
	assert.Equal(t, "Running", result.StatusPhase)
	assert.Equal(t, "node1", result.SpecNodeName)
	assert.Len(t, result.MetadataLabels, 1)
	assert.Equal(t, "app", result.MetadataLabels[0].Key)
	assert.Equal(t, "test", result.MetadataLabels[0].Value)
}

func TestGetNestedString(t *testing.T) {
	obj := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": "value",
			},
		},
	}

	tests := []struct {
		name     string
		obj      map[string]interface{}
		fields   []string
		expected string
	}{
		{"Existing nested field", obj, []string{"a", "b", "c"}, "value"},
		{"Non-existent field", obj, []string{"x", "y", "z"}, ""},
		{"Partially existing field", obj, []string{"a", "b", "d"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNestedString(tt.obj, tt.fields...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildRelationships(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"kind": "Deployment",
						"name": "test-deployment",
						"uid":  "67890",
					},
				},
			},
			"spec": map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"name": "config",
						"configMap": map[string]interface{}{
							"name": "test-config",
						},
					},
				},
			},
		},
	}

	relationships := BuildRelationships(obj)

	assert.Len(t, relationships, 3)

	// Check owner relationship
	ownerRel := relationships[0]
	assert.Equal(t, "owned-by", ownerRel.RelationshipType)
	assert.Equal(t, "67890", ownerRel.TargetK8sUID)

	// Check namespace relationship
	namespaceRel := relationships[1]
	assert.Equal(t, "v1/Namespace", namespaceRel.TargetResource)
	assert.Equal(t, "belongs-to", namespaceRel.RelationshipType)
	assert.Equal(t, "default", namespaceRel.TargetName)

	// Check ConfigMap relationship
	configMapRel := relationships[2]
	assert.Equal(t, "v1/ConfigMap", configMapRel.TargetResource)
	assert.Equal(t, "mount-volume", configMapRel.RelationshipType)
	assert.Equal(t, "test-config", configMapRel.TargetName)
}

func TestGetValuesFromJSONPath(t *testing.T) {
	obj := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "item1"},
			map[string]interface{}{"name": "item2"},
		},
	}

	tests := []struct {
		name         string
		obj          interface{}
		jsonPath     string
		expectedVals []interface{}
		expectError  bool
	}{
		{"Valid path", obj, "{.items[*].name}", []interface{}{"item1", "item2"}, false},
		{"Invalid path", obj, "{.nonexistent}", []interface{}{}, false},
		{"Syntax error", obj, "{.items[*}", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := getValuesFromJSONPath(tt.obj, tt.jsonPath)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVals, values)
			}
		})
	}
}

func TestGetJSONPathLastPart(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		expected string
	}{
		{"Simple path", "metadata.name", "name"},
		{"Complex path", "spec.template.spec.containers[*].name", "name"},
		{"Empty path", "", ""},
		{"Single part", "uid", "uid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJSONPathLastPart(tt.jsonPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
