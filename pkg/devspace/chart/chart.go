package chart

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// ImageNamePlaceholder is the value to replace
const ImageNamePlaceholder = "image: yourusername/devspace"

// PortPlaceholder is the value to replace
const PortPlaceholder = "containerPort: 32127"

// ReplaceImage will try to replace the default image name in the values.yaml with the correct image name
func ReplaceImage(valuesPath string, imageName string) error {
	data, err := ioutil.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("Couldn't read %s: %v", valuesPath, err)
	}

	newContent := string(data)
	newContent = strings.Replace(newContent, ImageNamePlaceholder, "image: "+imageName, -1)
	if newContent == string(data) {
		return nil
	}

	return ioutil.WriteFile(valuesPath, []byte(newContent), 0644)
}

// ReplacePort will try to replace the default port in the values.yaml with the correct port entered by the user
func ReplacePort(valuesPath string, portValue string) error {
	data, err := ioutil.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("Couldn't read %s: %v", valuesPath, err)
	}

	newContent := string(data)
	newContent = strings.Replace(newContent, PortPlaceholder, "containerPort: "+portValue, -1)
	if newContent == string(data) {
		return nil
	}

	return ioutil.WriteFile(valuesPath, []byte(newContent), 0644)
}
