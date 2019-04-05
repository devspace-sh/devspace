package chart

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
)

// ListAvailableComponents lists all available devspace components
func ListAvailableComponents() ([]*generator.ComponentSchema, error) {
	// Create component generator
	componentGenerator, err := generator.NewComponentGenerator()
	if err != nil {
		return nil, fmt.Errorf("Error initializing component generator: %v", err)
	}

	return componentGenerator.ListComponents()
}
