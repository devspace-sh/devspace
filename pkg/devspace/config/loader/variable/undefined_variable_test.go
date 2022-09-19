package variable

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestGetParams(t *testing.T) {
	testCases := map[string]bool{"testing/with_default_value/devspace.yaml": true, "testing/without_default_value/devspace.yaml": false}
	for input, expected := range testCases {
		config := getConfig(input)
		variable := config.Vars["MYSQL_VERSION"]
		actual := getParams(variable)
		if expected != actual.DefaultValueSet {
			t.Errorf("TestCase %s\nactual:%t\nexpected:%t", input, actual.DefaultValueSet, true)
		}
	}
}

func getConfig(filename string) *latest.Config {
	v := &latest.Config{}
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, v)
	if err != nil {
		fmt.Printf("Unmarshal: %v", err)
	}
	return v
}
