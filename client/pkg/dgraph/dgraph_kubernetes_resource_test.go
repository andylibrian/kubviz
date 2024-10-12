package dgraph

import (
	"testing"

	dgraphModel "github.com/intelops/kubviz/model/dgraph"

	"github.com/stretchr/testify/assert"
)

func TestDiffLabels(t *testing.T) {
	tests := []struct {
		name         string
		oldLabels    []dgraphModel.Label
		newLabels    []dgraphModel.Label
		expectAdd    []dgraphModel.Label
		expectRemove []dgraphModel.Label
		expectUpdate []dgraphModel.Label
	}{
		{
			name:         "Add new label",
			oldLabels:    []dgraphModel.Label{{Key: "app", Value: "test"}},
			newLabels:    []dgraphModel.Label{{Key: "app", Value: "test"}, {Key: "env", Value: "prod"}},
			expectAdd:    []dgraphModel.Label{{Key: "env", Value: "prod"}},
			expectRemove: []dgraphModel.Label{},
			expectUpdate: []dgraphModel.Label{},
		},
		{
			name:         "Remove label",
			oldLabels:    []dgraphModel.Label{{Key: "app", Value: "test"}, {Key: "env", Value: "prod"}},
			newLabels:    []dgraphModel.Label{{Key: "app", Value: "test"}},
			expectAdd:    []dgraphModel.Label{},
			expectRemove: []dgraphModel.Label{{Key: "env", Value: "prod"}},
			expectUpdate: []dgraphModel.Label{},
		},
		{
			name:         "Update label",
			oldLabels:    []dgraphModel.Label{{Key: "app", Value: "test"}},
			newLabels:    []dgraphModel.Label{{Key: "app", Value: "newtest"}},
			expectAdd:    []dgraphModel.Label{},
			expectRemove: []dgraphModel.Label{},
			expectUpdate: []dgraphModel.Label{{Key: "app", Value: "newtest"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			add, remove, update := diffLabels(tt.oldLabels, tt.newLabels)
			assert.Equal(t, tt.expectAdd, add, "Added labels mismatch")
			assert.Equal(t, tt.expectRemove, remove, "Removed labels mismatch")
			assert.Equal(t, tt.expectUpdate, update, "Updated labels mismatch")

			// Additional checks to ensure slices are never nil
			assert.NotNil(t, add, "Added labels should not be nil")
			assert.NotNil(t, remove, "Removed labels should not be nil")
			assert.NotNil(t, update, "Updated labels should not be nil")
		})
	}
}

func TestGenerateLabelDeleteNquads(t *testing.T) {
	uid := "0x1234"
	labels := []dgraphModel.Label{
		{Key: "app", Value: "test"},
		{Key: "env", Value: "prod"},
	}

	expected := []string{
		`<0x1234> <metadata_labels> * (key = "app") .`,
		`<0x1234> <metadata_labels> * (key = "env") .`,
	}

	result := generateLabelDeleteNquads(uid, labels)
	assert.Equal(t, expected, result)
}

func TestGenerateLabelAddNquads(t *testing.T) {
	uid := "0x1234"
	labels := []dgraphModel.Label{
		{Key: "app", Value: "test"},
		{Key: "env", Value: "prod"},
	}

	expected := []string{
		`<0x1234> <metadata_labels> _:label_app .`,
		`_:label_app <dgraph.type> "Label" .`,
		`_:label_app <key> "app" .`,
		`_:label_app <value> "test" .`,
		`<0x1234> <metadata_labels> _:label_env .`,
		`_:label_env <dgraph.type> "Label" .`,
		`_:label_env <key> "env" .`,
		`_:label_env <value> "prod" .`,
	}

	result := generateLabelAddNquads(uid, labels)
	assert.Equal(t, expected, result)
}
