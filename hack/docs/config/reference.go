package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

// Run executes the command logic
func main() {
	mustGenerateOpenAPISpec := len(os.Args) > 1 && os.Args[1] == "true"
	prefix := ""
	if mustGenerateOpenAPISpec {
		prefix = "      "
	}

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

	schemaJSON, err := json.MarshalIndent(schema, prefix, "  ")
	if err != nil {
		panic(err)
	}

	schemaString := strings.ReplaceAll(string(schemaJSON), "#/$defs/", "#/definitions/Config/$defs/")

	if mustGenerateOpenAPISpec {
		fmt.Printf(`{
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
	} else {
		fmt.Printf(`%s`, schemaString)
	}
}
