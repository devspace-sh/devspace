package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

const jsonschemaFile = "devspace-schema.json"
const openapiSchemaFile = "devspace-openapi.json"

// Run executes the command logic
func main() {
	r := new(jsonschema.Reflector)
	r.AllowAdditionalProperties = true
	r.PreferYAMLSchema = true
	r.RequiredFromJSONSchemaTags = false
	r.YAMLEmbeddedStructs = false
	r.ExpandedStruct = true

	err := r.AddGoComments("github.com/loft-sh/devspace", "./pkg/devspace/config/versions/latest")
	if err != nil {
		panic(err)
	}

	schema := r.Reflect(&latest.Config{})

	genSchema(schema, jsonschemaFile)
	genSchema(schema, openapiSchemaFile)
}

func genSchema(schema *jsonschema.Schema, schemaFile string) {
	isOpenAPISpec := schemaFile == openapiSchemaFile
	prefix := ""
	if isOpenAPISpec {
		prefix = "      "
	}

	// vars
	vars, ok := schema.Properties.Get("vars")
	if ok {
		vars.(*jsonschema.Schema).AnyOf = modifyAnyOf(vars)
		vars.(*jsonschema.Schema).PatternProperties = nil
	}

	// pipelines
	pipelines, ok := schema.Properties.Get("pipelines")
	if ok {
		pipelines.(*jsonschema.Schema).AnyOf = modifyAnyOf(pipelines)
		pipelines.(*jsonschema.Schema).PatternProperties = nil
	}

	// commands
	commands, ok := schema.Properties.Get("commands")
	if ok {
		commands.(*jsonschema.Schema).AnyOf = modifyAnyOf(commands)
		commands.(*jsonschema.Schema).PatternProperties = nil
	}

	// images
	images, ok := schema.Properties.Get("images")
	if ok {
		images.(*jsonschema.Schema).AnyOf = modifyAnyOf(images)
		images.(*jsonschema.Schema).PatternProperties = nil
	}

	//deployments
	deployments, ok := schema.Properties.Get("deployments")
	if ok {
		deployments.(*jsonschema.Schema).AnyOf = modifyAnyOf(deployments)
		deployments.(*jsonschema.Schema).PatternProperties = nil
	}

	//dependencies
	dependencies, ok := schema.Properties.Get("dependencies")
	if ok {
		dependencies.(*jsonschema.Schema).AnyOf = modifyAnyOf(dependencies)
		dependencies.(*jsonschema.Schema).PatternProperties = nil
	}
	//pullSecrets
	pullSecrets, ok := schema.Properties.Get("pullSecrets")
	if ok {
		pullSecrets.(*jsonschema.Schema).AnyOf = modifyAnyOf(pullSecrets)
		pullSecrets.(*jsonschema.Schema).PatternProperties = nil
	}

	schemaJSON, err := json.MarshalIndent(schema, prefix, "  ")
	if err != nil {
		panic(err)
	}

	schemaString := string(schemaJSON)

	if isOpenAPISpec {
		schemaString = strings.ReplaceAll(schemaString, "#/$defs/", "#/definitions/Config/$defs/")

		schemaString = fmt.Sprintf(`{
	"swagger": "2.0",
	"info": {
		"version": "%s",
		"title": "devspace.yaml"
	},
	"paths": {},
	"definitions": {
		"Config": %s
	}
}
`, latest.Version, schemaString)
	}

	err = os.MkdirAll(filepath.Dir(schemaFile), os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(schemaFile, []byte(schemaString), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func modifyAnyOf(field interface{}) []*jsonschema.Schema {
	return []*jsonschema.Schema{
		{
			Type: "object",
		},
		{
			Type:              "object",
			PatternProperties: field.(*jsonschema.Schema).PatternProperties,
		},
	}
}
