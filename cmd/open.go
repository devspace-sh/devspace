package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/loft-sh/devspace/helper/util/port"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/services/portforwarding"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/analyze"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	openLocalHostOption = "via localhost (provides private access only on your computer via port-forwarding)"
	openDomainOption    = "via domain (makes your application publicly available via ingress)"
)

// OpenCmd holds the open cmd flags
type OpenCmd struct {
	*flags.GlobalFlags

	Provider string
	Port     int

	log log.Logger
}

// NewOpenCmd creates a new open command
func NewOpenCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			plugin.SetPluginCommand(cobraCmd, args)
			return cmd.RunOpen(f)
		},
	}

	openCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")
	openCmd.Flags().IntVar(&cmd.Port, "port", 0, "The port on the localhost to listen on")

	return openCmd
}

// RunOpen executes the functionality "devspace open"
func (cmd *OpenCmd) RunOpen(f factory.Factory) error {
	// Set config root
	cmd.log = f.GetLog()
	configLoader, err := f.NewConfigLoader(cmd.ConfigPath)
	if err != nil {
		return err
	}
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}

	var (
		domain                   string
		tls                      bool
		ingressControllerWarning string
	)

	// Get kubernetes client
	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace)
	if err != nil {
		return err
	}

	// Load generated config if possible
	var localCache localcache.Cache
	if configExists {
		log.StartFileLogging()

		localCache, err = configLoader.LoadLocalCache()
		if err != nil {
			return err
		}
	}

	// If the current kube context or namespace is different from old,
	// show warnings and reset kube client if necessary
	client, err = kubectl.CheckKubeContext(client, localCache, cmd.NoWarn, cmd.SwitchContext, false, cmd.log)
	if err != nil {
		return err
	}

	// create devspace context
	ctx := devspacecontext.NewContext(context.Background(), nil, cmd.log).
		WithKubeClient(client)

	// Execute plugin hook
	err = hook.ExecuteHooks(ctx, nil, "open")
	if err != nil {
		return err
	}

	namespace := client.Namespace()
	ingressControllerWarning = ansi.Color(" ! an ingress controller must be installed in your cluster", "red+b")

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
	cmd.log.WriteString(logrus.InfoLevel, "\n")

	// Check if we should open locally
	if openingMode == openLocalHostOption {
		err := cmd.openLocal(ctx, domain)
		if err != nil {
			cmd.log.Error(err)
		}
		return nil
	}

	// create ingress for public access via domain
	domain, err = cmd.log.Question(&survey.QuestionOptions{
		Question: "Which domain do you want to use? (must be connected via DNS)",
	})
	if err != nil {
		return err
	}

	// Check if ingress for domain already exists
	existingIngressDomain, existingIngressTLS, err := cmd.findDomain(client, namespace, domain)
	if err != nil {
		return err
	}

	// No suitable ingress found => create ingress
	if existingIngressDomain == "" {
		serviceName, servicePort, _, err := cmd.getService(client, namespace, domain, false)
		if err != nil {
			return errors.Wrap(err, "get service")
		}

		domainHash := hash.String(domain)

		ingressName := "devspace-ingress-" + domainHash[:10]
		_, err = client.KubeClient().NetworkingV1().Ingresses(namespace).Create(context.TODO(), &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: ingressName},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						Host: domain,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: serviceName,
												Port: networkingv1.ServiceBackendPort{
													Number: int32(servicePort),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			cmd.log.WriteString(logrus.InfoLevel, "\n")
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
	log.Info("Waiting for ingress...")

	// Make sure the ingress has some time to take effect
	time.Sleep(time.Second * 5)

	now := time.Now()
	for time.Since(now) < maxWait {
		// Check if domain is ready => ignore error as we will retry request
		resp, _ := http.Get(url)
		if resp != nil && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
			time.Sleep(time.Second * 1)
			_ = open.Start(url)
			log.Donef("Successfully opened %s", url)
			return nil
		}

		if kubectlClient != nil && analyzeNamespace != "" {
			// Analyze space for issues
			report, err := analyze.NewAnalyzer(kubectlClient, log).CreateReport(analyzeNamespace, analyze.Options{Wait: true})
			if err != nil {
				return errors.Errorf("Error analyzing space: %v", err)
			}
			if len(report) > 0 {
				reportString := analyze.ReportToString(report)
				log.WriteString(logrus.InfoLevel, reportString)
			}
		}

		time.Sleep(time.Second * 3)
	}
	return nil
}

