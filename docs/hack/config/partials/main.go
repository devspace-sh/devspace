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

	createSections("", schema, schema.Definitions, 1)
}

func createSections(prefix string, schema *jsonschema.Schema, definitions jsonschema.Definitions, depth int) string {
	partialImports := &[]string{}
	content := ""

	for _, fieldName := range schema.Properties.Keys() {

		field, ok := schema.Properties.Get(fieldName)
		if ok {
			if fieldSchema, ok := field.(*jsonschema.Schema); ok {
				fieldContent := ""
				fieldFile := fmt.Sprintf(configPartialBasePath+"%s/%s%s.mdx", latest.Version, prefix, fieldName)

				ref := ""
				if fieldSchema.Type == "array" {
					ref = fieldSchema.Items.Ref
				} else if patternPropertySchema, ok := fieldSchema.PatternProperties[".*"]; ok {
					ref = patternPropertySchema.Ref
				} else if fieldSchema.Ref != "" {
					ref = fieldSchema.Ref
				}

				if ref != "" {
					refSplit := strings.Split(ref, "/")
					nestedSchema, ok := definitions[refSplit[len(refSplit)-1]]

					if ok {
						newPrefix := prefix + fieldName + "/"
						createSections(newPrefix, nestedSchema, definitions, depth+1)
					}
				} else {
					required := ""
					if contains(schema.Required, fieldName) {
						required = util.TemplateConfigFieldRequired
					}

					fieldTypeRaw := fieldSchema.Type
					if fieldTypeRaw == "array" {
						fieldTypeRaw = fieldSchema.Items.Type + "[]"
					}
					fieldType := fmt.Sprintf(util.TemplateConfigFieldType, fieldTypeRaw)
					fieldDefault := ""
					if fieldSchema.Default != nil {
						fieldDefault = fmt.Sprintf(util.TemplateConfigFieldType, fieldSchema.Default)
					}

					fieldContent = fmt.Sprintf(util.TemplateConfigField, fieldName, required, fieldType, fieldDefault, fieldSchema.Description)

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
					fieldSummary := fmt.Sprintf("<h%d><code>%s</code></h%d>", depth+1, fieldName, depth+1)
					fieldPartial = fmt.Sprintf("<details><summary>%s</summary>\n%s\n</details>", fieldSummary, fieldPartial)
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
