package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// SyncCmd is a struct that defines a command call for "sync"
type SyncCmd struct {
	*flags.GlobalFlags

	LabelSelector string
	Container     string
	Pod           string
	Pick          bool

	Exclude       []string
	ContainerPath string
	LocalPath     string
	Verbose       bool

	NoWatch               bool
	DownloadOnInitialSync bool

	Config string
}

// NewSyncCmd creates a new init command
func NewSyncCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SyncCmd{GlobalFlags: globalFlags}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Starts a bi-directional sync between the target container and the local path",
		Long: `
#######################################################
################### devspace sync #####################
#######################################################
Starts a bi-directionaly sync between the target container
and the current path:

devspace sync
devspace sync --local-path=subfolder --container-path=/app
devspace sync --exclude=node_modules --exclude=test
devspace sync --pod=my-pod --container=my-container
devspace sync --container-path=/my-path
#######################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, cobraCmd, args)
		},
	}

	syncCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	syncCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	syncCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().BoolVar(&cmd.Pick, "pick", false, "Select a pod")

	syncCmd.Flags().StringSliceVarP(&cmd.Exclude, "exclude", "e", []string{}, "Exclude directory from sync")
	syncCmd.Flags().StringVar(&cmd.LocalPath, "local-path", ".", "Local path to use (Default is current directory")
	syncCmd.Flags().StringVar(&cmd.ContainerPath, "container-path", "", "Container path to use (Default is working directory)")
	syncCmd.Flags().BoolVar(&cmd.DownloadOnInitialSync, "download-on-initial-sync", true, "Downloads all locally non existing remote files in the beginning")
	syncCmd.Flags().BoolVar(&cmd.NoWatch, "no-watch", false, "Synchronizes local and remote and then stops")
	syncCmd.Flags().BoolVar(&cmd.Verbose, "verbose", false, "Shows every file that is synced")

	syncCmd.Flags().StringVar(&cmd.Config, "config", "", "Tells DevSpace to load the sync configuration from the given devspace.yaml. Can be used together with --profile")

	return syncCmd
}

// Run executes the command logic
func (cmd *SyncCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Switch working directory
	if cmd.Config != "" {
		_, err := os.Stat(cmd.Config)
		if err != nil {
			return errors.Errorf("--config is specified, but config %s cannot be loaded: %v", cmd.Config, err)
		}

		configPath, _ := filepath.Abs(cmd.Config)
		configPath = filepath.Dir(configPath)

		err = os.Chdir(configPath)
		if err != nil {
			return errors.Wrap(err, "change working directory")
		}
	}

	// Load generated config if possible
	var err error
	var generatedConfig *generated.Config
	logger := f.GetLog()
	configLoader := f.NewConfigLoader(cmd.ToConfigOptions(), logger)
	if configLoader.Exists() {
		generatedConfig, err = configLoader.Generated()
		if err != nil {
			return err
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, logger)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, logger)
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = f.NewSpaceResumer(client, logger).ResumeSpace(true)
	if err != nil {
		return err
	}

	var config *latest.Config
	if configLoader.Exists() {
		config, err = configLoader.Load()
		if err != nil {
			return err
		}
	}

	// Build params
	params := targetselector.CmdParameter{
		ContainerName: cmd.Container,
		LabelSelector: cmd.LabelSelector,
		Namespace:     cmd.Namespace,
		PodName:       cmd.Pod,
	}
	if cmd.Pick != false {
		params.Pick = &cmd.Pick
	}

	selectorParameter := &targetselector.SelectorParameter{
		CmdParameter: params,
	}

	syncConfig := &latest.SyncConfig{
		LocalSubPath:          cmd.LocalPath,
		ContainerPath:         cmd.ContainerPath,
		DownloadOnInitialSync: &cmd.DownloadOnInitialSync,
		WaitInitialSync:       &cmd.NoWatch,
		ExcludePaths:          cmd.Exclude,
	}

	if cmd.Config != "" && config.Dev != nil && len(config.Dev.Sync) > 0 {
		// Check which sync config should be used
		loadedSyncConfig := config.Dev.Sync[0]
		if len(config.Dev.Sync) > 1 {
			// Select syncConfig to use
			syncConfigNames := []string{}
			for idx, sc := range config.Dev.Sync {
				localPath := sc.LocalSubPath
				if localPath == "" {
					localPath = "."
				}

				remotePath := sc.ContainerPath
				if remotePath == "" {
					remotePath = "."
				}

				syncConfigNames = append(syncConfigNames, fmt.Sprintf("%d: Sync %s (local) <-> %s (container)", idx, localPath, remotePath))
			}

			answer, err := logger.Question(&survey.QuestionOptions{
				Question:     "Multiple sync configurations found. Which one do you want to use?",
				DefaultValue: syncConfigNames[0],
				Options:      syncConfigNames,
			})
			if err != nil {
				return err
			}

			for idx, n := range syncConfigNames {
				if answer == n {
					loadedSyncConfig = config.Dev.Sync[idx]
					break
				}
			}
		}

		if syncConfig.LocalSubPath != "" {
			loadedSyncConfig.LocalSubPath = syncConfig.LocalSubPath
		}
		if syncConfig.ContainerPath != "" {
			loadedSyncConfig.ContainerPath = syncConfig.ContainerPath
		}
		if len(syncConfig.ExcludePaths) > 0 {
			loadedSyncConfig.ExcludePaths = syncConfig.ExcludePaths
		}
		if params.ContainerName != "" {
			loadedSyncConfig.ContainerName = params.ContainerName
		}
		if params.LabelSelector != "" || params.PodName != "" {
			loadedSyncConfig.LabelSelector = nil
			loadedSyncConfig.ImageName = ""
		}
		if params.Namespace != "" {
			loadedSyncConfig.Namespace = ""
		}

		syncConfig = loadedSyncConfig
		selectorParameter.ConfigParameter = targetselector.ConfigParameter{
			Namespace:     syncConfig.Namespace,
			LabelSelector: syncConfig.LabelSelector,
			ContainerName: syncConfig.ContainerName,
		}
	}

	// Start terminal
	servicesClient := f.NewServicesClient(config, generatedConfig, client, selectorParameter, logger)
	return servicesClient.StartSyncFromCmd(syncConfig, cmd.Verbose)
}
