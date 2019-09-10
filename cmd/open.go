package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"

	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"crypto/sha1"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const openLocalHostOption = "via localhost (provides private access only on your computer via port-forwarding)"
const openDomainOption = "via domain (makes your application publicly available via ingress)"

// OpenCmd holds the open cmd flags
type OpenCmd struct {
	Provider string

	Namespace   string
	KubeContext string
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
#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunOpen,
	}

	openCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	openCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The namespace to use")
	openCmd.Flags().StringVar(&cmd.KubeContext, "kube-context", "", "The kubernetes context to use")

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
		providerName             *string
		provider                 *cloud.Provider
		space                    *cloudlatest.Space
		domain                   string
		tls                      bool
		ingressControllerWarning = ""
	)

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		generatedConfig, err = generated.LoadConfig("")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Get kubernetes client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, false)
	if err != nil {
		log.Fatal(err)
	}

	err = client.PrintWarning(generatedConfig, false, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Signal that we are working on the space if there is any
	err = cloud.ResumeSpace(client, true, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	// Get default namespace
	var devspaceConfig *latest.Config
	if configExists {
		// Get config with adjusted cluster config
		devspaceConfig = configutil.GetConfig(context.WithValue(context.Background(), constants.KubeContextKey, client.CurrentContext), "")
	}

	namespace := client.Namespace
	currentContext := client.CurrentContext

	// Retrieve space
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
	} else {
		ingressControllerWarning = ansi.Color(" ! an ingress controller must be installed in your cluster", "red+b")
	}

	log.WriteString("\n")
	openingMode := survey.Question(&survey.QuestionOptions{
		Question:     "How do you want to open your application?",
		DefaultValue: openLocalHostOption,
		Options: []string{
			openLocalHostOption,
			openDomainOption + ingressControllerWarning,
		},
	})
	log.WriteString("\n")

	// Check if we should open locally
	if openingMode == openLocalHostOption {
		openLocal(devspaceConfig, nil, client, domain)
		return
	}

	// create ingress for public access via domain
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

		if len(domains) == 1 {
			domain = domains[0]
		} else {
			domain = survey.Question(&survey.QuestionOptions{
				Question: "Please select a domain to open",
				Options:  domains,
			})
		}
	} else {
		domain = survey.Question(&survey.QuestionOptions{
			Question: "Which domain do you want to use? (must be connected via DNS)",
		})
	}

	// Check if ingress for domain already exists
	existingIngressDomain, existingIngressTLS, err := findDomain(client, namespace, domain)
	if err != nil {
		log.Fatal(err)
	}

	// No suitable ingress found => create ingress
	if existingIngressDomain == "" {
		serviceName, servicePort, _, err := getService(devspaceConfig, client, namespace, domain, false)
		if err != nil {
			log.Fatalf("Error getting service: %v", err)
		}

		hash := sha1.New()
		hash.Write([]byte(domain))

		ingressName := "devspace-ingress-" + fmt.Sprintf("%x", hash.Sum(nil))
		_, err = client.Client.ExtensionsV1beta1().Ingresses(namespace).Create(&v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName},
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					v1beta1.IngressRule{
						Host: domain,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									v1beta1.HTTPIngressPath{
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceName,
											ServicePort: intstr.FromInt(servicePort),
										},
									},
								},
							},
						},
					},
				},
				TLS: []v1beta1.IngressTLS{
					v1beta1.IngressTLS{
						Hosts:      []string{domain},
						SecretName: "tls-" + ingressName,
					},
				},
			},
		})
		if err != nil {
			log.WriteString("\n")
			log.Fatalf("Unable to create ingress for domain %s: %v", domain, err)
		}

		domain, tls, err = findDomain(client, namespace, domain)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		domain = existingIngressDomain
		tls = existingIngressTLS
	}

	// Add schema
	if tls {
		domain = "https://" + domain
	} else {
		domain = "http://" + domain
	}

	err = openURL(domain, client, namespace, log.GetInstance(), 4*time.Minute)
	if err != nil {
		log.Fatalf("Timeout: domain %s still returns 502 code, even after several minutes. Either the app has no valid '/' route or it is listening on the wrong port: %v", domain, err)
	}
}

