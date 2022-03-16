package yamlutil

import (
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestWriteRead(t *testing.T) {
	dir := t.TempDir()

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
	}()

	inputObj := make(map[string]interface{})
	arr0 := make(map[string]interface{})
	arr0["testString"] = "hello"
	arr0["testTrue"] = true
	arr0["testFalse"] = false
	arr0["testBoolString"] = "false"
	arr0["testEmptyString"] = ""
	arr0["testInt"] = 1
	inputObj["someArr"] = []interface{}{arr0, true, 1, "somestring"}
	inputObj["testString"] = "hello"
	inputObj["testTrue"] = true
	inputObj["testFalse"] = false
	inputObj["testBoolString"] = "false"
	inputObj["testEmptyString"] = ""
	inputObj["testInt"] = 1

	err = WriteYamlToFile(inputObj, "yaml.yaml")
	if err != nil {
		t.Fatalf("Error writing yaml: %v", err)
	}

	outputObj := make(map[string]interface{})
	err = ReadYamlFromFile("yaml.yaml", &outputObj)
	if err != nil {
		t.Fatalf("Error reading yaml: %v", err)
	}
	assert.Equal(t, inputObj["testString"], outputObj["testString"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["testTrue"], outputObj["testTrue"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["testFalse"], outputObj["testFalse"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["testBoolString"], outputObj["testBoolString"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["testEmptyString"], outputObj["testEmptyString"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["testInt"], outputObj["testInt"], "Readed yaml doesn't match written yaml")

	assert.Equal(t, inputObj["someArr"].([]interface{})[0].(map[string]interface{})["testString"], outputObj["someArr"].([]interface{})[0].(map[string]interface{})["testString"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[0].(map[string]interface{})["testTrue"], outputObj["someArr"].([]interface{})[0].(map[string]interface{})["testTrue"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[0].(map[string]interface{})["testFalse"], outputObj["someArr"].([]interface{})[0].(map[string]interface{})["testFalse"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[0].(map[string]interface{})["testBoolString"], outputObj["someArr"].([]interface{})[0].(map[string]interface{})["testBoolString"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[0].(map[string]interface{})["testEmptyString"], outputObj["someArr"].([]interface{})[0].(map[string]interface{})["testEmptyString"], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[0].(map[string]interface{})["testInt"], outputObj["someArr"].([]interface{})[0].(map[string]interface{})["testInt"], "Readed yaml doesn't match written yaml")

	assert.Equal(t, inputObj["someArr"].([]interface{})[1], outputObj["someArr"].([]interface{})[1], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[2], outputObj["someArr"].([]interface{})[2], "Readed yaml doesn't match written yaml")
	assert.Equal(t, inputObj["someArr"].([]interface{})[3], outputObj["someArr"].([]interface{})[3], "Readed yaml doesn't match written yaml")
}
