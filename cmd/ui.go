package cmd

import (
	"encoding/json"
	"fmt"
	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/port"
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
func NewUICmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
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
			return cmd.RunUI(f, plugins, cobraCmd, args)
		},
	}

	uiCmd.Flags().StringVar(&cmd.Host, "host", "localhost", "The host to use when opening the ui server")
	uiCmd.Flags().IntVar(&cmd.Port, "port", 0, "The port to use when opening the ui server")
	uiCmd.Flags().BoolVar(&cmd.ForceServer, "server", false, "If enabled will force start a server (otherwise an existing UI server is searched)")
	uiCmd.Flags().BoolVar(&cmd.Dev, "dev", false, "Ignore errors when downloading UI")
	return uiCmd
}

// RunUI executes the functionality "devspace ui"
func (cmd *UICmd) RunUI(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}

	// Search for an already existing server
	if cmd.ForceServer == false && cmd.Dev == false && cmd.Host == "localhost" {
		checkPort := server.DefaultPort
		if cmd.Port != 0 {
			checkPort = cmd.Port
		}

		for i := 0; i < 20; i++ {
			unused, err := port.CheckHostPort(cmd.Host, checkPort)
			if unused == false {
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
				contents, err := ioutil.ReadAll(response.Body)
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
					open.Start(domain)
					return nil
				}

				checkPort++
				continue
			}

			break
		}
	}

	var (
		config          config2.Config
		generatedConfig *generated.Config
	)

	if configExists {
		// Load generated config
		generatedConfig, err = configLoader.LoadGenerated(configOptions)
		if err != nil {
			return errors.Errorf("Error loading generated.yaml: %v", err)
		}
		configOptions.GeneratedConfig = generatedConfig
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}
	configOptions.KubeClient = client

	// Warn the user if we deployed into a different context before
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Load config
	if configExists {
		config, err = configLoader.Load(configOptions, cmd.log)
		if err != nil {
			return err
		}
	}

	// Override error runtime handler
	log.OverrideRuntimeErrorHandler(true)

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "ui", client.CurrentContext(), client.Namespace(), config.Config())
	if err != nil {
		return err
	}

	// Check if we should force the port
	var forcePort *int
	if cmd.Port != 0 {
		forcePort = &cmd.Port
	}

	// Create server
	server, err := server.NewServer(config, cmd.Host, cmd.Dev, client.CurrentContext(), client.Namespace(), forcePort, cmd.log)
	if err != nil {
		return err
	}

	// Open the browser
	if cmd.Dev == false {
		go func(domain string) {
			time.Sleep(time.Second * 2)
			_ = open.Start("http://" + domain)
		}(server.Server.Addr)
	}

	cmd.log.Infof("Start listening on http://%s", server.Server.Addr)

	// Start server
	return server.ListenAndServe()
}