func openURL(url string, kubectlClient *kubectl.Client, analyzeNamespace string, log log.Logger, maxWait time.Duration) error {
	// Loop and check if http code is != 502
	log.StartWait("Waiting for ingress")
	defer log.StopWait()

	// Make sure the ingress has some time to take effect
	time.Sleep(time.Second * 2)

	now := time.Now()
	for time.Since(now) < maxWait {
		// Check if domain is ready => ignore error as we will retry request
		resp, _ := http.Get(url)
		if resp != nil && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
			log.StopWait()
			open.Start(url)
			log.Donef("Successfully opened %s", url)
			return nil
		}

		if kubectlClient != nil && analyzeNamespace != "" {
			// Analyze space for issues
			report, err := analyze.CreateReport(kubectlClient, analyzeNamespace, false)
			if err != nil {
				return errors.Errorf("Error analyzing space: %v", err)
			}
			if len(report) > 0 {
				reportString := analyze.ReportToString(report)
				log.WriteString(reportString)
			}
		}

		time.Sleep(time.Second * 3)
	}
	return nil
}

func openLocal(devspaceConfig *latest.Config, generatedConfig *generated.Config, client *kubectl.Client, domain string) {
	_, servicePort, serviceLabels, err := getService(devspaceConfig, client, client.Namespace, domain, true)
	if err != nil {
		log.Fatal("Unable to get service: %v", err)
	}

	localPort := servicePort

	if localPort < 1024 {
		localPort = localPort + 8000
	}

	domain = "http://localhost:" + strconv.Itoa(localPort)

	portMappings := []*latest.PortMapping{
		&latest.PortMapping{
			LocalPort:  &localPort,
			RemotePort: &servicePort,
		},
	}

	labelSelector := map[string]string{}
	for key, value := range *serviceLabels {
		labelSelector[key] = value
	}

	portforwardingConfig := []*latest.PortForwardingConfig{
		&latest.PortForwardingConfig{
			PortMappings:  portMappings,
			LabelSelector: labelSelector,
		},
	}

	// start port-forwarding for localhost access
	portForwarder, err := services.StartPortForwarding(&latest.Config{
		Dev: &latest.DevConfig{
			Ports: portforwardingConfig,
		},
	}, generatedConfig, client, log.GetInstance())
	if err != nil {
		log.Fatalf("Unable to start portforwarding: %v", err)
	}

	defer func() {
		for _, v := range portForwarder {
			v.Close()
		}
	}()

	// Loop and check if http code is != 502
	log.StartWait("Waiting for application")
	defer log.StopWait()

	// Make sure the ingress has some time to take effect
	time.Sleep(time.Second * 2)

	log.StopWait()
	open.Start(domain)
	log.Donef("Successfully opened %s", domain)
	log.WriteString("\n")
	log.Info("Press ENTER to terminate port-forwarding process")

	// Wait until user aborts the program or presses ENTER
	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadRune()
	return
}

func getService(config *latest.Config, client *kubectl.Client, namespace, host string, getEndpoints bool) (string, int, *map[string]string, error) {
	// Let user select service
	serviceNameList := []string{}
	serviceLabels := map[string]map[string]string{}

	serviceList, err := client.Client.CoreV1().Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", 0, nil, errors.Wrap(err, "list services")
	}

	for _, service := range serviceList.Items {
		// We skip tiller-deploy, because usually you don't want to create an ingress for tiller
		if service.Name == "tiller-deploy" {
			continue
		}

		if service.Spec.Type == v1.ServiceTypeClusterIP {
			if service.Spec.ClusterIP == "None" {
				continue
			}

			for _, ports := range service.Spec.Ports {
				port := ports.Port

				if getEndpoints {
					port = ports.TargetPort.IntVal
				}
				serviceNameList = append(serviceNameList, service.Name+":"+strconv.Itoa(int(port)))
			}

			if getEndpoints {
				serviceLabels[service.Name] = service.Spec.Selector
			} else {
				serviceLabels[service.Name] = service.Labels
			}
		}
	}

	serviceName := ""
	servicePort := ""

	if len(serviceNameList) == 0 {
		return "", 0, nil, fmt.Errorf("Couldn't find any active services an ingress could connect to. Please make sure you have a service for your application")
	} else if len(serviceNameList) == 1 {
		splitted := strings.Split(serviceNameList[0], ":")

		serviceName = splitted[0]
		servicePort = splitted[1]
	} else {
		// Ask user which service
		splitted := strings.Split(survey.Question(&survey.QuestionOptions{
			Question: fmt.Sprintf("Please specify the service you want to make available on '%s'", ansi.Color(host, "white+b")),
			Options:  serviceNameList,
		}), ":")

		serviceName = splitted[0]
		servicePort = splitted[1]
	}
	servicePortInt, err := strconv.Atoi(servicePort)
	if err != nil {
		return "", 0, nil, err
	}

	labels := serviceLabels[serviceName]

	return serviceName, servicePortInt, &labels, nil
}

func findDomain(client *kubectl.Client, namespace, host string) (string, bool, error) {
	log.StartWait("Retrieve ingresses")
	defer log.StopWait()

	// List all ingresses and only create one if there is none already
	ingressList, err := client.Client.ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{})
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
