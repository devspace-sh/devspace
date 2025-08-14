package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/loft-sh/devspace/docs/hack/util"
	basiccommands "github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/basichandler/commands"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/engine/pipelinehandler/commands"
)

const functionPartialBasePath = "docs/pages/configuration/_partials/functions/"

type Function struct {
	Name        string
	Handler     interface{}
	Flags       interface{}
	Description string
	Args        string
	ArgEnum     []string
	Return      string
	Group       string
	IsGlobal    bool
}

func main() {
	functionRefContent := "\n\n"
	globalFunctionRefContent := "\n\n"
	pipelineFunctionRefContent := "\n\n"
	functionRefFile := functionPartialBasePath + "reference.mdx"
	globalFunctionRefFile := functionPartialBasePath + "reference_global.mdx"
	pipelineFunctionRefFile := functionPartialBasePath + "reference_pipeline.mdx"

	groups := map[string]*util.Group{}

	for i := range Functions {
		function := Functions[i]

		functionFile := fmt.Sprintf(functionPartialBasePath+"%s.mdx", function.Name)
		pageContent := []byte{}

		partialImports := &[]string{}
		flagContent := ""

		if function.Flags != nil {
			flagRef := reflect.ValueOf(function.Flags).Type()
			flagContent = getFlagReference(function.Name, functionFile, flagRef, partialImports, string(pageContent))
		}

		importContent := ""
		for _, partialImport := range *partialImports {
			importContent = importContent + util.GetPartialImport(partialImport, functionFile)
		}

		err := os.MkdirAll(filepath.Dir(functionFile), os.ModePerm)
		if err != nil {
			panic(err)
		}

		argEnum := ""
		if function.ArgEnum != nil {
			argEnum = `<span>` + strings.Join(function.ArgEnum, " ") + `</span>`
		}

		anchorName := function.Name
		functionContent := importContent + "\n" + fmt.Sprintf(util.TemplateFunctionRef, flagContent != "", "", "### ", function.Name, function.Args, argEnum, function.Return, !function.IsGlobal, anchorName, function.Description, flagContent)

		err = os.WriteFile(functionFile, []byte(functionContent), os.ModePerm)
		if err != nil {
			panic(err)
		}

		partialImport := util.GetPartialImport(functionFile, functionRefFile)
		partialUse := fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(function.Name))
		partialImportGlobal := partialImport
		partialUseGlobal := partialUse
		partialImportPipeline := partialImport
		partialUsePipeline := partialUse
		writeRef := true
		writeGlobalRef := false
		writePipelineRef := false

		if function.Group != "" {
			groupID := strings.ToLower(function.Group)

			globalGroupID := fmt.Sprintf("%s_global", groupID)
			globalGroup, globalGroupExists := groups[globalGroupID]

			pipelineGroupID := fmt.Sprintf("%s_pipeline", groupID)
			pipelineGroup, pipelineGroupExists := groups[pipelineGroupID]

			group, groupExists := groups[groupID]
			if !groupExists {
				group = &util.Group{
					Name:    function.Group,
					File:    fmt.Sprintf(functionPartialBasePath+"group_%s.mdx", groupID),
					Imports: &[]string{},
					Content: "\n\n",
				}
				groups[groupID] = group
			}

			writeRef = !groupExists
			partialImportGlobal = ""
			partialUseGlobal = ""
			partialImportPipeline = ""
			partialUsePipeline = ""

			if !globalGroupExists {
				globalGroup = &util.Group{
					Name:    function.Group,
					File:    fmt.Sprintf(functionPartialBasePath+"group_%s.mdx", globalGroupID),
					Imports: &[]string{},
					Content: "\n\n",
				}
				groups[globalGroupID] = globalGroup
				writeGlobalRef = true
				partialImportGlobal = util.GetPartialImport(globalGroup.File, globalFunctionRefFile)
				partialUseGlobal = fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(globalGroup.File))
			}

			if !pipelineGroupExists {
				pipelineGroup = &util.Group{
					Name:    function.Group,
					File:    fmt.Sprintf(functionPartialBasePath+"group_%s.mdx", pipelineGroupID),
					Imports: &[]string{},
					Content: "\n\n",
				}
				groups[pipelineGroupID] = pipelineGroup
				writePipelineRef = true
				partialImportPipeline = util.GetPartialImport(pipelineGroup.File, pipelineFunctionRefFile)
				partialUsePipeline = fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(pipelineGroup.File))
			}

			group.Content = partialImport + group.Content + partialUse

			if function.IsGlobal {
				globalGroup.Content = partialImport + globalGroup.Content + partialUse
			} else {
				pipelineGroup.Content = partialImport + pipelineGroup.Content + partialUse
			}

			partialImport = util.GetPartialImport(group.File, functionRefFile)
			partialUse = fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(group.File))
		}

		if writeRef {
			functionRefContent = partialImport + functionRefContent + partialUse
		}

		if writeGlobalRef {
			globalFunctionRefContent = partialImportGlobal + globalFunctionRefContent + partialUseGlobal
		}

		if writePipelineRef {
			pipelineFunctionRefContent = partialImportPipeline + pipelineFunctionRefContent + partialUsePipeline
		}
	}

	util.ProcessGroups(groups)

	err := os.WriteFile(functionRefFile, []byte(functionRefContent), os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(globalFunctionRefFile, []byte(globalFunctionRefContent), os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(pipelineFunctionRefFile, []byte(pipelineFunctionRefContent), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func getFlagReference(functionName, functionFile string, flagRef reflect.Type, partialImports *[]string, pageContent string) string {
	flagPartialContent := ""

	for i := 0; i < flagRef.NumField(); i++ {
		flag := flagRef.Field(i)
		if flag.Anonymous {
			flagPartialContent = flagPartialContent + getFlagReference(functionName, functionFile, flag.Type, partialImports, pageContent)
			continue
		}

		long := flag.Tag.Get("long")
		if long == "" {
			continue
		}

		short := flag.Tag.Get("short")
		description := flag.Tag.Get("description")

		if short != "" {
			short = " / -" + short
		}

		anchorName := functionName + "-" + long
		flagContent := fmt.Sprintf(util.TemplateFunctionRef, false, "", "#### ", "--"+long+short, flag.Type.String(), "", "", false, anchorName, description, "")

		flagFile := fmt.Sprintf(functionPartialBasePath+"%s/%s.mdx", functionName, long)
		err := os.MkdirAll(filepath.Dir(flagFile), os.ModePerm)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(flagFile, []byte(flagContent), os.ModePerm)
		if err != nil {
			panic(err)
		}

		flagPartialContent = flagPartialContent + fmt.Sprintf(util.TemplatePartialUse, util.GetPartialImportName(flagFile))
		*partialImports = append(*partialImports, flagFile)
	}

	return flagPartialContent
}

const groupImages = "Images"
const groupDeployments = "Deployments"
const groupDev = "Dev"
const groupPipelines = "Pipelines"
const groupChecks = "Checks"
const groupOther = "Other"

var Functions = []Function{
	{
		Name:        "build_images",
		Description: `Builds all images passed as arguments in parallel`,
		Args:        `[image-1] [image-2] ...`,
		Handler:     commands.BuildImages,
		Flags:       commands.BuildImagesOptions{},
		Group:       groupImages,
	},
	{
		Name:        "ensure_pull_secrets",
		Description: `Creates pull secrets for all images passed as arguments`,
		Args:        `[image-1] [image-2] ...`,
		Handler:     commands.EnsurePullSecrets,
		Flags:       commands.EnsurePullSecretsOptions{},
		Group:       groupImages,
	},
	{
		Name:        "get_image",
		Description: `Returns the most recently built image and/or tag for a given image name`,
		Args:        `[image]`,
		Handler:     commands.GetImage,
		Flags:       commands.GetImageOptions{},
		Return:      reflect.String.String(),
		Group:       groupImages,
	},
	{
		Name:        "create_deployments",
		Description: `Creates all deployments passed as arguments in parallel`,
		Args:        `[deployment-1] [deployment-2] ...`,
		Handler:     commands.CreateDeployments,
		Flags:       commands.CreateDeploymentsOptions{},
		Group:       groupDeployments,
	},
	{
		Name:        "purge_deployments",
		Description: `Purges all deployments passed as arguments`,
		Args:        `[deployment-1] [deployment-2] ...`,
		Handler:     commands.PurgeDeployments,
		Flags:       commands.PurgeDeploymentsOptions{},
		Group:       groupDeployments,
	},
	{
		Name:        "start_dev",
		Description: `Starts all dev modes passed as arguments`,
		Args:        `[dev-1] [dev-2] ...`,
		Handler:     commands.StartDev,
		Flags:       commands.StartDevOptions{},
		Group:       groupDev,
	},
	{
		Name:        "stop_dev",
		Description: `Stops all dev modes passed as arguments`,
		Args:        `[dev-1] [dev-2] ...`,
		Handler:     commands.StopDev,
		Flags:       commands.StopDevOptions{},
		Group:       groupDev,
	},
	{
		Name:        "run_pipelines",
		Description: `Runs all pipelines passed as arguments`,
		Args:        `[pipeline-1] [pipeline-2] ...`,
		Handler:     commands.RunPipelines,
		Flags:       commands.RunPipelineOptions{},
		Group:       groupPipelines,
	},
	{
		Name:        "run_default_pipeline",
		Description: `Runs the default pipeline passed as arguments`,
		Args:        `[pipeline]`,
		Handler:     commands.RunDefaultPipeline,
		Group:       groupPipelines,
	},
	{
		Name:        "run_dependency_pipelines",
		Description: `Runs a pipeline of each dependency passed as arguments`,
		Args:        `[dependency-1] [dependency-2] ...`,
		Handler:     commands.RunDependencyPipelines,
		Flags:       commands.RunDependencyPipelinesOptions{},
		Group:       groupPipelines,
	},
	{
		Name:        "is_dependency",
		Description: `Returns exit code 0 if the pipeline currently being executed is run because the project is a dependency of another project`,
		Handler:     commands.IsDependency,
		Return:      reflect.Int.String(),
		Group:       groupChecks,
	},
	{
		Name:        "is_empty",
		Description: `Returns exit code 0 if the value of the argument is empty string`,
		Args:        `[value]`,
		Handler:     basiccommands.IsEmpty,
		Return:      reflect.Int.String(),
		Group:       groupChecks,
		IsGlobal:    true,
	},
	{
		Name:        "is_equal",
		Description: `Returns exit code 0 if the values of both arguments provided are equal`,
		Args:        `[value-1] [value-2]`,
		Handler:     basiccommands.IsEqual,
		Return:      reflect.Int.String(),
		Group:       groupChecks,
		IsGlobal:    true,
	},
	{
		Name:        "is_in",
		Description: "Returns exit code 0 if the value of the first argument can be found in the second argument (second argument being a blank-separated list of strings e.g `\"bananas apples peaches\"`)",
		Args:        `[value-1] [value-2]`,
		Handler:     basiccommands.IsIn,
		Return:      reflect.Int.String(),
		Group:       groupChecks,
		IsGlobal:    true,
	},
	{
		Name:        "is_os",
		Description: `Returns exit code 0 if the current operating system equals the value provided as argument`,
		Args:        `[os]`,
		ArgEnum:     []string{"darwin", "linux", "windows", "aix", "android", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "nacl", "netbsd", "openbsd", "plan9", "solaris", "zos"},
		Handler:     basiccommands.IsOS,
		Return:      reflect.Int.String(),
		Group:       groupChecks,
		IsGlobal:    true,
	},
	{
		Name:        "is_true",
		Description: `Returns exit code 0 if the value of the argument is "true"`,
		Args:        `[value]`,
		Handler:     basiccommands.IsTrue,
		Return:      reflect.Int.String(),
		Group:       groupChecks,
		IsGlobal:    true,
	},
	{
		Name:        "select_pod",
		Description: `Returns the name of a Kubernetes pod`,
		Handler:     commands.SelectPod,
		Flags:       commands.SelectPodOptions{},
		Return:      reflect.String.String(),
		Group:       groupOther,
	},
	{
		Name:        "wait_pod",
		Description: "Waits for a pod to become running",
		Args:        `[command]`,
		Handler:     commands.WaitPod,
		Flags:       commands.WaitPodOptions{},
		Group:       groupOther,
	},
	{
		Name:        "exec_container",
		Description: `Executes the command provided as argument inside a container`,
		Args:        `[command]`,
		Flags:       commands.ExecContainerOptions{},
		Handler:     commands.ExecContainer,
		Group:       groupOther,
	},
	{
		Name:        "get_config_value",
		Description: `Returns the value of the config loaded from devspace.yaml`,
		Args:        `[json.path]`,
		Handler:     commands.GetConfigValue,
		Return:      reflect.String.String(),
		Group:       groupOther,
	},
	{
		Name:        "cat",
		Description: `Returns the content of a file`,
		Args:        `[file-path]`,
		Handler:     basiccommands.Cat,
		Return:      reflect.String.String(),
		Group:       groupOther,
		IsGlobal:    true,
	},
	{
		Name:        "get_flag",
		Description: `Returns the value of the flag that is provided as argument`,
		Args:        `[flag-name]`,
		Handler:     basiccommands.GetFlag,
		Return:      reflect.String.String(),
		Group:       groupOther,
		IsGlobal:    true,
	},
	{
		Name:        "run_watch",
		Description: `Executes the command provided as argument and watches for conditions to restart the command`,
		Args:        `[command]`,
		Handler:     basiccommands.RunWatch,
		Flags:       basiccommands.RunWatchOptions{},
		Group:       groupOther,
		IsGlobal:    true,
	},
	{
		Name:        "sleep",
		Description: `Pauses the script execution for the number of seconds provided as argument`,
		Args:        `[seconds]`,
		Handler:     basiccommands.Sleep,
		Group:       groupOther,
		IsGlobal:    true,
	},
	{
		Name:        "xargs",
		Description: "Reads from stdin, splits input by blanks and executes the command provided as argument for each blank-separated input value (often used in pipes, e.g. `echo 'image-1 image-2' | xargs build_images`)",
		Args:        `[command]`,
		Handler:     basiccommands.XArgs,
		Flags:       basiccommands.XArgsOptions{},
		Group:       groupOther,
		IsGlobal:    true,
	},
}
