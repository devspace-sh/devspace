package cloud

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/encryption"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterNameValidationRegEx is the cluster name validation regex
var ClusterNameValidationRegEx = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$")

const (
	// DevSpaceCloudNamespace is the namespace to create
	DevSpaceCloudNamespace = "devspace-cloud"

	// DevSpaceServiceAccount is the service account to create
	DevSpaceServiceAccount = "devspace-cloud-user"

	// DevSpaceClusterRoleBinding is the name of the clusterrolebinding to create for the service account
	DevSpaceClusterRoleBinding = "devspace-cloud-user-binding"

	loadBalancerOption = "LoadBalancer (GKE, AKS, EKS etc.)"
	hostNetworkOption  = "Use host network"
)

// ConnectClusterOptions holds the options for connecting a cluster
type ConnectClusterOptions struct {
	DeployAdmissionController bool
	DeployGatekeeper          bool
	DeployGatekeeperRules     bool
	DeployIngressController   bool
	DeployCertManager         bool

	UseHostNetwork *bool

	ClusterName string
	KubeContext string
	Key         string

	OpenUI bool
	Public bool

	UseDomain bool
	Domain    string
}

type clusterResources struct {
	PodPolicy     bool
	NetworkPolicy bool
	CertManager   bool
}

// ConnectCluster connects a new cluster to DevSpace Cloud
func (p *provider) ConnectCluster(options *ConnectClusterOptions) error {
	var (
		client kubectl.Client
	)

	// Get cluster name
	clusterName, err := p.getClusterName(options.ClusterName)
	if err != nil {
		return err
	}

	// Check what kube context to use
	if options.KubeContext == "" {
		allowLocalClusters := true
		if p.Name == config.DevSpaceCloudProviderName {
			allowLocalClusters = false
		}

		// Get kube context to use
		client, err = kubectl.NewClientBySelect(allowLocalClusters, true, p.log)
		if err != nil {
			return errors.Wrap(err, "new kubectl client")
		}
	} else {
		// Get kube context to use
		client, err = kubectl.NewClientFromContext(options.KubeContext, "", false)
		if err != nil {
			return errors.Wrap(err, "new kubectl client")
		}
	}

	// Check available cluster resources
	availableResources, err := p.checkResources(client)
	if err != nil {
		return errors.Wrap(err, "check resource availability")
	}

	// Initialize namespace
	err = p.initializeNamespace(client.KubeClient())
	if err != nil {
		return errors.Wrap(err, "init namespace")
	}

	token, caCert, err := p.getServiceAccountCredentials(client)
	if err != nil {
		return errors.Wrap(err, "get service account credentials")
	}

	needKey, err := p.needKey()
	if err != nil {
		return errors.Wrap(err, "check cloud settings")
	}
	if options.Public {
		needKey = false
	}

	// Encrypt token if needed
	encryptedToken := token
	if needKey {
		if options.Key == "" {
			options.Key, err = p.getKey(false)
			if err != nil {
				return errors.Wrap(err, "get key")
			}
		}

		encryptedToken, err = encryption.EncryptAES([]byte(options.Key), token)
		if err != nil {
			return errors.Wrap(err, "encrypt token")
		}

		encryptedToken = []byte(base64.StdEncoding.EncodeToString(encryptedToken))
	}

	// Create cluster remotely
	p.log.StartWait("Initialize cluster")
	defer p.log.StopWait()
	var clusterID int
	if options.Public {
		clusterID, err = p.client.CreatePublicCluster(clusterName, client.RestConfig().Host, caCert, string(encryptedToken))
		if err != nil {
			return errors.Wrap(err, "create cluster")
		}
	} else {
		clusterID, err = p.client.CreateUserCluster(clusterName, client.RestConfig().Host, caCert, string(encryptedToken), availableResources.NetworkPolicy)
		if err != nil {
			return errors.Wrap(err, "create cluster")
		}
	}
	p.log.StopWait()

	// Save key
	if needKey {
		p.ClusterKey[clusterID] = options.Key
		err = p.Save()
		if err != nil {
			return errors.Wrap(err, "save key")
		}
	}
	p.log.StartWait("Initializing Cluster")
	defer p.log.StopWait()

	// Initialize roles and pod security policies
	err = p.client.InitCore(clusterID, options.Key, availableResources.PodPolicy)
	if err != nil {
		// Try to delete cluster if core initialization has failed
		p.log.StartWait("Error happened. Rolling back")
		p.client.DeleteCluster(&latest.Cluster{ClusterID: clusterID}, options.Key, false, false)

		return errors.Wrap(err, "initialize core")
	}
	p.log.Done("Initialized cluster")

	// Deploy admission controller, ingress controller and cert manager
	err = p.deployServices(client, clusterID, availableResources, options)
	if err != nil {
		return err
	}

	// Set space domain
	if options.UseDomain {
		// Set cluster domain to use for spaces
		err = p.specifyDomain(clusterID, options)
		if err != nil {
			return err
		}
	} else if options.DeployIngressController {
		err = p.defaultClusterSpaceDomain(client, *options.UseHostNetwork, clusterID, options.Key)
		if err != nil {
			p.log.Warnf("Couldn't configure default cluster space domain: %v", err)
		}
	}

	// Open ui
	if options.OpenUI {
		url := fmt.Sprintf("%s/clusters/%d/overview", p.Host, clusterID)
		err = p.browser.Start(url)
		if err != nil {
			p.log.Warnf("Couldn't open the url '%s': %v", url, err)
		}
	}

	return nil
}

