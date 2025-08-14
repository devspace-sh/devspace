package cmd

import (
	"context"
	"fmt"
	"os"

	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/sync"

	"github.com/loft-sh/devspace/pkg/devspace/hook"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/message"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/loft-sh/devspace/cmd/flags"
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
	Polling       bool

	Exclude []string
	Path    string

	InitialSync string

	NoWatch               bool
	DownloadOnInitialSync bool
	DownloadOnly          bool
	UploadOnly            bool

	// used for testing to allow interruption
	Ctx context.Context
}

// NewSyncCmd creates a new init command
func NewSyncCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SyncCmd{GlobalFlags: globalFlags}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Starts a bi-directional sync between the target container and the local path",
		Long: `
#############################################################################
################### devspace sync ###########################################
#############################################################################
Starts a bi-directional(default) sync between the target container path
and local path:

devspace sync --path=.:/app # localPath is current dir and remotePath is /app
devspace sync --path=.:/app --image-selector nginx:latest
devspace sync --path=.:/app --exclude=node_modules,test
devspace sync --path=.:/app --pod=my-pod --container=my-container
#############################################################################`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Print upgrade message if new version available
			upgrade.PrintUpgradeMessage(f.GetLog())
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
	syncCmd.Flags().StringVar(&cmd.Path, "path", "", "Path to use (Default is current directory). Example: ./local-path:/remote-path or local-path:.")

	syncCmd.Flags().BoolVar(&cmd.DownloadOnInitialSync, "download-on-initial-sync", true, "DEPRECATED: Downloads all locally non existing remote files in the beginning")
	syncCmd.Flags().StringVar(&cmd.InitialSync, "initial-sync", "", "The initial sync strategy to use (mirrorLocal, mirrorRemote, preferLocal, preferRemote, preferNewest, keepAll)")

	syncCmd.Flags().BoolVar(&cmd.NoWatch, "no-watch", false, "Synchronizes local and remote and then stops")

	syncCmd.Flags().BoolVar(&cmd.UploadOnly, "upload-only", false, "If set DevSpace will only upload files")
	syncCmd.Flags().BoolVar(&cmd.DownloadOnly, "download-only", false, "If set DevSpace will only download files")

	syncCmd.Flags().BoolVar(&cmd.Wait, "wait", true, "Wait for the pod(s) to start if they are not running")
	syncCmd.Flags().BoolVar(&cmd.Polling, "polling", false, "If polling should be used to detect file changes in the container")

	return syncCmd
}

type nameConfig struct {
	name          string
	devPod        *latest.DevPod
	containerName string
	syncConfig    *latest.SyncConfig
}

// Run executes the command logic
func (cmd *SyncCmd) Run(f factory.Factory) error {
	if cmd.Ctx == nil {
		var cancelFn context.CancelFunc
		cmd.Ctx, cancelFn = context.WithCancel(context.Background())
		defer cancelFn()
	}

	// Switch working directory
	if cmd.ConfigPath != "" {
		_, err := os.Stat(cmd.ConfigPath)
		if err != nil {
			return errors.Errorf("--config is specified, but config %s cannot be loaded: %v", cmd.GlobalFlags.ConfigPath, err)
		}
	}

	// Load generated config if possible
	var err error
	var localCache localcache.Cache
	logger := f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	if configLoader.Exists() {
		if cmd.GlobalFlags.ConfigPath != "" {
			configExists, err := configLoader.SetDevSpaceRoot(logger)
			if err != nil {
				return err
			} else if !configExists {
				return errors.New(message.ConfigNotFound)
			}

			localCache, err = configLoader.LoadLocalCache()
			if err != nil {
				return err
			}
		} else {
			logger.Warnf("If you want to use the sync paths from `devspace.yaml`, use the `DEVSPACE_CONFIG=devspace.yaml` environment variable for this command.")
		}
	}

	// Get config with adjusted cluster config
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Wrap(err, "new kube client")
	}

	// If the current kube context or namespace is different from old,
	// show warnings and reset kube client if necessary
	client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, logger)
	if err != nil {
		return err
	}

	var configInterface config.Config
	if configLoader.Exists() && cmd.GlobalFlags.ConfigPath != "" {
		configInterface, err = configLoader.LoadWithCache(context.Background(), localCache, client, configOptions, logger)
		if err != nil {
			return err
		}
	}

	// create the devspace context
	ctx := devspacecontext.NewContext(cmd.Ctx, nil, logger).
		WithConfig(configInterface).
		WithKubeClient(client)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "sync")
	if err != nil {
		return err
	}

	// get image selector if specified
	imageSelector, err := getImageSelector(ctx, configLoader, configOptions, cmd.ImageSelector)
	if err != nil {
		return err
	}

	// Build params
	options := targetselector.NewOptionsFromFlags(cmd.Container, cmd.LabelSelector, imageSelector, cmd.Namespace, cmd.Pod).
		WithPick(cmd.Pick).
		WithWait(cmd.Wait)

	if cmd.DownloadOnly && cmd.UploadOnly {
		return errors.New("--upload-only cannot be used together with --download-only")
	}

	// Create the sync config to apply
	syncConfig := nameConfig{
		devPod:     &latest.DevPod{},
		syncConfig: &latest.SyncConfig{},
	}
	if cmd.GlobalFlags.ConfigPath != "" && configInterface != nil {
		devSection := configInterface.Config().Dev
		syncConfigs := []nameConfig{}
		for _, v := range devSection {
			loader.EachDevContainer(v, func(devContainer *latest.DevContainer) bool {
				for _, s := range devContainer.Sync {
					n, err := fromSyncConfig(v, devContainer.Container, s)
					if err != nil {
						return true
					}
					syncConfigs = append(syncConfigs, n)
				}
				return true
			})
		}
		if len(syncConfigs) == 0 {
			return fmt.Errorf("no sync config found in %s", cmd.GlobalFlags.ConfigPath)
		}

		// Check which sync config should be used
		if len(syncConfigs) > 1 {
			// Select syncConfig to use
			syncConfigNames := []string{}
			for _, sc := range syncConfigs {
				syncConfigNames = append(syncConfigNames, sc.name)
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
					syncConfig = syncConfigs[idx]
					break
				}
			}
		} else {
			syncConfig = syncConfigs[0]
		}
	}

	// apply the flags to the empty sync config or loaded sync config from the devspace.yaml
	var configImageSelector []string
	if syncConfig.devPod.ImageSelector != "" {
		imageSelector, err := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsImageSelector(ctx.Context(), syncConfig.devPod.ImageSelector, ctx.Config(), ctx.Dependencies())
		if err != nil {
			return err
		}

		configImageSelector = []string{imageSelector.Image}
	}
	options = options.ApplyConfigParameter(syncConfig.containerName, syncConfig.devPod.LabelSelector, configImageSelector, syncConfig.devPod.Namespace, "")
	options, err = cmd.applyFlagsToSyncConfig(syncConfig.syncConfig, options)
	if err != nil {
		return errors.Wrap(err, "apply flags to sync config")
	}

	// Start sync
	options = options.WithSkipInitContainers(true)
	return sync.StartSyncFromCmd(ctx, targetselector.NewTargetSelector(options), syncConfig.devPod.Name, syncConfig.syncConfig, cmd.NoWatch)
}

