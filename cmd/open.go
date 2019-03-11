package cmd

import (
	"net/http"
	"os"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

// OpenCmd holds the open cmd flags
type OpenCmd struct {
	Provider string
}

// NewOpenCmd creates a new open command
func NewOpenCmd() *cobra.Command {
	cmd := &OpenCmd{}

	openCmd := &cobra.Command{
		Use:   "open",
		Short: "Opens the space in the browser",
		Long: `
#######################################################
#################### devspace open ####################
#######################################################
Opens the space domain in the browser

Example:
devspace open
devspace open myspace
#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunOpen,
	}

	openCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return openCmd
}

// RunOpen executes the functionality "devspace open"
func (cmd *OpenCmd) RunOpen(cobraCmd *cobra.Command, args []string) {
	// Set config root
	configExists, err := configutil.SetDevSpaceRoot()
	if err != nil {
		log.Fatal(err)
	}
	if configExists == false && len(args) == 0 {
		log.Fatal("Please specify a space name or run this command in a devspace project")
	}

	// Get space name
	var (
		spaceName    = ""
		providerName *string
	)

	if len(args) == 1 {
		spaceName = args[0]
	} else {
		generatedConfig, err := generated.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}
		if generatedConfig.CloudSpace == nil || generatedConfig.CloudSpace.Name == "" {
			log.Fatalf("No space configured in project, please specify a space name or run: \n- `%s` to create a new space\n- `%s` to use an existing space", ansi.Color("devspace create space [NAME]", "white+b"), ansi.Color("devspace use space [NAME]", "white+b"))
		}

		spaceName = generatedConfig.CloudSpace.Name
		providerName = &generatedConfig.CloudSpace.ProviderName
	}

	// Get provider
	provider, err := cloud.GetProvider(providerName, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Get space
	space, err := provider.GetSpaceByName(spaceName)
	if err != nil {
		log.Fatal(err)
	}

	kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, provider.Name)
	err = cloud.UpdateKubeConfig(kubeContext, space, false)
	if err != nil {
		log.Fatal(err)
	}

	// Loop and check if http code is != 502
	log.StartWait("Getting things ready")
	defer log.StopWait()

	// Check if domain there is a domain for the space
	if space.Domain == nil {
		log.Fatalf("Space %s has no domain. See https://devspace.cloud/docs/domains/connect on how to connect domains", space.Name)
	}

	now := time.Now()
	domain := "https://" + *space.Domain

	// Get kubernetes config
	config, err := kubectl.GetClientConfigFromContext(kubeContext)
	if err != nil {
		log.Fatal(err)
	}

	for time.Since(now) < time.Minute*4 {
		// Check if domain is ready
		resp, err := http.Get(domain)
		if err != nil {
			log.Fatalf("Error making request to %s: %v", domain, err)
		}
		if resp.StatusCode != http.StatusBadGateway {
			break
		}

		// Analyze space for issues
		report, err := analyze.CreateReport(config, space.Namespace, false)
		if err != nil {
			log.Fatalf("Error analyzing space: %v", err)
		}
		if len(report) > 0 {
			reportString := analyze.ReportToString(report)
			log.WriteString(reportString)
			os.Exit(1)
		}

		time.Sleep(time.Second * 4)
	}

	log.StopWait()
	if time.Since(now) > time.Minute*4 {
		log.Fatalf("Timeout: domain %s still returns 502 code, even after several minutes. Either the app has no valid '/' route or it is listening on the wrong port", domain)
	}

	open.Start(domain)
	log.Donef("Successfully opened %s", domain)
}
