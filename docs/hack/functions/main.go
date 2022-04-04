package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/loft-sh/devspace/docs/hack/util"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/pipelinehandler/commands"
)

type Function struct {
	Handler interface{}
	Flags   interface{}
}

func main() {
	for functionName := range Functions {
		function := Functions[functionName]

		pageFile := fmt.Sprintf(util.PagePathFunction, functionName)
		pageContent := []byte{}

		_, err := os.Stat(pageFile)
		if err == nil {
			pageContent, err = ioutil.ReadFile(pageFile)
			if err != nil {
				log.Fatal(err)
			}
		}

		functionDescription := util.GetExistingDescription(string(pageContent))

		partialImports := &[]string{}

		argsContent := fmt.Sprintf("The function `%s` does not expect any arguments.", functionName)

		funcHandlerRef := reflect.ValueOf(function.Handler).Type()
		maxArguments := funcHandlerRef.NumIn()
		if maxArguments > 0 {
			lastArgument := funcHandlerRef.In(maxArguments - 1)
			if lastArgument.String() == "[]string" {
				argsContent = fmt.Sprintf("The function `%s` expects arguments.", functionName)

				existingArgsContent := util.GetSection("Arguments", string(pageContent))
				if existingArgsContent != "" {
					argsContent = existingArgsContent
				}
			}
		}

		flagRef := reflect.ValueOf(function.Flags).Type()
		flagContent := getFlagReference(functionName, flagRef, partialImports, string(pageContent))
		if flagContent == "" {
			continue
		}

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

		content := fmt.Sprintf(util.TemplatePage, functionName, functionName, importContent, functionDescription, argsContent, flagContent)

		err = ioutil.WriteFile(pageFile, []byte(content), 0)
		if err != nil {
			panic(err)
		}
	}
}

func getFlagReference(functionName string, flagRef reflect.Type, partialImports *[]string, pageContent string) string {
	content := ""

	for i := 0; i < flagRef.NumField(); i++ {
		field := flagRef.Field(i)
		if field.Anonymous {
			content = content + getFlagReference(functionName, field.Type, partialImports, pageContent)
			continue
		}

		long := field.Tag.Get("long")
		if long == "" {
			continue
		}

		existingFlagContent := util.GetPartOfAutogenSection("`--"+long, pageContent)

		short := flagRef.Field(i).Tag.Get("short")
		description := flagRef.Field(i).Tag.Get("description")

		if short != "" {
			short = " / -" + short
		}

		flagPartial := fmt.Sprintf(util.PartialPathFlag, functionName, long)
		_, err := os.Stat(flagPartial)
		if err == nil {
			*partialImports = append(*partialImports, flagPartial)
			description = description + "\n\n" + fmt.Sprintf(util.TemplatePartialUseFlag, util.GetPartialImportName(flagPartial), functionName, long)
		}

		content = content + fmt.Sprintf(util.TemplateFlag, long, short, description) + existingFlagContent
	}

	return content
}

var Functions = map[string]Function{
	"build_images": {
		Handler: commands.BuildImages,
		Flags:   commands.BuildImagesOptions{},
	},
	"create_deployments": {
		Handler: commands.CreateDeployments,
		Flags:   commands.CreateDeploymentsOptions{},
	},
}