func fromSyncConfig(devPod *latest.DevPod, containerName string, sc *latest.SyncConfig) (nameConfig, error) {
	localPath, remotePath, err := sync.ParseSyncPath(sc.Path)
	if err != nil {
		return nameConfig{}, err
	}

	selector := ""
	if devPod.ImageSelector != "" {
		selector = "img-selector: " + devPod.ImageSelector
	} else if len(devPod.LabelSelector) > 0 {
		selector = "selector: " + labels.Set(devPod.LabelSelector).String()
	}
	if containerName != "" {
		selector += "/" + containerName
	}

	return nameConfig{
		name:          fmt.Sprintf("%s: Sync %s: %s <-> %s ", devPod.Name, selector, localPath, remotePath),
		devPod:        devPod,
		containerName: containerName,
		syncConfig:    sc,
	}, nil
}

func (cmd *SyncCmd) applyFlagsToSyncConfig(syncConfig *latest.SyncConfig, options targetselector.Options) (targetselector.Options, error) {
	if cmd.Path != "" {
		syncConfig.Path = cmd.Path
	}
	if len(cmd.Exclude) > 0 {
		syncConfig.ExcludePaths = cmd.Exclude
	}
	if cmd.UploadOnly {
		syncConfig.DisableDownload = cmd.UploadOnly
	}
	if cmd.DownloadOnly {
		syncConfig.DisableUpload = cmd.DownloadOnly
	}

	// if selection is specified through flags, we don't want to use the loaded
	// sync config selection from the devspace.yaml.
	if cmd.Container != "" {
		options = options.WithContainer(cmd.Container)
	}
	if cmd.LabelSelector != "" {
		options = options.WithLabelSelector(cmd.LabelSelector)
	}
	if cmd.Pod != "" {
		options = options.WithPod(cmd.Pod)
	}
	if cmd.Namespace != "" {
		options = options.WithNamespace(cmd.Namespace)
	}

	if cmd.DownloadOnInitialSync {
		syncConfig.InitialSync = latest.InitialSyncStrategyPreferLocal
	} else {
		syncConfig.InitialSync = latest.InitialSyncStrategyMirrorLocal
	}

	if cmd.InitialSync != "" {
		if !versions.ValidInitialSyncStrategy(latest.InitialSyncStrategy(cmd.InitialSync)) {
			return options, errors.Errorf("--initial-sync is not valid '%s'", cmd.InitialSync)
		}

		syncConfig.InitialSync = latest.InitialSyncStrategy(cmd.InitialSync)
	}

	if cmd.Polling {
		syncConfig.Polling = cmd.Polling
	}

	return options, nil
}
