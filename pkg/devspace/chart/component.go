package chart

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	yaml "gopkg.in/yaml.v2"
)

// ListAvailableComponents lists all available devspace components
func ListAvailableComponents() ([]*generator.ComponentSchema, error) {
	// Create component generator
	componentGenerator, err := generator.NewComponentGenerator("")
	if err != nil {
		return nil, fmt.Errorf("Error initializing component generator: %v", err)
	}

	return componentGenerator.ListComponents()
}

// AddComponent adds a component with the given name to the chart
func AddComponent(chartPath string, name string) error {
	// Check if devspace chart
	_, err := os.Stat(filepath.Join(chartPath, "devspace.yaml"))
	if os.IsNotExist(err) {
		return errors.New("Chart is not a devspace chart. `devspace add component` only works with the devspace chart")
	}

	// Create component generator
	componentGenerator, err := generator.NewComponentGenerator(chartPath)
	if err != nil {
		return fmt.Errorf("Error initializing component generator: %v", err)
	}

	// Get values.yaml
	valuesYaml := filepath.Join(chartPath, "values.yaml")
	content, err := ioutil.ReadFile(valuesYaml)
	if err != nil {
		return err
	}

	// Split into content lines, we don't parse the values.yaml here
	// because all comments would be lost on write so we do a little
	// workaround by inserting the components directly after the
	// components and volumes identifier
	contentLines := strings.Split(string(content), "\n")

	// Get component template
	template, err := componentGenerator.GetComponentTemplate(name)
	if err != nil {
		return fmt.Errorf("Error retrieving template: %v", err)
	}

	// Fill components
	if len(template.Components) > 0 {
		out, err := yaml.Marshal(template.Components)
		if err != nil {
			return err
		}

		componentsString := "components:\n" + strings.TrimSpace(string(out))
		contentLines = insertAfterLine(componentsString, "components:", contentLines)
	}

	// Fill volumes
	if len(template.Volumes) > 0 {
		out, err := yaml.Marshal(template.Volumes)
		if err != nil {
			return err
		}

		volumesString := "volumes:\n" + strings.TrimSpace(string(out))
		contentLines = insertAfterLine(volumesString, "volumes:", contentLines)
	}

	err = ioutil.WriteFile(valuesYaml, []byte(strings.Join(contentLines, "\n")), 0666)
	if err != nil {
		return err
	}

	return nil
}

func insertAfterLine(insertString, prefix string, contentLines []string) []string {
	index := -1
	for idx, line := range contentLines {
		if strings.HasPrefix(line, prefix) {
			index = idx
			break
		}
	}

	// Check if found
	if index > -1 {
		// Set components: [] to components: etc.
		contentLines[index] = insertString
	} else {
		contentLines = append([]string{insertString}, contentLines...)
	}

	return contentLines
}
