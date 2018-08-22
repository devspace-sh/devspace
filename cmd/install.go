// +build windows

package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"

	"github.com/covexo/devspace/pkg/devspace/config/v1"

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/registry"
	"k8s.io/client-go/kubernetes"
)

type InstallCmd struct {
	flags         *InstallCmdFlags
	helm          *helmClient.HelmClientWrapper
	kubectl       *kubernetes.Clientset
	privateConfig *v1.PrivateConfig
	dsConfig      *v1.DevSpaceConfig
	workdir       string
}

type InstallCmdFlags struct {
}

func init() {
	cmd := &InstallCmd{
		flags: &InstallCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "install",
		Short: "Installs the DevSpace CLI",
		Long: `
#######################################################
################## devspace install ###################
#######################################################
Registers the devspace executable in your PATH
variable.
#######################################################`,
		Run: cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

func (cmd *InstallCmd) Run(cobraCmd *cobra.Command, args []string) {
	executablePath, err := os.Executable()

	if err != nil {
		panic(err)
	}
	executableDir := filepath.Dir(executablePath)
	addToPath(executableDir)
}

func addToPath(path string) {
	if runtime.GOOS == "windows" {
		envVarPath := "PATH"
		pathSeparator := ";"
		registryKey, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.ALL_ACCESS)

		if err != nil {
			log.WithError(err).Panic("Unable to open env var registry key.")
		}
		defer registryKey.Close()

		pathVar, _, err := registryKey.GetStringValue(envVarPath)

		if err != nil {
			log.WithError(err).Panic("Unable to read " + envVarPath + " env var from registry.")
		}
		paths := strings.Split(pathVar, pathSeparator)
		pathIsPresent := false

		for _, existingPath := range paths {
			if path == existingPath {
				pathIsPresent = true
				break
			}
		}

		if !pathIsPresent {
			paths = append(paths, path)
		}
		err = registryKey.SetStringValue(envVarPath, strings.Join(paths, pathSeparator))

		if err != nil {
			log.WithError(err).Panic("Unable to write " + envVarPath + " env var in registry.")
		}
	}
}
