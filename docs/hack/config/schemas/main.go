package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/expression"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	varspkg "github.com/loft-sh/devspace/pkg/util/vars"
)

const jsonschemaFile = "devspace-schema.json"
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

	openapiSchema := r.Reflect(&latest.Config{})
	genSchema(openapiSchema, openapiSchemaFile)

	jsonSchema := r.Reflect(&latest.Config{})
	genSchema(jsonSchema, jsonschemaFile, Expressions, Vars, CleanUp)
}

func genSchema(schema *jsonschema.Schema, schemaFile string, visitors ...func(s *jsonschema.Schema)) {
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

	// Apply visitors
	if len(visitors) > 0 {
		for _, visitor := range visitors {
			walk(schema, visitor)
		}
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

	err = os.WriteFile(schemaFile, []byte(schemaString), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func modifyAnyOf(field interface{}) []*jsonschema.Schema {
	return []*jsonschema.Schema{
		{
			Type: "object",
			PatternProperties: map[string]*jsonschema.Schema{
				".*": {
					Type: "string",
				},
			},
		},
		{
			Type:              "object",
			PatternProperties: field.(*jsonschema.Schema).PatternProperties,
		},
		{
			Type: "object",
		},
	}
}

func walk(schema *jsonschema.Schema, visit func(s *jsonschema.Schema)) {
	for _, name := range schema.Properties.Keys() {
		property, ok := schema.Properties.Get(name)
		if ok {
			visit(property.(*jsonschema.Schema))
		}
	}

	for _, definition := range schema.Definitions {
		for _, name := range definition.Properties.Keys() {
			property, ok := definition.Properties.Get(name)
			if ok {
				visit(property.(*jsonschema.Schema))
			}
		}
	}
}

func Expressions(s *jsonschema.Schema) {
	if s.Type == "" && s.Ref == "" {
		return
	}

	if s.Type == "string" {
		return
	}

	if len(s.AnyOf) > 0 {
		s.AnyOf = append(s.AnyOf, &jsonschema.Schema{
			Type:    "string",
			Pattern: expression.ExpressionMatchRegex.String(),
		})
	} else {
		if len(s.OneOf) == 0 {
			// Preserve original type
			if s.Ref != "" {
				s.OneOf = append(s.OneOf, &jsonschema.Schema{
					Ref: s.Ref,
				})
			} else {
				s.OneOf = append(s.OneOf, &jsonschema.Schema{
					Type:              s.Type,
					Items:             s.Items,
					PatternProperties: s.PatternProperties,
				})
			}
		}
		s.OneOf = append(s.OneOf, &jsonschema.Schema{
			Type:    "string",
			Pattern: expression.ExpressionMatchRegex.String(),
		})
	}
}

func Vars(s *jsonschema.Schema) {
	if s.Type == "" {
		return
	}

	if s.Type == "string" {
		return
	}

	if s.Type == "object" {
		return
	}

	if s.Type == "array" {
		return
	}

	if len(s.AnyOf) > 0 {
		s.AnyOf = append(s.AnyOf, &jsonschema.Schema{
			Type:    "string",
			Pattern: varspkg.VarMatchRegex.String(),
		})
	} else {
		if len(s.OneOf) == 0 {
			// Preserve original type
			s.OneOf = append(s.OneOf, &jsonschema.Schema{
				Type:              s.Type,
				Items:             s.Items,
				PatternProperties: s.PatternProperties,
			})
		}
		s.OneOf = append(s.OneOf, &jsonschema.Schema{
			Type:    "string",
			Pattern: varspkg.VarMatchRegex.String(),
		})
	}
}

func CleanUp(s *jsonschema.Schema) {
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		s.Ref = ""
		s.Type = ""
		s.Items = nil
		s.PatternProperties = nil
	}
}
