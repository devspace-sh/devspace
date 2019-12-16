package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/server"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/port"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

// UICmd holds the open cmd flags
type UICmd struct {
	*flags.GlobalFlags

	Dev bool

	Port        int
	ForceServer bool

	log log.Logger
}

// NewUICmd creates a new ui command
func NewUICmd(globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: cmd.RunUI,
	}

	uiCmd.Flags().IntVar(&cmd.Port, "port", 0, "The port to use when opening the server")
	uiCmd.Flags().BoolVar(&cmd.ForceServer, "server", false, "If enabled will force start a server (otherwise an existing UI server is searched)")
	uiCmd.Flags().BoolVar(&cmd.Dev, "dev", false, "Ignore errors when downloading UI")
	return uiCmd
}

// RunUI executes the functionality "devspace ui"
func (cmd *UICmd) RunUI(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), cmd.log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}

	// Search for an already existing server
	if cmd.ForceServer == false && cmd.Dev == false {
		checkPort := server.DefaultPort
		if cmd.Port != 0 {
			checkPort = cmd.Port
		}

		for {
			unused, _ := port.CheckHostPort("localhost", checkPort)
			if unused == false {
				domain := fmt.Sprintf("http://localhost:%d", checkPort)

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
		config          *latest.Config
		generatedConfig *generated.Config
	)

	if configExists {
		// Load generated config
		generatedConfig, err = configLoader.Generated()
		if err != nil {
			return errors.Errorf("Error loading generated.yaml: %v", err)
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Warn the user if we deployed into a different context before
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	if configExists {
		// Deprecated: Fill DEVSPACE_DOMAIN vars
		err = fillDevSpaceDomainVars(client, generatedConfig)
		if err != nil {
			return err
		}

		// Add current kube context to context
		config, err = configLoader.Load()
		if err != nil {
			return err
		}
	}

	// Override error runtime handler
	log.OverrideRuntimeErrorHandler(true)

	// Check if we should force the port
	var forcePort *int
	if cmd.Port != 0 {
		forcePort = &cmd.Port
	}

	// Create server
	server, err := server.NewServer(configLoader, config, generatedConfig, cmd.Dev, client.CurrentContext(), client.Namespace(), forcePort, cmd.log)
	if err != nil {
		return err
	}

	// Open the browser
	if cmd.Dev == false {
		go func(domain string) {
			time.Sleep(time.Second * 2)
			open.Start("http://" + domain)
		}(server.Server.Addr)
	}

	cmd.log.Infof("Start listening on http://%s", server.Server.Addr)

	// Start server
	return server.ListenAndServe()
}