func (cmd *OpenCmd) openLocal(ctx devspacecontext.Context, domain string) error {
	_, servicePort, serviceLabels, err := cmd.getService(ctx.KubeClient(), ctx.KubeClient().Namespace(), domain, true)
	if err != nil {
		return errors.Errorf("Unable to get service: %v", err)
	}

	localPort := servicePort
	if cmd.Port != 0 {
		localPort = cmd.Port
	} else {
		if localPort < 1024 {
			localPort = localPort + 8000
		}

		// Check if port is open
		portOpen, _ := port.IsAvailable(fmt.Sprintf(":%d", localPort))
		for i := 0; i < 10 && !portOpen; i++ {
			localPort++
			portOpen, _ = port.IsAvailable(fmt.Sprintf(":%d", localPort))
		}
	}

	domain = "http://localhost:" + strconv.Itoa(localPort)
	portMappings := []*latest.PortMapping{
		{
			Port: fmt.Sprintf("%d:%d", localPort, servicePort),
		},
	}

	labelSelector := map[string]string{}
	for key, value := range *serviceLabels {
		labelSelector[key] = value
	}

	devPod := &latest.DevPod{
		Name:          "open",
		LabelSelector: labelSelector,
		Ports:         portMappings,
	}
	fakeConfig := &latest.Config{
		Dev: map[string]*latest.DevPod{
			"open": devPod,
		},
	}

	devSpaceConfig := config.Ensure(config.NewConfig(nil, nil, fakeConfig, nil, nil, nil, constants.DefaultConfigPath))
	ctx = ctx.WithConfig(devSpaceConfig)
	options := targetselector.NewEmptyOptions().
		WithLabelSelector(labels.Set(labelSelector).String()).
		WithWaitingStrategy(targetselector.NewUntilNewestRunningWaitingStrategy(time.Second))

	// start port-forwarding for localhost access
	ctx, t := ctx.WithNewTomb()
	<-t.NotifyGo(func() error {
		return portforwarding.StartPortForwarding(ctx, devPod, targetselector.NewTargetSelector(options), t)
	})
	if !t.Alive() {
		return t.Err()
	}

	// Loop and check if http code is != 502
	cmd.log.Info("Waiting for application...")

	// Make sure the ingress has some time to take effect
	time.Sleep(time.Second * 2)

	_ = open.Start(domain)
	cmd.log.Donef("Successfully opened %s", domain)
	cmd.log.WriteString(logrus.InfoLevel, "\n")
	cmd.log.Info("Press ENTER to terminate port-forwarding process")

	// Wait until user aborts the program or presses ENTER
	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadRune()
	return nil
}

func (cmd *OpenCmd) getService(client kubectl.Client, namespace, host string, getEndpoints bool) (string, int, *map[string]string, error) {
	// Let user select service
	serviceNameList := []string{}
	serviceLabels := map[string]map[string]string{}

	serviceList, err := client.KubeClient().CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", 0, nil, errors.Wrap(err, "list services")
	}

	for _, service := range serviceList.Items {
		// We skip tiller-deploy, because usually you don't want to create an ingress for tiller
		if service.Name == "tiller-deploy" {
			continue
		}

		if service.Spec.Type == v1.ServiceTypeClusterIP || service.Spec.Type == v1.ServiceTypeLoadBalancer {
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
	cmd.log.Info("Retrieve ingresses...")

	// List all ingresses and only create one if there is none already
	ingressList, err := client.KubeClient().NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
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
