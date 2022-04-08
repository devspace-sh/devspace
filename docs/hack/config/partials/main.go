package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gertd/go-pluralize"
	"github.com/invopop/jsonschema"
	"github.com/loft-sh/devspace/docs/hack/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

const configPartialBasePath = "docs/pages/configuration/_partials/"
const nameFieldName = "name"
const versionFieldName = "version"
const groupKey = "group"
const groupNameKey = "group_name"

var pluralizeClient = pluralize.NewClient()

func main() {
	r := new(jsonschema.Reflector)
	r.AllowAdditionalProperties = true
	r.PreferYAMLSchema = true
	r.RequiredFromJSONSchemaTags = true
	r.YAMLEmbeddedStructs = false
	r.ExpandedStruct = true

	err := r.AddGoComments("github.com/loft-sh/devspace", "./pkg/devspace/config/versions/latest")
	if err != nil {
		panic(err)
	}

	schema := r.Reflect(&latest.Config{})

	versionField, ok := schema.Properties.Get(versionFieldName)
	if ok {
		if fieldSchema, ok := versionField.(*jsonschema.Schema); ok {
			versionEnum := []string{}
			for version := range versions.VersionLoader {
				versionEnum = append(versionEnum, version)
			}

			sort.SliceStable(versionEnum, func(a, b int) bool {
				majorA, _ := strconv.Atoi(string(versionEnum[a][1]))
				majorB, _ := strconv.Atoi(string(versionEnum[b][1]))
				minorA, _ := strconv.Atoi(string(versionEnum[a][6:]))
				minorB, _ := strconv.Atoi(string(versionEnum[b][6:]))

				if majorA == majorB {
					return minorA > minorB
				} else {
					return majorA > majorB
				}
			})

			fieldSchema.Enum = []interface{}{}
			for _, version := range versionEnum {
				fieldSchema.Enum = append(fieldSchema.Enum, version)
			}
		}
	}

	createSections("", schema, schema.Definitions, 1, false)
}

type Group struct {
	File    string
	Name    string
	Content string
	Imports *[]string
}

