package cmd

import (
	"fmt"
	"os"

	"github.com/loft-sh/devspace/pkg/devspace/hook"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/message"
	"k8s.io/apimachinery/pkg/labels"

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
	ImageSelector string
	Container     string
	Pod           string
	Pick          bool
	Wait          bool

	Exclude       []string
	ContainerPath string
	LocalPath     string

	InitialSync string

	Verbose bool

	NoWatch               bool
	DownloadOnInitialSync bool
	DownloadOnly          bool
	UploadOnly            bool

	// used for testing to allow interruption
	Interrupt chan error
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
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage()

			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.Run(f)
		},
	}

	syncCmd.Flags().StringVarP(&cmd.Container, "container", "c", "", "Container name within pod where to sync to")
	syncCmd.Flags().StringVar(&cmd.Pod, "pod", "", "Pod to sync to")
	syncCmd.Flags().StringVarP(&cmd.LabelSelector, "label-selector", "l", "", "Comma separated key=value selector list (e.g. release=test)")
	syncCmd.Flags().StringVar(&cmd.ImageSelector, "image-selector", "", "The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})")
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

	syncCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait for the pod(s) to start if they are not running")

	return syncCmd
}

// Run executes the command logic
func (cmd *SyncCmd) Run(f factory.Factory) error {
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
	configOptions := cmd.ToConfigOptions(logger)
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	if configLoader.Exists() {
		if cmd.GlobalFlags.ConfigPath != "" {
			configExists, err := configLoader.SetDevSpaceRoot(logger)
			if err != nil {
				return err
			} else if !configExists {
				return errors.New(message.ConfigNotFound)
			}

			generatedConfig, err = configLoader.LoadGenerated(configOptions)
			if err != nil {
				return err
			}

			configOptions.GeneratedConfig = generatedConfig
		} else {
			logger.Warnf("If you want to use the sync paths from `devspace.yaml`, use the `--config=devspace.yaml` flag for this command.")
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

	// If the current kube context or namespace is different than old,
	// show warnings and reset kube client if necessary
	client, err = client.CheckKubeContext(generatedConfig, cmd.NoWarn, logger)
	if err != nil {
		return err
	}

	configOptions.KubeClient = client

	var configInterface config.Config
	var config *latest.Config
	if configLoader.Exists() && cmd.GlobalFlags.ConfigPath != "" {
		configInterface, err = configLoader.Load(configOptions, logger)
		if err != nil {
			return err
		}

		config = configInterface.Config()
	}

	// Execute plugin hook
	err = hook.ExecuteHooks(nil, nil, nil, nil, nil, "sync")
	if err != nil {
		return err
	}

	// Build params
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, cmd.Namespace, cmd.Pod, cmd.Pick)
	// get image selector if specified
	imageSelector, err := getImageSelector(client, configLoader, configOptions, "", cmd.ImageSelector, logger)
	if err != nil {
		return err
	}

	// set image selector
	options.ImageSelector = imageSelector
	options.Wait = &cmd.Wait

	if cmd.DownloadOnly && cmd.UploadOnly {
		return errors.New("--upload-only cannot be used together with --download-only")
	}

	// Create the sync config to apply
	syncConfig := &latest.SyncConfig{}
	if cmd.GlobalFlags.ConfigPath != "" && config != nil {
		if len(config.Dev.Sync) == 0 {
			return fmt.Errorf("no sync config found in %s", cmd.GlobalFlags.ConfigPath)
		}

		// Check which sync config should be used
		syncConfig = config.Dev.Sync[0]
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
				if sc.ImageSelector != "" {
					selector = "img-selector: " + sc.ImageSelector
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
					syncConfig = config.Dev.Sync[idx]
					break
				}
			}
		}
	}

	// apply the flags to the empty sync config or loaded sync config from the devspace.yaml
	err = cmd.applyFlagsToSyncConfig(syncConfig)
	if err != nil {
		return errors.Wrap(err, "apply flags to sync config")
	}

	options = options.ApplyConfigParameter(syncConfig.LabelSelector, syncConfig.Namespace, syncConfig.ContainerName, "")

	// Start sync
	return f.NewServicesClient(configInterface, nil, client, logger).StartSyncFromCmd(options, syncConfig, cmd.Interrupt, cmd.NoWatch, cmd.Verbose)
}

func (cmd *SyncCmd) applyFlagsToSyncConfig(syncConfig *latest.SyncConfig) error {
	if cmd.LocalPath != "" {
		syncConfig.LocalSubPath = cmd.LocalPath
	}
	if cmd.ContainerPath != "" {
		syncConfig.ContainerPath = cmd.ContainerPath
	}
	if len(cmd.Exclude) > 0 {
		syncConfig.ExcludePaths = cmd.Exclude
	}
	if cmd.UploadOnly {
		syncConfig.DisableDownload = &cmd.UploadOnly
	}
	if cmd.DownloadOnly {
		syncConfig.DisableUpload = &cmd.DownloadOnly
	}

	// if selection is specified through flags, we don't want to use the loaded
	// sync config selection from the devspace.yaml.
	if cmd.Container != "" {
		syncConfig.ContainerName = ""
	}
	if cmd.LabelSelector != "" || cmd.Pod != "" || cmd.ImageSelector != "" {
		syncConfig.LabelSelector = nil
		syncConfig.ImageSelector = ""
	}
	if cmd.Namespace != "" {
		syncConfig.Namespace = ""
	}

	if cmd.DownloadOnInitialSync {
		syncConfig.InitialSync = latest.InitialSyncStrategyPreferLocal
	} else {
		syncConfig.InitialSync = latest.InitialSyncStrategyMirrorLocal
	}

	if cmd.InitialSync != "" {
		if !loader.ValidInitialSyncStrategy(latest.InitialSyncStrategy(cmd.InitialSync)) {
			return errors.Errorf("--initial-sync is not valid '%s'", cmd.InitialSync)
		}

		syncConfig.InitialSync = latest.InitialSyncStrategy(cmd.InitialSync)
	}

	return nil
}
