package cmd

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"os"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
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

	InitialSync string

	Verbose bool

	NoWatch               bool
	DownloadOnInitialSync bool
	DownloadOnly          bool
	UploadOnly            bool
}

// NewSyncCmd creates a new init command
func NewSyncCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
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
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	syncCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to execute command")
	syncCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to open a shell to")
	syncCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().BoolVar(&cmd.Pick, "pick", false, "Select a pod")

	syncCmd.Flags().StringSliceVarP(&cmd.Exclude, "exclude", "e", []string{}, "Exclude directory from sync")
	syncCmd.Flags().StringVar(&cmd.LocalPath, "local-path", "", "Local path to use (Default is current directory")
	syncCmd.Flags().StringVar(&cmd.ContainerPath, "container-path", "", "Container path to use (Default is working directory)")

	syncCmd.Flags().BoolVar(&cmd.DownloadOnInitialSync, "download-on-initial-sync", true, "DEPRECATED: Downloads all locally non existing remote files in the beginning")
	syncCmd.Flags().StringVar(&cmd.InitialSync, "initial-sync", "", "The initial sync strategy to use (mirrorLocal, mirrorRemote, preferLocal, preferRemote, preferNewest, keepAll)")

	syncCmd.Flags().BoolVar(&cmd.NoWatch, "no-watch", false, "Synchronizes local and remote and then stops")
	syncCmd.Flags().BoolVar(&cmd.Verbose, "verbose", false, "Shows every file that is synced")

	syncCmd.Flags().BoolVar(&cmd.UploadOnly, "upload-only", false, "If set DevSpace will only upload files")
	syncCmd.Flags().BoolVar(&cmd.DownloadOnly, "download-only", false, "If set DevSpace will only download files")

	return syncCmd
}

// Run executes the command logic
func (cmd *SyncCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Switch working directory
	if cmd.GlobalFlags.ConfigPath != "" {
		_, err := os.Stat(cmd.GlobalFlags.ConfigPath)
		if err != nil {
			return errors.Errorf("--config is specified, but config %s cannot be loaded: %v", cmd.GlobalFlags.ConfigPath, err)
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

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, "sync", cmd.KubeContext, cmd.Namespace)
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

	if cmd.DownloadOnly && cmd.UploadOnly {
		return errors.New("--upload-only cannot be used together with --download-only")
	}

	syncConfig := &latest.SyncConfig{
		LocalSubPath:    cmd.LocalPath,
		ContainerPath:   cmd.ContainerPath,
		DisableDownload: &cmd.UploadOnly,
		DisableUpload:   &cmd.DownloadOnly,
		WaitInitialSync: &cmd.NoWatch,
		ExcludePaths:    cmd.Exclude,
	}

	if cmd.DownloadOnInitialSync {
		syncConfig.InitialSync = latest.InitialSyncStrategyPreferLocal
	} else {
		syncConfig.InitialSync = latest.InitialSyncStrategyMirrorLocal
	}
	if cmd.InitialSync != "" {
		if loader.ValidInitialSyncStrategy(latest.InitialSyncStrategy(cmd.InitialSync)) == false {
			return errors.Errorf("--initial-sync is not valid '%s'", cmd.InitialSync)
		}

		syncConfig.InitialSync = latest.InitialSyncStrategy(cmd.InitialSync)
	}

	if cmd.GlobalFlags.ConfigPath != "" && config.Dev != nil && len(config.Dev.Sync) > 0 {
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

		loadedSyncConfig.InitialSync = syncConfig.InitialSync
		if syncConfig.WaitInitialSync != nil && *syncConfig.WaitInitialSync == true {
			loadedSyncConfig.WaitInitialSync = syncConfig.WaitInitialSync
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
		if *syncConfig.DisableDownload {
			loadedSyncConfig.DisableDownload = syncConfig.DisableDownload
		}
		if *syncConfig.DisableUpload {
			loadedSyncConfig.DisableUpload = syncConfig.DisableUpload
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
	return servicesClient.StartSyncFromCmd(syncConfig, nil, cmd.Verbose)
}