var waitTimeout = time.Minute * 5

func (p *provider) defaultClusterSpaceDomain(client kubectl.Client, useHostNetwork bool, clusterID int, key string) error {
	if useHostNetwork {
		p.log.StartWait("Waiting for loadbalancer to get an IP address")
		defer p.log.StopWait()

		nodeList, err := client.KubeClient().CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "list nodes")
		}
		if len(nodeList.Items) == 0 {
			return errors.New("Couldn't find a node in cluster")
		}

		ip := ""
		for _, node := range nodeList.Items {
			for _, address := range node.Status.Addresses {
				if address.Type == v1.NodeExternalIP && address.Address != "" {
					ip = address.Address
					break
				}
			}

			if ip != "" {
				break
			}
		}
		if ip == "" {
			return errors.New("Couldn't find a node with a valid external IP address in cluster, make sure your nodes are accessable from the outside")
		}
	} else {
		p.log.StartWait("Waiting for loadbalancer to get an IP address. This may take several minutes")
		defer p.log.StopWait()

		now := time.Now()

	Outer:
		for time.Since(now) < waitTimeout {
			// Get loadbalancer
			services, err := client.KubeClient().CoreV1().Services(constants.DevSpaceCloudNamespace).List(metav1.ListOptions{})
			if err != nil {
				return errors.Wrap(err, "list services")
			}

			// Check loadbalancer for an ip
			for _, service := range services.Items {
				if service.Spec.Type == v1.ServiceTypeLoadBalancer {
					for _, ingress := range service.Status.LoadBalancer.Ingress {
						if ingress.Hostname != "" {
							break Outer
						}
						if ingress.IP != "" {
							break Outer
						}
					}
				}
			}

			time.Sleep(5 * time.Second)
		}

		if time.Since(now) >= waitTimeout {
			return errors.New("Loadbalancer didn't receive a valid IP address in time. Skipping configuration of default domain for space subdomains")
		}
	}

	useDefaultClusterDomain, err := p.client.UseDefaultClusterDomain(clusterID, key)
	if err != nil {
		return errors.Wrap(err, "graphql client")
	}
	if useDefaultClusterDomain != "" {
		p.log.Donef("The domain '%s' has been successfully configured for your clusters spaces and now points to your clusters ingress controller. The dns change however can take several minutes to take affect", ansi.Color("*."+useDefaultClusterDomain, "white+b"))
	}

	return nil
}

