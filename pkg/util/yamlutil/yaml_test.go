package yamlutil

import (
	"os"
	"testing"
)

func TestWriteYamlToFile(t *testing.T) {

	data := map[interface{}]interface{}{}

	data["TopLevelString"] = "Level 1 String"
	data["TopLevelInt"] = 5
	data["TopLevelArray"] = [1]string{
		"Index1",
	}

	childObject := map[interface{}]interface{}{}
	childObject["ChildString"] = "Level 2 String"
	childObject["ChildInt"] = 2

	data["ChildObj"] = childObject

	objectArray := [1]map[interface{}]interface{}{}
	objectArray[0]["IndexString"] = "Object Index 0"
	objectArray[0]["IndexInt"] = 0

	fileName := os.TempDir() + "/test.yaml"

	WriteYamlToFile(data, fileName)

}

type ChildObject struct {
	ChildString string
	ChildInt    int
}
