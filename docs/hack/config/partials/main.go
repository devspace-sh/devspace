package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/devspace/docs/hack/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

const configPartialBasePath = "docs/pages/configuration/_partials/"
const nameFieldName = "name"

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

	createSections("", schema, schema.Definitions, 1, false)
}

func createSections(prefix string, schema *jsonschema.Schema, definitions jsonschema.Definitions, depth int, parentIsNameObjectMap bool) string {
	partialImports := &[]string{}
	content := ""
	headlinePrefix := strings.Repeat("#", depth+1) + " "

	for _, fieldName := range schema.Properties.Keys() {
		if parentIsNameObjectMap && fieldName == nameFieldName {
			continue
		}

		field, ok := schema.Properties.Get(fieldName)
		if ok {
			if fieldSchema, ok := field.(*jsonschema.Schema); ok {
				fieldContent := ""
				fieldFile := fmt.Sprintf(configPartialBasePath+"%s/%s%s.mdx", latest.Version, prefix, fieldName)
				fieldType := "object"
				isNameObjectMap := false

				var nestedSchema *jsonschema.Schema

				ref := ""
				if fieldSchema.Type == "array" {
					ref = fieldSchema.Items.Ref
					fieldType = "object[]"
				} else if patternPropertySchema, ok := fieldSchema.PatternProperties[".*"]; ok {
					ref = patternPropertySchema.Ref
					isNameObjectMap = true
				} else if fieldSchema.Ref != "" {
					ref = fieldSchema.Ref
				}

				if ref != "" {
					refSplit := strings.Split(ref, "/")
					nestedSchema, ok = definitions[refSplit[len(refSplit)-1]]

					if ok {
						newPrefix := prefix + fieldName + "/"
						createSections(newPrefix, nestedSchema, definitions, depth+1, isNameObjectMap)
					}
				} else {
					required := contains(schema.Required, fieldName)
					fieldType = fieldSchema.Type
					if fieldType == "array" {
						fieldType = fieldSchema.Items.Type + "[]"
					}

					fieldDefault, ok := fieldSchema.Default.(string)
					if !ok {
						fieldDefault = ""
					}

					fieldContent = fmt.Sprintf(util.TemplateConfigField, false, " open", headlinePrefix, fieldName, required, fieldType, fieldDefault, fieldSchema.Description, "")

					err := os.MkdirAll(filepath.Dir(fieldFile), os.ModePerm)
					if err != nil {
						panic(err)
					}

					err = ioutil.WriteFile(fieldFile, []byte(fieldContent), os.ModePerm)
					if err != nil {
						panic(err)
					}
				}

				*partialImports = append(*partialImports, fieldFile)
				fieldPartial := fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(fieldFile))
				if ref != "" {
					if isNameObjectMap && nestedSchema != nil {
						nameField, ok := nestedSchema.Properties.Get(nameFieldName)
						if ok {
							if nameFieldSchema, ok := nameField.(*jsonschema.Schema); ok {
								fieldPartial = fmt.Sprintf(util.TemplateConfigField, true, "open", headlinePrefix, "<"+nameFieldName+">", false, "object", "", nameFieldSchema.Description, fieldPartial)
							}
						}
					}
					fieldPartial = fmt.Sprintf(util.TemplateConfigField, true, "", headlinePrefix, fieldName, false, fieldType, "", fieldSchema.Description, fieldPartial)
				}

				content = content + "\n\n" + fieldPartial
			}
		}
	}

	if prefix == "" {
		prefix = "reference"
	}

	pageFile := fmt.Sprintf(configPartialBasePath+"%s/%s.mdx", latest.Version, strings.TrimSuffix(prefix, "/"))

	importContent := ""
	for _, partialImport := range *partialImports {
		partialImportPath, err := filepath.Rel(filepath.Dir(pageFile), partialImport)
		if err != nil {
			panic(err)
		}

		if partialImportPath[0:1] != "." {
			partialImportPath = "./" + partialImportPath
		}

		importContent = importContent + fmt.Sprintf(util.TemplatePartialImport, util.GetPartialImportName(partialImport), partialImportPath)
	}

	content = fmt.Sprintf("%s%s", importContent, content)

	err := ioutil.WriteFile(pageFile, []byte(content), os.ModePerm)
	if err != nil {
		panic(err)
	}

	//fmt.Println(content)

	return content
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
