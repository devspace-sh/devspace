package cmd

import (
	"os"
	"path"

	helmClient "github.com/covexo/devspace/pkg/devspace/clients/helm"
	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/covexo/devspace/pkg/util/stdinutil"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

// ResetCmd holds the needed command information
type ResetCmd struct {
	flags   *ResetCmdFlags
	helm    *helmClient.HelmClientWrapper
	kubectl *kubernetes.Clientset
	workdir string
}

// ResetCmdFlags holds the command flags
type ResetCmdFlags struct {
	deleteDockerfile     bool
	deleteChart          bool
	deleteRegistry       bool
	deleteTiller         bool
	deleteDevspaceFolder bool
	deleteRelease        bool
}

func init() {
	cmd := &ResetCmd{
		flags: &ResetCmdFlags{},
	}

	cobraCmd := &cobra.Command{
		Use:   "reset",
		Short: "Remove devspace completely from your project",
		Long: `
#######################################################
################### devspace reset ####################
#######################################################
Resets your project by removing all DevSpace related 
data from your project and your cluster, including:
1. DevSpace release (cluster)
2. Docker registry (cluster)
3. DevSpace config files in .devspace/ (local)

Use the flag --all-data to also remove:
1. Tiller server (cluster)
2. Helm home (local)

If you simply want to shutdown your DevSpace, use the 
command: devspace down
#######################################################`,
		Args: cobra.NoArgs,
		Run:  cmd.Run,
	}
	rootCmd.AddCommand(cobraCmd)
}

// Run executes the reset command logic
func (cmd *ResetCmd) Run(cobraCmd *cobra.Command, args []string) {
	var err error
	cmd.determineResetExtent()

	if cmd.flags.deleteRelease {
		err = cmd.deleteRelease()

		if err != nil {
			log.Failf("Error deleting release: %s", err.Error())
		} else {
			log.Done("Successfully deleted release")
		}
	}

	if cmd.flags.deleteRegistry {
		err = cmd.deleteRegistry()

		if err != nil {
			log.Failf("Error deleting docker registry: %s", err.Error())

			// if cmd.shouldContinue() == false {
			// 	return
			// }
		} else {
			log.Done("Successfully deleted docker registry")
		}
	}

	if cmd.flags.deleteTiller {
		err = cmd.deleteTiller()

		if err != nil {
			log.Failf("Error deleting tiller: %s", err.Error())

			if cmd.shouldContinue() == false {
				return
			}
		} else {
			log.Done("Successfully deleted tiller server")
		}
	}

	if cmd.flags.deleteChart {
		err = cmd.deleteChart()

		if err != nil {
			log.Failf("Error deleting chart: %s", err.Error())

			if cmd.shouldContinue() == false {
				return
			}
		} else {
			log.Done("Successfully deleted chart")
		}
	}

	if cmd.flags.deleteDockerfile {
		err = cmd.deleteDockerfile()

		if err != nil {
			log.Failf("Error deleting Dockerfile: %s", err.Error())

			if cmd.shouldContinue() == false {
				return
			}
		} else {
			log.Done("Successfully deleted Dockerfile")
		}
	}

	if cmd.flags.deleteDevspaceFolder {
		err = cmd.deleteDevspaceFolder()

		if err != nil {
			log.Failf("Error deleting .devspace folder: ", err.Error())

			if cmd.shouldContinue() == false {
				return
			}
		} else {
			log.Done("Successfully deleted .devspace folder")
		}
	}
}

func (cmd *ResetCmd) determineResetExtent() {
	cmd.flags.deleteDevspaceFolder = true
	cmd.flags.deleteRelease = true

	cmd.flags.deleteDockerfile = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Should the Dockerfile be removed? (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"

	cmd.flags.deleteChart = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Should the Chart (chart/*) be removed ? (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"

	cmd.flags.deleteRegistry = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Should the docker registry be removed ? (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"

	cmd.flags.deleteTiller = *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Should the tiller server be removed ? (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"
}

func (cmd *ResetCmd) shouldContinue() bool {
	return *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "An error occurred, should the reset command continue? (y/n)",
		DefaultValue:           "y",
		ValidationRegexPattern: "^(y|n)$",
	}) == "y"
}

func (cmd *ResetCmd) deleteRelease() error {
	var err error
	config := configutil.GetConfig(false)

	releaseName := *config.DevSpace.Release.Name

	if cmd.kubectl == nil || cmd.helm == nil {
		cmd.kubectl, err = kubectl.NewClient()

		if err != nil {
			return err
		}

		isDeployed := helmClient.IsTillerDeployed(cmd.kubectl, config.Services.Tiller)

		if isDeployed == false {
			return nil
		}

		cmd.helm, err = helmClient.NewClient(cmd.kubectl, false)

		if err != nil {
			return err
		}
	}

	_, err = cmd.helm.DeleteRelease(releaseName, true)

	return err
}

func (cmd *ResetCmd) deleteRegistry() error {
	var err error
	config := configutil.GetConfig(false)

	registryReleaseName := *config.Services.InternalRegistry.Release.Name

	if cmd.kubectl == nil || cmd.helm == nil {
		cmd.kubectl, err = kubectl.NewClient()

		if err != nil {
			return err
		}

		isDeployed := helmClient.IsTillerDeployed(cmd.kubectl, config.Services.Tiller)

		if isDeployed == false {
			return nil
		}

		cmd.helm, err = helmClient.NewClient(cmd.kubectl, false)

		if err != nil {
			return err
		}
	}

	_, err = cmd.helm.DeleteRelease(registryReleaseName, true)

	return err
}

func (cmd *ResetCmd) deleteTiller() error {
	var err error
	config := configutil.GetConfig(false)

	if cmd.kubectl == nil {
		cmd.kubectl, err = kubectl.NewClient()

		if err != nil {
			return err
		}
	}

	return helmClient.DeleteTiller(cmd.kubectl, config.Services.Tiller)
}

func (cmd *ResetCmd) deleteDockerfile() error {
	return os.Remove(path.Join(cmd.workdir, "Dockerfile"))
}

func (cmd *ResetCmd) deleteChart() error {
	return os.RemoveAll(path.Join(cmd.workdir, "chart"))
}

func (cmd *ResetCmd) deleteDevspaceFolder() error {
	return os.RemoveAll(path.Join(cmd.workdir, ".devspace"))
}
