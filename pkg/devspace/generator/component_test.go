package generator

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	fakelog "github.com/devspace-cloud/devspace/pkg/util/log/testing"

	"gotest.tools/assert"
)

func TestComponentGenerator(t *testing.T) {
	//Create TmpFolder
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// Cleanup temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	componentGenerator, err := NewComponentGenerator()
	if err != nil {
		t.Fatalf("Error creating componentGenerator: %v", err)
	}

	componentGenerator.gitRepo.LocalPath = "."

	//Test ListComponents
	componentList, err := componentGenerator.ListComponents()
	if err == nil {
		t.Fatal("No Error when listing components without the folder being created.")
	}
	assert.Equal(t, 0, len(componentList), "Components shown before the first component was created")

	//Test ListComponents with one malformed component
	err = fsutil.WriteToFile([]byte(``), "components/malformed")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	componentList, err = componentGenerator.ListComponents()
	if err == nil {
		t.Fatalf("No Error listing components with a malformed component: %v", err)
	}
	assert.Equal(t, 0, len(componentList), "Components shown before the first component was created")
	err = os.Remove("components/malformed")
	if err != nil {
		t.Fatalf("Error deleting file: %v", err)
	}

	err = fsutil.WriteToFile([]byte(`invalidField`), "components/badyaml/component.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	componentList, err = componentGenerator.ListComponents()
	if err == nil {
		t.Fatalf("No Error listing components with a component that has invalid yaml: %v", err)
	}
	assert.Equal(t, 0, len(componentList), "Components shown before the first component was created")
	err = os.RemoveAll("components/badyaml")
	if err != nil {
		t.Fatalf("Error deleting file: %v", err)
	}

	//Test ListComponents with one empty component
	err = fsutil.WriteToFile([]byte(`description: hello world`), "components/mycomponent/component.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	componentList, err = componentGenerator.ListComponents()
	if err != nil {
		t.Fatalf("Error listing components: %v", err)
	}
	assert.Equal(t, 1, len(componentList), "The created component doesn't appear")
	assert.Equal(t, "hello world", componentList[0].Description, "Wrong component returned from ListComponents")

	//Test GetComponentTemplate with not existing component
	_, err = componentGenerator.GetComponentTemplate("doesnotexist", log.GetInstance())
	if err == nil {
		t.Fatalf("No Error getting template of not existing component")
	}

	//Test GetComponentTemplate with not existing template
	_, err = componentGenerator.GetComponentTemplate("mycomponent", log.GetInstance())
	if err == nil {
		t.Fatalf("No Error getting template of not existing template")
	}

	//Test GetComponentTemplate with template that has invalid yaml content
	err = fsutil.WriteToFile([]byte(`wrongYamlField: hello`), "components/mycomponent/template.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	_, err = componentGenerator.GetComponentTemplate("mycomponent", log.GetInstance())
	if err == nil {
		t.Fatalf("No Error getting template of template with invalid yaml")
	}

	//Test GetComponentTemplate with template
	err = fsutil.WriteToFile([]byte(`replicas: 1234`), "components/mycomponent/template.yaml")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	template, err := componentGenerator.GetComponentTemplate("mycomponent", log.GetInstance())
	if err != nil {
		t.Fatalf("Error getting template of template: %v", err)
	}
	assert.Equal(t, *template.Replicas, 1234, "Wrong template returned")
}

func TestVarReplaceFn(t *testing.T) {
	fakeLogger := fakelog.NewFakeLogger()
	comp := ComponentSchema{
		VariableValues: map[string]string{
			"hello":       "world",
			"isThisATest": "true",
			"OnePlusOne":  "2",
		},
		Variables: []latest.Variable{
			latest.Variable{
				Name: "NeedsQuestion",
			},
			latest.Variable{
				Name:              "AlsoNeedsQuestion",
				Question:          "SomeQuestion",
				Default:           "SomeDefault",
				ValidationPattern: "SomeValidationPattern",
				ValidationMessage: "SomeValidationMessage",
			},
		},
	}

	fakeLogger.Survey.SetNextAnswer("DoesNeedQuestion")

	val, _ := comp.varReplaceFn("", "${hello}", fakeLogger)
	assert.Equal(t, "world", val, "Wrong value returned for hello")

	val, _ = comp.varReplaceFn("", "${isThisATest}", fakeLogger)
	assert.Equal(t, true, val, "Wrong value returned for isThisATest")
	val, _ = comp.varReplaceFn("", "${OnePlusOne}", fakeLogger)
	assert.Equal(t, 2, val, "Wrong value returned for OnePlusOne")
	val, _ = comp.varReplaceFn("", "${NeedsQuestion}", fakeLogger)
	assert.Equal(t, "DoesNeedQuestion", val, "Wrong value returned for NeedsQuestion")

	fakeLogger.Survey.SetNextAnswer("DoesNeedQuestionAsWell")
	val, _ = comp.varReplaceFn("", "${AlsoNeedsQuestion}", fakeLogger)
	assert.Equal(t, "DoesNeedQuestionAsWell", val, "Wrong value returned for AlsoNeedsQuestion")

	val, _ = comp.varReplaceFn("", "${Doesn'tMatchRegex", fakeLogger)
	assert.Equal(t, "", val, "Wrong value returned for not matching input")
}
