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

const jsonschemaFile = "docs/schemas/config-jsonschema.json"
const openapiSchemaFile = "docs/schemas/config-openapi.json"

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