func createSections(prefix string, schema *jsonschema.Schema, definitions jsonschema.Definitions, depth int, parentIsNameObjectMap bool) string {
	partialImports := &[]string{}
	content := ""
	headlinePrefix := strings.Repeat("#", depth+1) + " "

	groups := map[string]*Group{}

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
				groupID, _ := fieldSchema.Extras[groupKey].(string)

				var patternPropertySchema *jsonschema.Schema
				var nestedSchema *jsonschema.Schema

				ref := ""
				if fieldSchema.Type == "array" {
					ref = fieldSchema.Items.Ref
					fieldType = "object[]"
				} else if patternPropertySchema, ok = fieldSchema.PatternProperties[".*"]; ok {
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
					fieldDefault := ""

					fieldType = fieldSchema.Type
					if fieldType == "" && fieldSchema.OneOf != nil {
						for _, oneOfType := range fieldSchema.OneOf {
							if fieldType != "" {
								fieldType = fieldType + "|"
							}
							fieldType = fieldType + oneOfType.Type
						}
					}

					if isNameObjectMap {
						fieldNameSingular := pluralizeClient.Singular(fieldName)
						fieldType = "&lt;" + fieldNameSingular + "_name&gt;:"

						if patternPropertySchema != nil && patternPropertySchema.Type != "" {
							fieldType = fieldType + patternPropertySchema.Type
						} else {
							fieldType = fieldType + "object"
						}
					}

					if fieldType == "array" {
						fieldType = fieldSchema.Items.Type + "[]"
					}

					if fieldType == "boolean" {
						fieldDefault = "false"
						if required {
							fieldDefault = "true"
							required = false
						}
					} else {
						fieldDefault, ok = fieldSchema.Default.(string)
						if !ok {
							fieldDefault = ""
						}
					}

					enumValues := GetEumValues(fieldSchema, required, &fieldDefault)

					fieldContent = fmt.Sprintf(util.TemplateConfigField, false, " open", headlinePrefix, fieldName, required, fieldType, fieldDefault, enumValues, fieldSchema.Description, "")

					err := os.MkdirAll(filepath.Dir(fieldFile), os.ModePerm)
					if err != nil {
						panic(err)
					}

					err = ioutil.WriteFile(fieldFile, []byte(fieldContent), os.ModePerm)
					if err != nil {
						panic(err)
					}
				}

				fieldPartial := fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(fieldFile))
				if ref != "" {
					if isNameObjectMap && nestedSchema != nil {
						nameField, ok := nestedSchema.Properties.Get(nameFieldName)
						if ok {
							if nameFieldSchema, ok := nameField.(*jsonschema.Schema); ok {
								fieldNameSingular := pluralizeClient.Singular(fieldName)
								nameFieldRequired := true
								nameFieldDefault := ""
								nameFieldEnumValues := GetEumValues(nameFieldSchema, nameFieldRequired, &nameFieldDefault)

								fieldPartial = fmt.Sprintf(util.TemplateConfigField, true, "open", headlinePrefix, "<"+fieldNameSingular+"_"+nameFieldName+">", nameFieldRequired, "string", nameFieldDefault, nameFieldEnumValues, nameFieldSchema.Description, fieldPartial)
								fieldType = "&lt;" + fieldNameSingular + "_name&gt;:object"
							}
						}
					}

					fieldPartial = fmt.Sprintf(util.TemplateConfigField, true, "", headlinePrefix, fieldName, false, fieldType, "", "", fieldSchema.Description, fieldPartial)
				}

				if groupID != "" {
					group, ok := groups[groupID]
					if !ok {
						group = &Group{
							File:    fmt.Sprintf(configPartialBasePath+"%s/%sgroup_%s.mdx", latest.Version, prefix, groupID),
							Imports: &[]string{},
						}
						groups[groupID] = group

						groupPartial := fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(group.File))

						content = content + "\n\n" + groupPartial
						*partialImports = append(*partialImports, group.File)
					}

					if groupName, ok := fieldSchema.Extras[groupNameKey]; ok {
						group.Name = groupName.(string)
					}

					group.Content = group.Content + fieldPartial
					*group.Imports = append(*group.Imports, fieldFile)
				} else {
					content = content + "\n\n" + fieldPartial
					*partialImports = append(*partialImports, fieldFile)
				}
			}
		}
	}

	for groupID, group := range groups {
		groupContent := group.Content

		if group.Name != "" {
			groupContent = "\n" + `<div className="group-name">` + group.Name + `</div>` + "\n\n" + groupContent
		}

		groupImportContent := ""
		for _, partialFile := range *group.Imports {
			groupImportContent = groupImportContent + GetPartialImport(partialFile, group.File)
		}

		if groupImportContent != "" {
			groupImportContent = groupImportContent + "\n\n"
		}

		groupFileContent := fmt.Sprintf(`%s<div className="group" data-group="%s">%s`+"\n"+`</div>`, groupImportContent, groupID, groupContent)

		err := os.MkdirAll(filepath.Dir(group.File), os.ModePerm)
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(group.File, []byte(groupFileContent), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	if prefix == "" {
		prefix = "reference"
	}

	pageFile := fmt.Sprintf(configPartialBasePath+"%s/%s.mdx", latest.Version, strings.TrimSuffix(prefix, "/"))

	importContent := ""
	for _, partialFile := range *partialImports {
		importContent = importContent + GetPartialImport(partialFile, pageFile)
	}

	content = fmt.Sprintf("%s%s", importContent, content)

	err := ioutil.WriteFile(pageFile, []byte(content), os.ModePerm)
	if err != nil {
		panic(err)
	}

	//fmt.Println(content)

	return content
}

func GetPartialImport(partialFile, importingFile string) string {
	partialImportPath, err := filepath.Rel(filepath.Dir(importingFile), partialFile)
	if err != nil {
		panic(err)
	}

	if partialImportPath[0:1] != "." {
		partialImportPath = "./" + partialImportPath
	}

	return fmt.Sprintf(util.TemplatePartialImport, util.GetPartialImportName(partialFile), partialImportPath)
}

func GetEumValues(fieldSchema *jsonschema.Schema, required bool, fieldDefault *string) string {
	enumValues := ""
	if fieldSchema.Enum != nil {
		for i, enumVal := range fieldSchema.Enum {
			enumValString, ok := enumVal.(string)
			if ok {
				if i == 0 && !required && *fieldDefault == "" {
					*fieldDefault = enumValString
				}

				if enumValues != "" {
					enumValues = enumValues + "<br/>"
				}
				enumValues = enumValues + enumValString
			}
		}
		enumValues = fmt.Sprintf("<span>%s</span>", enumValues)
	}
	return enumValues
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