func (p *provider) specifyDomain(clusterID int, options *ConnectClusterOptions) error {
	if options.Domain == "" {
		var err error

		options.Domain, err = p.log.Question(&survey.QuestionOptions{
			Question:               "DevSpace will automatically create an ingress for each space, which base domain do you want to use for the created spaces? (e.g. users.test.com)",
			ValidationRegexPattern: "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$",
			ValidationMessage:      "Please enter a valid hostname (e.g. users.my-domain.com)",
		})
		if err != nil {
			return err
		}
	}

	p.log.StartWait("Updating domain name")
	defer p.log.StopWait()

	err := p.client.UpdateClusterDomain(clusterID, options.Domain)
	if err != nil {
		return errors.Wrap(err, "graphql client")
	}

	p.log.StopWait()
	if *options.UseHostNetwork == false {
		p.log.Donef("Please create an A dns record for '*.%s' that points to external IP address of the loadbalancer service 'devspace-cloud/nginx-ingress-controller'.\n Run `%s` to view the service", options.Domain, ansi.Color("kubectl get svc nginx-ingress-controller -n devspace-cloud", "white+b"))
	} else {
		p.log.Donef("Please create an A dns record for '*.%s' that points to the external IP address of one of your cluster nodes.\n Run `%s` to view your cluster nodes and their IP adresses. \n Please make also sure the ports 80 and 443 can be accessed on these nodes from the internet", options.Domain, ansi.Color("kubectl get nodes -o wide", "white+b"))
	}

	return nil
}

func (p *provider) deployServices(client kubectl.Client, clusterID int, availableResources *clusterResources, options *ConnectClusterOptions) error {
	defer p.log.StopWait()

	// Check if devspace-cloud is deployed in the namespace
	configmaps, err := client.KubeClient().CoreV1().ConfigMaps(DevSpaceCloudNamespace).List(metav1.ListOptions{
		LabelSelector: "NAME=devspace-cloud,OWNER=TILLER,STATUS=DEPLOYED",
	})
	if err != nil {
		return errors.Wrap(err, "list configmaps")
	} else if len(configmaps.Items) != 0 {
		options.DeployIngressController = false
	}

	// Ingress controller
	if options.DeployIngressController {
		// Ask if we should use the host network
		if options.UseHostNetwork == nil {
			useHostNetwork, err := p.log.Question(&survey.QuestionOptions{
				Question:     "Should the ingress controller use a LoadBalancer or the host network?",
				DefaultValue: loadBalancerOption,
				Options: []string{
					loadBalancerOption,
					hostNetworkOption,
				},
			})
			if err != nil {
				return err
			}

			options.UseHostNetwork = ptr.Bool(useHostNetwork == hostNetworkOption)
		}

		p.log.StartWait("Deploying ingress controller")
		err = p.client.DeployIngressController(clusterID, options.Key, *options.UseHostNetwork)
		if err != nil {
			return errors.Wrap(err, "graphql client")
		}

		p.log.Done("Deployed ingress controller")
	}

	// Admission controller
	if options.DeployAdmissionController {
		p.log.StartWait("Deploying admission controller")

		err = p.client.DeployAdmissionController(clusterID, options.Key)
		if err != nil {
			return errors.Wrap(err, "graphql client")
		}

		if err != nil {
			p.log.Warnf("Error deploying admission controller: %v", err)
		} else {
			p.log.Done("Deployed admission controller")
		}
	}

	// Gatekeeper
	if options.DeployGatekeeper {
		p.log.StartWait("Deploying gatekeeper")

		err = p.client.DeployGatekeeper(clusterID, options.Key)
		if err != nil {
			p.log.Warnf("Error deploying gatekeeper: %v", err)
		} else {
			p.log.Done("Deployed gatekeeper")
		}
	}

	// Gatekeeper rules
	if options.DeployGatekeeperRules {
		p.log.StartWait("Deploying gatekeeper rules")

		// Deploy gatekeeper rules
		err := p.client.DeployGatekeeperRules(clusterID, options.Key)
		if err != nil {
			p.log.Warnf("Error deploying gatekeeper rules: %v", err)
		} else {
			p.log.Done("Deployed gatekeeper rules")
		}
	}

	// Cert manager
	if availableResources.CertManager == false && options.DeployCertManager {
		p.log.StartWait("Deploying cert manager")

		// Deploy cert manager
		err := p.client.DeployCertManager(clusterID, options.Key)
		if err != nil {
			p.log.Warnf("Error deploying cert manager: %v", err)
		} else {
			p.log.Done("Deployed cert manager")
		}
	}

	return nil
}

