package patch

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Patch is a patch that can be applied to a Kubernetes object.
type Patch interface {
	// Type is the PatchType of the patch.
	Type() types.PatchType
	// Data is the raw data representing the patch.
	Data(obj runtime.Object) ([]byte, error)
}

// MergeFrom creates a Patch that patches using the merge-patch strategy with the given object as base.
// The difference between MergeFrom and StrategicMergeFrom lays in the handling of modified list fields.
// When using MergeFrom, existing lists will be completely replaced by new lists.
// When using StrategicMergeFrom, the list field's `patchStrategy` is respected if specified in the API type,
// e.g. the existing list is not replaced completely but rather merged with the new one using the list's `patchMergeKey`.
// See https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/ for more details on
// the difference between merge-patch and strategic-merge-patch.
func MergeFrom(obj runtime.Object) Patch {
	return &mergeFromPatch{patchType: types.MergePatchType, createPatch: createMergePatch, from: obj}
}

type mergeFromPatch struct {
	patchType   types.PatchType
	createPatch func(originalJSON, modifiedJSON []byte, dataStruct interface{}) ([]byte, error)
	from        runtime.Object
}

// Type implements patch.
func (s *mergeFromPatch) Type() types.PatchType {
	return s.patchType
}

// Data implements Patch.
func (s *mergeFromPatch) Data(obj runtime.Object) ([]byte, error) {
	originalJSON, err := json.Marshal(s.from)
	if err != nil {
		return nil, err
	}

	modifiedJSON, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	data, err := s.createPatch(originalJSON, modifiedJSON, obj)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func createMergePatch(originalJSON, modifiedJSON []byte, _ interface{}) ([]byte, error) {
	return jsonpatch.CreateMergePatch(originalJSON, modifiedJSON)
}
