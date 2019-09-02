package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"

	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const openLocalHostOption = "via localhost (provides private access only on your computer via port-forwarding)"
const openDomainOption = "via domain (makes your application publicly available via ingress)"

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

	var (
		providerName *string
		provider     *cloud.Provider
		spaceName    = ""
		space        *cloud.Space
		domain       string
		tls          bool
	)

	// Get default namespace
	var devspaceConfig *latest.Config
	if configExists {
		// Get config with adjusted cluster config
		_, err := configutil.GetContextAdjustedConfig("", "", false)
		if err != nil {
			log.Fatal(err)
		}

		// Signal that we are working on the space if there is any
		err = cloud.ResumeLatestSpace(devspaceConfig, true, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}
	}
	namespace, err := configutil.GetDefaultNamespace(devspaceConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Get kubernetes client
	client, err := kubectl.NewClient(devspaceConfig)
	if err != nil {
		log.Fatal(err)
	}

	currentContext, _, err := kubeconfig.GetCurrentContext()

	if len(args) == 1 {
		spaceName = args[0]

		// Get provider
		provider, err = cloud.GetProvider(providerName, log.GetInstance())
		if err != nil {
			log.Fatal(err)
		}

		// Get space
		space, err = provider.GetSpaceByName(spaceName)
		if err != nil {
			log.Fatal(err)
		}

		// Update the current kubectl context to the one of the Space provided
		log.StartWait("Retrieve service account data")

		// Change kube context
		kubeContext := cloud.GetKubeContextNameFromSpace(space.Name, space.ProviderName)
		serviceAccount, err := provider.GetServiceAccount(space)
		if err != nil {
			log.Fatalf("Error retrieving space service account: %v", err)
		}
		err = cloud.UpdateKubeConfig(kubeContext, serviceAccount, space.SpaceID, space.ProviderName, true)
		if err != nil {
			log.Fatalf("Error saving kube config: %v", err)
		}

		log.StopWait()
	} else {
		spaceID, currentContextProvider, err := kubeconfig.GetSpaceID(currentContext)
		if err == nil { // Current kube-context is a Space
			if providerName == nil {
				providerName = &currentContextProvider
			}

			// Get provider
			provider, err = cloud.GetProvider(providerName, log.GetInstance())
			if err != nil {
				log.Fatal(err)
			}

			// Get space
			space, err = provider.GetSpace(spaceID)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	openingMode := survey.Question(&survey.QuestionOptions{
		Question:     "How do you want to open your application?",
		DefaultValue: openLocalHostOption,
		Options: []string{
			openLocalHostOption,
			openDomainOption,
		},
	})

	if openingMode == openLocalHostOption {
		ports := []string{}

		if devspaceConfig.Dev == nil {
			devspaceConfig.Dev = &latest.DevConfig{}
		}

		if devspaceConfig.Dev.Ports == nil {
			portConfigs := []*latest.PortForwardingConfig{}
			devspaceConfig.Dev.Ports = &portConfigs
		}
		portConfigs := *devspaceConfig.Dev.Ports

		// search for ports
		for i := range portConfigs {
			portMappings := *portConfigs[i].PortMappings

			for ii := range portConfigs {
				portMap := *portMappings[ii]

				if portMap.LocalPort != nil {
					ports = append(ports, strconv.Itoa(*portMap.LocalPort))
				}
			}
		}
		port := ""

		// if no port is found, ask users which port and add it to config in-memory
		if len(ports) == 0 {
			port := survey.Question(&survey.QuestionOptions{
				Question: "Which port does your application run on?",
			})

			intPort, err := strconv.Atoi(port)
			if err != nil {
				log.Fatal("Invalid port '%s': %v", port, err)
			}

			portMappings := []*latest.PortMapping{
				&latest.PortMapping{
					LocalPort: ptr.Int(intPort),
				},
			}

			portConfigs = append(portConfigs, &latest.PortForwardingConfig{
				PortMappings: &portMappings,
			})
		} else if len(ports) == 1 {
			port = ports[0]
		} else {
			port = survey.Question(&survey.QuestionOptions{
				Question: "Which port do you want to access?",
				Options:  ports,
			})
		}
		domain = "localhost:" + port

		// start port-forwarding for localhost access
		portForwarder, err := services.StartPortForwarding(devspaceConfig, client, log.GetInstance())
		if err != nil {
			log.Fatalf("Unable to start portforwarding: %v", err)
		}

		defer func() {
			for _, v := range portForwarder {
				v.Close()
			}
		}()
	} else { // create ingress for public access via domain
		if space != nil {
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

			// Check if domain exists
			domain, tls, err = findDomain(client, namespace, host)
			if err != nil {
				log.Fatal(err)
			}

			// Not found
			if domain == "" {
				err = provider.CreateIngress(devspaceConfig, client, space, host)
				if err != nil {
					log.Fatalf("Error creating ingress: %v", err)
				}

				domain, tls, err = findDomain(client, namespace, host)
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			// TODO: create ingress for regular namespaces
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

	// Make sure the ingress has some time to take effect
	time.Sleep(time.Second * 2)

	now := time.Now()
	for time.Since(now) < time.Minute*4 {
		// Check if domain is ready
		resp, err := http.Get(domain)
		if err != nil {
			log.Fatalf("Error making request to %s: %v", domain, err)
		} else if resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
			log.StopWait()
			open.Start(domain)
			log.Donef("Successfully opened %s", domain)
			return
		}

		// Analyze space for issues
		report, err := analyze.CreateReport(client, namespace, false)
		if err != nil {
			log.Fatalf("Error analyzing space: %v", err)
		}
		if len(report) > 0 {
			reportString := analyze.ReportToString(report)
			log.WriteString(reportString)
			log.Fatal("")
		}

		time.Sleep(time.Second * 5)
	}

	log.StopWait()
	log.Fatalf("Timeout: domain %s still returns 502 code, even after several minutes. Either the app has no valid '/' route or it is listening on the wrong port", domain)
}

func findDomain(client kubernetes.Interface, namespace, host string) (string, bool, error) {
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
