package cmd

import (
	"time"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	latest "github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/server"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

// UICmd holds the open cmd flags
type UICmd struct {
	*flags.GlobalFlags

	Dev bool
}

// NewUICmd creates a new ui command
func NewUICmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UICmd{GlobalFlags: globalFlags}

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Opens the client ui in the browser",
		Long: `
#######################################################
##################### devspace ui #####################
#######################################################
Opens the client ui in the browser
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: cmd.RunUI,
	}

	uiCmd.Flags().BoolVar(&cmd.Dev, "dev", false, "Will ignore download ui errors")
	return uiCmd
}

// ClientUIPort of devspace ui
const ClientUIPort = 8090

// RunUI executes the functionality "devspace ui"
func (cmd *UICmd) RunUI(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot(log.GetInstance())
	if err != nil {
		return err
	}

	var (
		config          *latest.Config
		generatedConfig *generated.Config
	)

	if configExists {
		// Load generated config
		generatedConfig, err = generated.LoadConfig(cmd.Profile)
		if err != nil {
			return errors.Errorf("Error loading generated.yaml: %v", err)
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, log.GetInstance())
	if err != nil {
		return err
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// Warn the user if we deployed into a different context before
	err = client.PrintWarning(generatedConfig, cmd.NoWarn, true, log.GetInstance())
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
		configOptions := cmd.ToConfigOptions()
		config, err = configutil.GetConfig(configOptions)
		if err != nil {
			return err
		}
	}

	// Override error runtime handler
	log.OverrideRuntimeErrorHandler(true)

	// Create server
	server, err := server.NewServer(client, config, generatedConfig, cmd.Dev, log.GetInstance())
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

	// Start server
	return server.ListenAndServe()
}
