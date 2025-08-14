package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devspace/helper/util/port"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"io"
	"net/http"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/hook"

	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

// UICmd holds the open cmd flags
type UICmd struct {
	*flags.GlobalFlags

	Dev bool

	Host        string
	Port        int
	ForceServer bool

	log log.Logger
}

// NewUICmd creates a new ui command
func NewUICmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UICmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Opens the localhost UI in the browser",
		Long: `
#######################################################
##################### devspace ui #####################
#######################################################
Opens the localhost UI in the browser
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.RunUI(f)
		},
	}

	uiCmd.Flags().StringVar(&cmd.Host, "host", "localhost", "The host to use when opening the ui server")
	uiCmd.Flags().IntVar(&cmd.Port, "port", 0, "The port to use when opening the ui server")
	uiCmd.Flags().BoolVar(&cmd.ForceServer, "server", false, "If enabled will force start a server (otherwise an existing UI server is searched)")
	uiCmd.Flags().BoolVar(&cmd.Dev, "dev", false, "Ignore errors when downloading UI")
	return uiCmd
}

// RunUI executes the functionality "devspace ui"
func (cmd *UICmd) RunUI(f factory.Factory) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}

	// Search for an already existing server
	if !cmd.ForceServer && !cmd.Dev && cmd.Host == "localhost" {
		checkPort := server.DefaultPort
		if cmd.Port != 0 {
			checkPort = cmd.Port
		}

		for i := 0; i < 20; i++ {
			available, _ := port.IsAvailable(fmt.Sprintf(":%d", checkPort))
			if !available {
				if i+1 == 20 {
					return errors.Wrap(err, "check for open port")
				}

				domain := fmt.Sprintf("http://%s:%d", cmd.Host, checkPort)

				// Check if DevSpace server
				response, err := http.Get(domain + "/api/version")
				if err != nil {
					checkPort++
					continue
				}

				defer response.Body.Close()
				contents, err := io.ReadAll(response.Body)
				if err != nil {
					checkPort++
					continue
				}

				serverVersion := &server.UIServerVersion{}
				err = json.Unmarshal(contents, serverVersion)
				if err != nil {
					checkPort++
					continue
				}

				if serverVersion.DevSpace {
					cmd.log.Infof("Found running UI server at %s", domain)
					_ = open.Start(domain)
					return nil
				}

				checkPort++
				continue
			}

			break
		}
	}

	var (
		config     config2.Config
		localCache localcache.Cache
	)

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	if configExists {
		// Load generated config
		localCache, err = configLoader.LoadLocalCache()
		if err != nil {
			return errors.Errorf("Error loading generated.yaml: %v", err)
		}
	}

	// If the current kube context or namespace is different from old,
	// show warnings and reset kube client if necessary
	client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, cmd.log)
	if err != nil {
		return err
	}

	// Load config
	if configExists {
		config, err = configLoader.LoadWithCache(context.Background(), localCache, client, configOptions, cmd.log)
		if err != nil {
			return err
		}
	}

	// dev context
	ctx := devspacecontext.NewContext(context.Background(), nil, cmd.log).
		WithConfig(config).
		WithKubeClient(client)

	// Override error runtime handler
	log.OverrideRuntimeErrorHandler(true)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "ui")
	if err != nil {
		return err
	}

	// Check if we should force the port
	var forcePort *int
	if cmd.Port != 0 {
		forcePort = &cmd.Port
	}

	// Create server
	server, err := server.NewServer(ctx, cmd.Host, cmd.Dev, forcePort, nil)
	if err != nil {
		return err
	}

	// Open the browser
	if !cmd.Dev {
		go func(domain string) {
			time.Sleep(time.Second * 2)
			_ = open.Start("http://" + domain)
		}(server.Server.Addr)
	}

	cmd.log.Infof("Start listening on http://%s", server.Server.Addr)

	// Start server
	return server.ListenAndServe()
}
