package chart

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/generator"
	"github.com/pkg/errors"
)

// ListAvailableComponents lists all available devspace components
func ListAvailableComponents() ([]*generator.ComponentSchema, error) {
	// Create component generator
	componentGenerator, err := generator.NewComponentGenerator()
	if err != nil {
		return nil, errors.Errorf("Error initializing component generator: %v", err)
	}

	return componentGenerator.ListComponents()
}