// SettingDefaultClusterEncryptToken is the setting name to check if we need an encryption key
const SettingDefaultClusterEncryptToken = "DEFAULT_CLUSTER_ENCRYPT_TOKEN"

func (p *provider) needKey() (bool, error) {
	p.log.StartWait("Retrieving cloud settings")
	defer p.log.StopWait()

	settings, err := p.client.Settings(SettingDefaultClusterEncryptToken)
	if err != nil {
		errors.Wrap(err, "graphql client")
	}

	// We don't need a key if not specified
	if len(settings) != 1 {
		return false, nil
	}

	return settings[0].ID == SettingDefaultClusterEncryptToken && settings[0].Value == "true", nil
}

func (p *provider) getServiceAccountCredentials(client kubectl.Client) ([]byte, string, error) {
	p.log.StartWait("Retrieving service account credentials")
	defer p.log.StopWait()

	// Create main service account
	sa, err := client.KubeClient().CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	beginTimeStamp := time.Now()
	timeout := time.Second * 90

	for len(sa.Secrets) == 0 && time.Since(beginTimeStamp) < timeout {
		time.Sleep(time.Second)

		sa, err = client.KubeClient().CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
	}

	if time.Since(beginTimeStamp) >= timeout {
		return nil, "", errors.New("ServiceAccount did not receive secret in time")
	}

	// Get secret
	secret, err := client.KubeClient().CoreV1().Secrets(DevSpaceCloudNamespace).Get(sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	return secret.Data["token"], base64.StdEncoding.EncodeToString(secret.Data["ca.crt"]), nil
}

func (p *provider) getKey(forceQuestion bool) (string, error) {
	if forceQuestion == false && len(p.ClusterKey) > 0 {
		keyMap := make(map[string]bool)
		useKey := ""

		for _, key := range p.ClusterKey {
			keyMap[key] = true
			useKey = key
		}

		if len(keyMap) == 1 {
			return useKey, nil
		}
	}

	for true {
		firstKey, err := p.log.Question(&survey.QuestionOptions{
			Question:               "Please enter a secure encryption key for your cluster credentials",
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		})
		if err != nil {
			return "", err
		}

		secondKey, err := p.log.Question(&survey.QuestionOptions{
			Question:               "Please re-enter the key",
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		})
		if err != nil {
			return "", err
		}

		if firstKey != secondKey {
			p.log.Info("Keys do not match! Please reenter")
			continue
		}

		hashedKey, err := hash.Password(firstKey)
		if err != nil {
			return "", errors.Wrap(err, "hash key")
		}

		return hashedKey, nil
	}

	// We never reach that point
	return "", nil
}

func (p *provider) getClusterName(clusterName string) (string, error) {
	if clusterName != "" && ClusterNameValidationRegEx.MatchString(clusterName) == false {
		return "", errors.Errorf("Cluster name %s can only contain letters, numbers and dashes (-)", clusterName)
	} else if clusterName != "" {
		return clusterName, nil
	}

	// Ask for cluster name
	for true {
		clusterName, err := p.log.Question(&survey.QuestionOptions{
			Question:     "Please enter a cluster name (e.g. my-cluster)",
			DefaultValue: "my-cluster",
		})
		if err != nil {
			return "", err
		}

		if ClusterNameValidationRegEx.MatchString(clusterName) == false {
			p.log.Infof("Cluster name %s can only contain letters, numbers and dashes (-)", clusterName)
			continue
		}

		return clusterName, nil
	}

	return "", errors.New("We should never reach this point")
}

// This function checks the available resource on the api server
// Required checks
// 	rbac.authorization.k8s.io/v1beta1
// Feature checks
// 	certmanager.k8s.io/v1alpha1
// 	networking.k8s.io/v1 networkpolicies
// 	extensions/v1beta1 podsecuritypolicies
func (p *provider) checkResources(client kubectl.Client) (*clusterResources, error) {
	p.log.StartWait("Checking cluster resources")
	defer p.log.StopWait()

	// Check if cluster has active nodes
	nodeList, err := client.KubeClient().CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list cluster nodes")
	}
	if len(nodeList.Items) == 0 {
		return nil, errors.Errorf("The cluster specified has no nodes, please choose a cluster where at least one node is up and running")
	}

	groupResources, err := client.KubeClient().Discovery().ServerResources()
	if err != nil {
		return nil, errors.Wrap(err, "discover server resources")
	}

	exist := kubectl.GroupVersionExist("rbac.authorization.k8s.io/v1beta1", groupResources)
	if exist == false {
		return nil, errors.New("Group version rbac.authorization.k8s.io/v1beta1 does not exist in cluster, but is required. Is RBAC enabled?")
	}

	return &clusterResources{
		PodPolicy:     kubectl.ResourceExist("extensions/v1beta1", "podsecuritypolicies", groupResources),
		NetworkPolicy: kubectl.ResourceExist("networking.k8s.io/v1", "networkpolicies", groupResources),
		CertManager:   kubectl.GroupVersionExist("certmanager.k8s.io/v1alpha1", groupResources),
	}, nil
}

