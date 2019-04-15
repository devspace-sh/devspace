package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	// Check if domain there is a domain for the space
	if len(space.Domains) == 0 {
		log.Fatalf("Space %s has no connected domain", space.Name)
	}

	// Select domain
	domains := make([]string, 0, len(space.Domains))
	for _, domain := range space.Domains {
		domains = append(domains, domain.URL)
	}

	host := ""
	if len(domains) == 1 {
		host = domains[0]
	} else {
		host = survey.Question(&survey.QuestionOptions{
			Question: "Please select a domain to open",
			Options:  domains,
		})
	}

	// If there is no config make sure the current kubectl context is correct
	if configExists == false {
		log.StartWait("Retrieve service account data")

		// Change kube context
		kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)
		serviceAccount, err := provider.GetServiceAccount(space)
		if err != nil {
			log.Fatalf("Error retrieving space service account: %v", err)
		}
		err = cloud.UpdateKubeConfig(kubeContext, serviceAccount, true)
		if err != nil {
			log.Fatalf("Error saving kube config: %v", err)
		}

		log.StopWait()
	}

	// Get default namespace
	var devspaceConfig *latest.Config
	if configExists {
		devspaceConfig = configutil.GetConfig()
	}
	namespace, err := configutil.GetDefaultNamespace(devspaceConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Get kubernetes config
	config, err := kubectl.GetClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Check if domain exists
	domain, tls, err := findDomain(client, namespace, host)
	if err != nil {
		log.Fatal(err)
	}

	// Not found
	if domain == "" {
		err = provider.CreateIngress(client, space, host)
		if err != nil {
			log.Fatalf("Error creating ingress: %v", err)
		}

		domain, tls, err = findDomain(client, namespace, host)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Add schema
	if tls {
		domain = "https://" + domain
	} else {
		domain = "http://" + domain
	}

	// Loop and check if http code is != 502
	log.StartWait("Waiting for ingress")
	defer log.StopWait()

	now := time.Now()
	for time.Since(now) < time.Minute*4 {
		// Check if domain is ready
		resp, err := http.Get(domain)
		if err != nil {
			log.Fatalf("Error making request to %s: %v", domain, err)
		} else if resp.StatusCode != http.StatusBadGateway {
			log.StopWait()
			open.Start(domain)
			log.Donef("Successfully opened %s", domain)
			os.Exit(0)
		}

		// Analyze space for issues
		report, err := analyze.CreateReport(config, namespace, false)
		if err != nil {
			log.Fatalf("Error analyzing space: %v", err)
		}
		if len(report) > 0 {
			reportString := analyze.ReportToString(report)
			log.WriteString(reportString)
			os.Exit(1)
		}

		time.Sleep(time.Second * 5)
	}

	log.StopWait()
	log.Fatalf("Timeout: domain %s still returns 502 code, even after several minutes. Either the app has no valid '/' route or it is listening on the wrong port", domain)
}

func findDomain(client *kubernetes.Clientset, namespace, host string) (string, bool, error) {
	log.StartWait("Retrieve ingresses")
	defer log.StopWait()

	// List all ingresses and only create one if there is none already
	ingressList, err := client.ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", false, fmt.Errorf("Error listing ingresses: %v", err)
	}

	// Check ingresses for our domain
	domain := ""
	tls := false
	for _, ingress := range ingressList.Items {
		for _, rule := range ingress.Spec.Rules {
			if strings.TrimSpace(rule.Host) == host {
				domain = host
			}
		}

		// Check if tls is enabled
		if domain != "" {
			for _, tlsEntry := range ingress.Spec.TLS {
				for _, host := range tlsEntry.Hosts {
					if strings.TrimSpace(host) == host {
						tls = true
					}
				}
			}

			break
		}
	}

	return domain, tls, nil
}
