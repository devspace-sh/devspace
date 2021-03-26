package cmd

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"k8s.io/apimachinery/pkg/labels"
	"os"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	latest "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/survey"
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
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()

			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	syncCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to sync to")
	syncCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to sync to")
	syncCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().BoolVar(&cmd.Pick, "pick", true, "Select a pod")

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
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	if configLoader.Exists() {
		generatedConfig, err = configLoader.LoadGenerated(configOptions)
		if err != nil {
			return err
		}

		configOptions.GeneratedConfig = generatedConfig
	} else {
		logger.Warnf("If you want to use the sync paths from `devspace.yaml`, use the `--config=devspace.yaml` flag for this command.")
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
	configOptions.KubeClient = client

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, logger)
	if err != nil {
		return err
	}

	var config *latest.Config
	if configLoader.Exists() {
		configInterface, err := configLoader.Load(configOptions, logger)
		if err != nil {
			return err
		}

		config = configInterface.Config()
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "sync", client.CurrentContext(), client.Namespace(), config)
	if err != nil {
		return err
	}

	// Build params
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)
	options.Wait = ptr.Bool(false)
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

				selector := ""
				if sc.ImageName != "" {
					selector = "image: " + sc.ImageName
				} else if len(sc.LabelSelector) > 0 {
					selector = "selector: " + labels.Set(sc.LabelSelector).String()
				}

				syncConfigNames = append(syncConfigNames, fmt.Sprintf("%d: Sync %s: %s <-> %s ", idx, selector, localPath, remotePath))
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
		if options.ContainerName != "" {
			loadedSyncConfig.ContainerName = options.ContainerName
		}
		if options.LabelSelector != "" || options.Pod != "" {
			loadedSyncConfig.LabelSelector = nil
			loadedSyncConfig.ImageName = ""
		}
		if options.Namespace != "" {
			loadedSyncConfig.Namespace = ""
		}
		if *syncConfig.DisableDownload {
			loadedSyncConfig.DisableDownload = syncConfig.DisableDownload
		}
		if *syncConfig.DisableUpload {
			loadedSyncConfig.DisableUpload = syncConfig.DisableUpload
		}

		syncConfig = loadedSyncConfig
		options = options.ApplyConfigParameter(syncConfig.LabelSelector, syncConfig.Namespace, syncConfig.ContainerName, "")
	}

	// Start terminal
	return f.NewServicesClient(config, generatedConfig, client, logger).StartSyncFromCmd(options, syncConfig, nil, cmd.Verbose)
}
