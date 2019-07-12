package configutil

import (
	"testing"

	v1 "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestSimpleMerge(t *testing.T) {
	deployment1 := ptr.String("testDeployment1")
	deployment2 := ptr.String("testDeployment2")
	version := ptr.String("testVersion")
	newImageTag := ptr.String("newImageTag")

	object1 := &v1.Config{
		Version: ptr.String("oldVersion"),
		Deployments: &[]*v1.DeploymentConfig{
			&v1.DeploymentConfig{
				Name: ptr.String("oldDeployment"),
			},
		},
		Images: &map[string]*v1.ImageConfig{
			"testImage": &v1.ImageConfig{
				Tag: ptr.String("old"),
			},
		},
	}

	object2 := &v1.Config{
		Version: version,
		Deployments: &[]*v1.DeploymentConfig{
			&v1.DeploymentConfig{
				Name: deployment1,
			},
			&v1.DeploymentConfig{
				Name: deployment2,
			},
		},
		Images: &map[string]*v1.ImageConfig{
			"testImage": &v1.ImageConfig{
				Tag: newImageTag,
			},
		},
	}

	// Merge object2 in object1
	Merge(&object1, object2)

	if object1.Version == nil || object1.Version != version {
		t.Fatal("Version is not equal")
	}
	if object1.Deployments == nil || len(*object1.Deployments) != 2 || (*object1.Deployments)[0].Name != deployment1 || (*object1.Deployments)[1].Name != deployment2 {
		t.Fatal("Deployments are not correct")
	}
	if object1.Images == nil || len(*object1.Images) != 1 || (*object1.Images)["testImage"].Tag != newImageTag {
		t.Fatal("Deployments are not correct")
	}
}

type testStruct1 struct {
	Field1 *string
}

func TestMapMergeUntrivialKey(t *testing.T) {
	value := ptr.String("someString")
	key := testStruct1{}

	object1 := &map[interface{}]interface{}{
		"nil": "nil",
	}

	object2 := &map[interface{}]interface{}{
		key:   value,
		"nil": nil,
	}

	// Merge object2 in object1
	Merge(&object1, object2)
	assert.Equal(t, (*object1)[key], value, "Value of untrivial key in map is not in target")
	if (*object1)["nil"] == nil {
		t.Fatal("Value of \"nil\" in target is nil")
	}
}

func TestMergeWithStrings(t *testing.T) {
	object1 := ptr.String("object1")
	object2 := ptr.String("object2")

	Merge(&object1, object2)
	assert.Equal(t, *object1, "object2", "Strings not merged")
	assert.Equal(t, *object2, "object2", "Source string altered")
	if object1 == object2 {
		t.Fatal("Inputs are now equal after merge.")
	}
}