func (p *provider) initializeNamespace(client kubernetes.Interface) error {
	p.log.StartWait("Initializing namespace")
	defer p.log.StopWait()

	// Create devspace-cloud namespace
	_, err := client.CoreV1().Namespaces().Get(DevSpaceCloudNamespace, metav1.GetOptions{})
	if err != nil {
		_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: DevSpaceCloudNamespace,
				Labels: map[string]string{
					"devspace.cloud/control-plane": "true",
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "create namespace")
		}

		p.log.Donef("Created namespace '%s'", DevSpaceCloudNamespace)
	}

	// Create serviceaccount
	_, err = client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
	if err != nil {
		_, err = client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Create(&v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: DevSpaceServiceAccount,
			},
		})
		if err != nil {
			return errors.Wrap(err, "create service account")
		}

		p.log.Donef("Created service account '%s'", DevSpaceServiceAccount)
	}

	// Create cluster-admin clusterrole binding
	_, err = client.RbacV1().ClusterRoleBindings().Get(DevSpaceClusterRoleBinding, metav1.GetOptions{})
	if err != nil {
		_, err = client.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: DevSpaceClusterRoleBinding,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      DevSpaceServiceAccount,
					Namespace: DevSpaceCloudNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
		})
		if err != nil {
			return errors.Wrap(err, "create cluster role binding")
		}

		p.log.Infof("Created cluster role binding '%s'", DevSpaceClusterRoleBinding)
	}

	return nil
}

// ResetKey resets a cluster key
func (p *provider) ResetKey(clusterName string) error {
	cluster, err := p.client.GetClusterByName(clusterName)
	if err != nil {
		return errors.Wrap(err, "get cluster")
	}
	clusterUser, err := p.client.GetClusterUser(cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "get cluster user")
	}

	// Get kube context to use
	client, err := kubectl.NewClientBySelect(false, false, p.log)
	if err != nil {
		return err
	}
	if client.RestConfig().Host != *cluster.Server {
		return errors.Errorf("Selected context does not point to the correct host. Selected %s <> %s", client.RestConfig().Host, *cluster.Server)
	}

	key, err := p.getKey(true)
	if err != nil {
		return errors.Wrap(err, "get key")
	}

	token, _, err := p.getServiceAccountCredentials(client)
	if err != nil {
		return errors.Wrap(err, "get service account credentials")
	}

	encryptedToken, err := encryption.EncryptAES([]byte(key), token)
	if err != nil {
		return errors.Wrap(err, "encrypt token")
	}

	// Update token
	p.log.StartWait("Update token")
	defer p.log.StopWait()

	// Do the request
	err = p.client.UpdateUserClusterUser(clusterUser.ClusterUserID, encryptedToken)
	if err != nil {
		return errors.Wrap(err, "update token")
	}

	// Save key
	p.ClusterKey[cluster.ClusterID] = key
	err = p.Save()
	if err != nil {
		return errors.Wrap(err, "save key")
	}

	return nil
}
