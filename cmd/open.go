package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/resume"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"

	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/port"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	openLocalHostOption           = "via localhost (provides private access only on your computer via port-forwarding)"
	openDomainOption              = "via domain (makes your application publicly available via ingress)"
	allowedIngressHostsAnnotation = "devspace.cloud/allowed-hosts"
)

// OpenCmd holds the open cmd flags
type OpenCmd struct {
	*flags.GlobalFlags

	Provider string
	log      log.Logger
}

// NewOpenCmd creates a new open command
func NewOpenCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &OpenCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

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
		RunE: cmd.RunOpen,
	}

	openCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return openCmd
}

// RunOpen executes the functionality "devspace open"
func (cmd *OpenCmd) RunOpen(cobraCmd *cobra.Command, args []string) error {
	// Set config root
	configLoader := loader.NewConfigLoader(cmd.ToConfigOptions(), cmd.log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}

	var (
		providerName             string
		provider                 cloud.Provider
		space                    *cloudlatest.Space
		domain                   string
		tls                      bool
		ingressControllerWarning = ""
	)

	// Load generated config if possible
	var generatedConfig *generated.Config
	if configExists {
		log.StartFileLogging()

		generatedConfig, err = configLoader.Generated()
		if err != nil {
			return err
		}
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	// Get kubernetes client
	client, err := kubectl.NewClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return err
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = resume.NewSpaceResumer(client, cmd.log).ResumeSpace(true)
	if err != nil {
		return err
	}

	// Get default namespace
	var devspaceConfig *latest.Config
	if configExists {
		// Get config with adjusted cluster config
		devspaceConfig, err = configLoader.Load()
		if err != nil {
			return err
		}
	}

	namespace := client.Namespace()
	currentContext := client.CurrentContext()

	// Retrieve space
	spaceID, currentContextProvider, err := kubeconfig.GetSpaceID(currentContext)
	if err == nil { // Current kube-context is a Space
		if providerName == "" {
			providerName = currentContextProvider
		}

		// Get provider
		provider, err = cloud.GetProvider(providerName, cmd.log)
		if err != nil {
			return err
		}

		// Get space
		space, err = provider.Client().GetSpace(spaceID)
		if err != nil {
			return err
		}
	} else {
		ingressControllerWarning = ansi.Color(" ! an ingress controller must be installed in your cluster", "red+b")
	}

	openingMode, err := cmd.log.Question(&survey.QuestionOptions{
		Question:     "How do you want to open your application?",
		DefaultValue: openLocalHostOption,
		Options: []string{
			openLocalHostOption,
			openDomainOption + ingressControllerWarning,
		},
	})
	if err != nil {
		return err
	}
	cmd.log.WriteString("\n")

	// Check if we should open locally
	if openingMode == openLocalHostOption {
		cmd.openLocal(devspaceConfig, nil, client, domain)
		return nil
	}

	// create ingress for public access via domain
	if space != nil {
		namespace, err := client.KubeClient().CoreV1().Namespaces().Get(space.Namespace, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "get space namespace")
		}

		// Check if domain there is a domain for the space
		if namespace.Annotations == nil || namespace.Annotations[allowedIngressHostsAnnotation] == "" {
			return errors.Errorf("Space %s has no allowed domains", space.Name)
		}

		// Select domain
		domains := strings.Split(namespace.Annotations[allowedIngressHostsAnnotation], ",")
		if len(domains) == 1 {
			domain = domains[0]
		} else {
			domain, err = cmd.log.Question(&survey.QuestionOptions{
				Question: "Please select a domain to open",
				Options:  domains,
			})
			if err != nil {
				return err
			}
		}

		// Check if domain has wildcard
		if strings.Index(domain, "*") != -1 {
			replaceValue, err := cmd.log.Question(&survey.QuestionOptions{
				Question: fmt.Sprintf("Please enter a value for wildcard in domain '%s'", domain),
			})
			if err != nil {
				return err
			}

			domain = strings.Replace(domain, "*", replaceValue, -1)
		}
	} else {
		domain, err = cmd.log.Question(&survey.QuestionOptions{
			Question: "Which domain do you want to use? (must be connected via DNS)",
		})
		if err != nil {
			return err
		}
	}

	// Check if ingress for domain already exists
	existingIngressDomain, existingIngressTLS, err := cmd.findDomain(client, namespace, domain)
	if err != nil {
		return err
	}

	// No suitable ingress found => create ingress
	if existingIngressDomain == "" {
		serviceName, servicePort, _, err := cmd.getService(devspaceConfig, client, namespace, domain, false)
		if err != nil {
			return errors.Wrap(err, "get service")
		}

		domainHash := hash.String(domain)

		ingressName := "devspace-ingress-" + domainHash[:10]
		_, err = client.KubeClient().ExtensionsV1beta1().Ingresses(namespace).Create(&v1beta1.Ingress{
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
			},
		})
		if err != nil {
			cmd.log.WriteString("\n")
			return errors.Errorf("Unable to create ingress for domain %s: %v", domain, err)
		}

		domain, tls, err = cmd.findDomain(client, namespace, domain)
		if err != nil {
			return err
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

	err = openURL(domain, client, namespace, cmd.log, 4*time.Minute)
	if err != nil {
		return errors.Errorf("Timeout: domain %s still returns 502 code, even after several minutes. Either the app has no valid '/' route or it is listening on the wrong port: %v", domain, err)
	}

	return nil
}

func openURL(url string, kubectlClient kubectl.Client, analyzeNamespace string, log log.Logger, maxWait time.Duration) error {
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
			report, err := analyze.NewAnalyzer(kubectlClient, log).CreateReport(analyzeNamespace, false)
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

func (cmd *OpenCmd) openLocal(devspaceConfig *latest.Config, generatedConfig *generated.Config, client kubectl.Client, domain string) error {
	_, servicePort, serviceLabels, err := cmd.getService(devspaceConfig, client, client.Namespace(), domain, true)
	if err != nil {
		return errors.Errorf("Unable to get service: %v", err)
	}

	localPort := servicePort
	if localPort < 1024 {
		localPort = localPort + 8000
	}

	// Check if port is open
	portOpen, _ := port.Check(localPort)
	for portOpen == false {
		localPort++
		portOpen, _ = port.Check(localPort)
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
	servicesClient := services.NewClient(&latest.Config{
		Dev: &latest.DevConfig{
			Ports: portforwardingConfig,
		},
	}, generatedConfig, client, nil, cmd.log)
	err = servicesClient.StartPortForwarding()
	if err != nil {
		return errors.Wrap(err, "start port forwarding")
	}

	// Loop and check if http code is != 502
	cmd.log.StartWait("Waiting for application")
	defer cmd.log.StopWait()

	// Make sure the ingress has some time to take effect
	time.Sleep(time.Second * 2)

	cmd.log.StopWait()
	open.Start(domain)
	cmd.log.Donef("Successfully opened %s", domain)
	cmd.log.WriteString("\n")
	cmd.log.Info("Press ENTER to terminate port-forwarding process")

	// Wait until user aborts the program or presses ENTER
	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadRune()
	return nil
}

func (cmd *OpenCmd) getService(config *latest.Config, client kubectl.Client, namespace, host string, getEndpoints bool) (string, int, *map[string]string, error) {
	// Let user select service
	serviceNameList := []string{}
	serviceLabels := map[string]map[string]string{}

	serviceList, err := client.KubeClient().CoreV1().Services(namespace).List(metav1.ListOptions{})
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
		return "", 0, nil, errors.Errorf(message.ServiceNotFound, namespace)
	} else if len(serviceNameList) == 1 {
		splitted := strings.Split(serviceNameList[0], ":")

		serviceName = splitted[0]
		servicePort = splitted[1]
	} else {
		servicePickerQuestion := "Select the service you want to open:"
		if host != "" {
			servicePickerQuestion = fmt.Sprintf("Select the service you want to make available on '%s':", ansi.Color(host, "white+b"))
		}

		// Ask user which service
		service, err := cmd.log.Question(&survey.QuestionOptions{
			Question: servicePickerQuestion,
			Options:  serviceNameList,
		})
		if err != nil {
			return "", 0, nil, err
		}

		splitted := strings.Split(service, ":")

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

func (cmd *OpenCmd) findDomain(client kubectl.Client, namespace, host string) (string, bool, error) {
	cmd.log.StartWait("Retrieve ingresses")
	defer cmd.log.StopWait()

	// List all ingresses and only create one if there is none already
	ingressList, err := client.KubeClient().ExtensionsV1beta1().Ingresses(namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", false, errors.Errorf("Error listing ingresses: %v", err)
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
