package cloud

import (
	"encoding/base64"
	"regexp"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/rbac/v1beta1"
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
	DeployIngressController   bool
	DeployCertManager         bool

	UseHostNetwork *bool

	ClusterName string
	KubeContext string
	Key         string

	UseDomain bool
	Domain    string
}

type clusterResources struct {
	PodPolicy     bool
	NetworkPolicy bool
	CertManager   bool
}

// ConnectCluster connects a new cluster to DevSpace Cloud
func (p *Provider) ConnectCluster(options *ConnectClusterOptions, log log.Logger) error {
	var (
		client *kubectl.Client
	)

	// Get cluster name
	clusterName, err := getClusterName(options.ClusterName, log)
	if err != nil {
		return err
	}

	// Check what kube context to use
	if options.KubeContext == "" {
		// Get kube context to use
		client, err = kubectl.NewClientBySelect(false, true, log)
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
	availableResources, err := checkResources(client.Client)
	if err != nil {
		return errors.Wrap(err, "check resource availability")
	}

	// Initialize namespace
	err = initializeNamespace(client.Client)
	if err != nil {
		return errors.Wrap(err, "init namespace")
	}

	token, caCert, err := getServiceAccountCredentials(client)
	if err != nil {
		return errors.Wrap(err, "get service account credentials")
	}

	needKey, err := p.needKey()
	if err != nil {
		return errors.Wrap(err, "check cloud settings")
	}

	// Encrypt token if needed
	encryptedToken := token
	if needKey {
		if options.Key == "" {
			options.Key, err = getKey(p, false, log)
			if err != nil {
				return errors.Wrap(err, "get key")
			}
		}

		encryptedToken, err = EncryptAES([]byte(options.Key), token)
		if err != nil {
			return errors.Wrap(err, "encrypt token")
		}

		encryptedToken = []byte(base64.StdEncoding.EncodeToString(encryptedToken))
	}

	// Create cluster remotely
	log.StartWait("Initialize cluster")
	defer log.StopWait()
	clusterID, err := p.CreateUserCluster(clusterName, client.RestConfig.Host, caCert, string(encryptedToken), availableResources.NetworkPolicy)
	if err != nil {
		return errors.Wrap(err, "create cluster")
	}
	log.StopWait()

	// Save key
	if needKey {
		p.ClusterKey[clusterID] = options.Key
		err = p.Save()
		if err != nil {
			return errors.Wrap(err, "save key")
		}
	}

	// Initialize roles and pod security policies
	err = p.initCore(clusterID, options.Key, availableResources.PodPolicy)
	if err != nil {
		// Try to delete cluster if core initialization has failed
		deleteCluster(p, clusterID, options.Key)

		return errors.Wrap(err, "initialize core")
	}

	// Deploy admission controller, ingress controller and cert manager
	err = p.deployServices(client, clusterID, availableResources, options, log)
	if err != nil {
		return err
	}

	// Set space domain
	if options.UseDomain {
		// Set cluster domain to use for spaces
		err = p.specifyDomain(clusterID, options, log)
		if err != nil {
			return err
		}
	} else if options.DeployIngressController {
		err = defaultClusterSpaceDomain(p, client, *options.UseHostNetwork, clusterID, options.Key)
		if err != nil {
			log.Warnf("Couldn't configure default cluster space domain: %v", err)
		}
	}

	return nil
}

var waitTimeout = time.Minute * 5

func defaultClusterSpaceDomain(p *Provider, client *kubectl.Client, useHostNetwork bool, clusterID int, key string) error {
	if useHostNetwork {
		log.StartWait("Waiting for loadbalancer to get an IP address")
		defer log.StopWait()

		nodeList, err := client.Client.CoreV1().Nodes().List(metav1.ListOptions{})
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
		log.StartWait("Waiting for loadbalancer to get an IP address. This may take several minutes")
		defer log.StopWait()

		now := time.Now()

	Outer:
		for time.Since(now) < waitTimeout {
			// Get loadbalancer
			services, err := client.Client.CoreV1().Services(constants.DevSpaceCloudNamespace).List(metav1.ListOptions{})
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

	// Do the graphql request
	log.StopWait()
	log.StartWait("Configure cluster space domain")

	output := struct {
		UseDefaultClusterDomain string `json:"manager_useDefaultClusterDomain"`
	}{}

	err := p.GrapqhlRequest(`
		mutation($key:String!,$clusterID:Int!) {
			manager_useDefaultClusterDomain(key:$key,clusterID:$clusterID)
	  	}
	`, map[string]interface{}{
		"key":       key,
		"clusterID": clusterID,
	}, &output)
	if err != nil {
		return err
	}
	if output.UseDefaultClusterDomain != "" {
		log.Donef("The domain '%s' has been successfully configured for your clusters spaces and now points to your clusters ingress controller. The dns change however can take several minutes to take affect", ansi.Color("*."+output.UseDefaultClusterDomain, "white+b"))
	}

	return nil
}

// DeleteCluster deletes an cluster
func deleteCluster(p *Provider, clusterID int, key string) error {
	log.StartWait("Rolling back")
	defer log.StopWait()

	err := p.GrapqhlRequest(`
		mutation($key:String!,$clusterID:Int!,$deleteServices:Boolean!,$deleteKubeContexts:Boolean!){
			manager_deleteCluster(
				key:$key,
				clusterID:$clusterID,
				deleteServices:$deleteServices,
				deleteKubeContexts:$deleteKubeContexts
			)
		}
	`, map[string]interface{}{
		"key":                key,
		"clusterID":          clusterID,
		"deleteServices":     false,
		"deleteKubeContexts": false,
	}, &struct {
		DeleteCluster bool `json:"manager_deleteCluster"`
	}{})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) specifyDomain(clusterID int, options *ConnectClusterOptions, log log.Logger) error {
	if options.Domain == "" {
		var err error

		options.Domain, err = survey.Question(&survey.QuestionOptions{
			Question:               "DevSpace will automatically create an ingress for each space, which base domain do you want to use for the created spaces? (e.g. users.test.com)",
			ValidationRegexPattern: "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$",
			ValidationMessage:      "Please enter a valid hostname (e.g. users.my-domain.com)",
		}, log)
		if err != nil {
			return err
		}
	}

	log.StartWait("Updating domain name")
	defer log.StopWait()

	// Update cluster domain
	err := p.GrapqhlRequest(`
		mutation ($clusterID:Int!, $domain:String!) {
			manager_updateClusterDomain(
				clusterID:$clusterID,
				domain:$domain,
				useSSL:false
			)
	  	}
	`, map[string]interface{}{
		"clusterID": clusterID,
		"domain":    options.Domain,
	}, &struct {
		UpdateClusterDomain bool `json:"manager_updateClusterDomain"`
	}{})
	if err != nil {
		return errors.Wrap(err, "update cluster domain")
	}

	log.StopWait()
	if *options.UseHostNetwork == false {
		log.Donef("Please create an A dns record for '*.%s' that points to external IP address of the loadbalancer service 'devspace-cloud/nginx-ingress-controller'.\n Run `%s` to view the service", options.Domain, ansi.Color("kubectl get svc nginx-ingress-controller -n devspace-cloud", "white+b"))
	} else {
		log.Donef("Please create an A dns record for '*.%s' that points to the external IP address of one of your cluster nodes.\n Run `%s` to view your cluster nodes and their IP adresses. \n Please make also sure the ports 80 and 443 can be accessed on these nodes from the internet", options.Domain, ansi.Color("kubectl get nodes -o wide", "white+b"))
	}

	return nil
}

func (p *Provider) deployServices(client *kubectl.Client, clusterID int, availableResources *clusterResources, options *ConnectClusterOptions, log log.Logger) error {
	defer log.StopWait()

	// Check if devspace-cloud is deployed in the namespace
	configmaps, err := client.Client.CoreV1().ConfigMaps(DevSpaceCloudNamespace).List(metav1.ListOptions{
		LabelSelector: "NAME=devspace-cloud,OWNER=TILLER,STATUS=DEPLOYED",
	})
	if err != nil {
		return errors.Wrap(err, "list configmaps")
	}
	if len(configmaps.Items) != 0 {
		options.DeployIngressController = false
	}

	// Ingress controller
	if options.DeployIngressController {
		// Ask if we should use the host network
		if options.UseHostNetwork == nil {
			useHostNetwork, err := survey.Question(&survey.QuestionOptions{
				Question:     "Should the ingress controller use a LoadBalancer or the host network?",
				DefaultValue: loadBalancerOption,
				Options: []string{
					loadBalancerOption,
					hostNetworkOption,
				},
			}, log)
			if err != nil {
				return err
			}

			options.UseHostNetwork = ptr.Bool(useHostNetwork == hostNetworkOption)
		}

		log.StartWait("Deploying ingress controller")

		// Deploy ingress controller
		err := p.GrapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!, $useHostNetwork:Boolean!) {
				manager_deployIngressController(
					clusterID:$clusterID,
					key:$key,
					useHostNetwork:$useHostNetwork
				)
			}
		`, map[string]interface{}{
			"clusterID":      clusterID,
			"key":            options.Key,
			"useHostNetwork": *options.UseHostNetwork,
		}, &struct {
			Deploy bool `json:"manager_deployIngressController"`
		}{})
		if err != nil {
			return errors.Wrap(err, "deploy ingress controller")
		}

		log.Done("Deployed ingress controller")
	}

	// Admission controller
	if options.DeployAdmissionController {
		log.StartWait("Deploying admission controller")

		// Deploy admission controller
		err := p.GrapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!) {
				manager_deployAdmissionController(
					clusterID:$clusterID,
					key:$key
				)
			}
		`, map[string]interface{}{
			"clusterID": clusterID,
			"key":       options.Key,
		}, &struct {
			Deploy bool `json:"manager_deployAdmissionController"`
		}{})
		if err != nil {
			return errors.Wrap(err, "deploy admission controller")
		}

		log.Done("Deployed admission controller")
	}

	// Cert manager
	if availableResources.CertManager == false && options.DeployCertManager {
		log.StartWait("Deploying cert manager")

		// Deploy cert manager
		err := p.GrapqhlRequest(`
			mutation ($clusterID:Int!, $key:String!) {
				manager_deployCertManager(
					clusterID:$clusterID,
					key:$key
				)
			}
		`, map[string]interface{}{
			"clusterID": clusterID,
			"key":       options.Key,
		}, &struct {
			Deploy bool `json:"manager_deployCertManager"`
		}{})
		if err != nil {
			return errors.Wrap(err, "deploy cert manager")
		}

		log.Done("Deployed cert manager")
	}

	return nil
}

func (p *Provider) initCore(clusterID int, key string, enablePodPolicy bool) error {
	log.StartWait("Initializing Cluster")
	defer log.StopWait()

	// Do the request
	err := p.GrapqhlRequest(`
		mutation ($clusterID:Int!, $key:String!, $enablePodPolicy:Boolean!){
			manager_initializeCore(
				clusterID:$clusterID,
				key:$key,
				enablePodPolicy:$enablePodPolicy
			)
	  	}
	`, map[string]interface{}{
		"clusterID":       clusterID,
		"key":             key,
		"enablePodPolicy": enablePodPolicy,
	}, &struct {
		InitCore bool `json:"manager_initializeCore"`
	}{})
	if err != nil {
		return err
	}

	log.Done("Initialized cluster")
	return nil
}

// SettingDefaultClusterEncryptToken is the setting name to check if we need an encryption key
const SettingDefaultClusterEncryptToken = "DEFAULT_CLUSTER_ENCRYPT_TOKEN"

func (p *Provider) needKey() (bool, error) {
	log.StartWait("Retrieving cloud settings")
	defer log.StopWait()

	// Response struct
	response := struct {
		Settings []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"manager_settings"`
	}{}

	// Do the request
	err := p.GrapqhlRequest(`
		query ($settings: [String!]!) {
			manager_settings(settings:$settings) {
				id
				value
			}
		}
	`, map[string]interface{}{
		"settings": []string{SettingDefaultClusterEncryptToken},
	}, &response)
	if err != nil {
		return false, err
	}

	// We don't need a key if not specified
	if len(response.Settings) != 1 {
		return false, nil
	}

	return response.Settings[0].ID == SettingDefaultClusterEncryptToken && response.Settings[0].Value == "true", nil
}

func getServiceAccountCredentials(client *kubectl.Client) ([]byte, string, error) {
	log.StartWait("Retrieving service account credentials")
	defer log.StopWait()

	// Create main service account
	sa, err := client.Client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	beginTimeStamp := time.Now()
	timeout := time.Second * 90

	for len(sa.Secrets) == 0 && time.Since(beginTimeStamp) < timeout {
		time.Sleep(time.Second)

		sa, err = client.Client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
	}

	if time.Since(beginTimeStamp) >= timeout {
		return nil, "", errors.New("ServiceAccount did not receive secret in time")
	}

	// Get secret
	secret, err := client.Client.CoreV1().Secrets(DevSpaceCloudNamespace).Get(sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	return secret.Data["token"], base64.StdEncoding.EncodeToString(secret.Data["ca.crt"]), nil
}

func getKey(provider *Provider, forceQuestion bool, log log.Logger) (string, error) {
	if forceQuestion == false && len(provider.ClusterKey) > 0 {
		keyMap := make(map[string]bool)
		useKey := ""

		for _, key := range provider.ClusterKey {
			keyMap[key] = true
			useKey = key
		}

		if len(keyMap) == 1 {
			return useKey, nil
		}
	}

	for true {
		firstKey, err := survey.Question(&survey.QuestionOptions{
			Question:               "Please enter a secure encryption key for your cluster credentials",
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		}, log)
		if err != nil {
			return "", err
		}

		secondKey, err := survey.Question(&survey.QuestionOptions{
			Question:               "Please re-enter the key",
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		}, log)
		if err != nil {
			return "", err
		}

		if firstKey != secondKey {
			log.Info("Keys do not match! Please reenter")
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

func getClusterName(clusterName string, log log.Logger) (string, error) {
	if clusterName != "" && ClusterNameValidationRegEx.MatchString(clusterName) == false {
		return "", errors.Errorf("Cluster name %s can only contain letters, numbers and dashes (-)", clusterName)
	} else if clusterName != "" {
		return clusterName, nil
	}

	// Ask for cluster name
	for true {
		clusterName, err := survey.Question(&survey.QuestionOptions{
			Question:     "Please enter a cluster name (e.g. my-cluster)",
			DefaultValue: "my-cluster",
		}, log)
		if err != nil {
			return "", err
		}

		if ClusterNameValidationRegEx.MatchString(clusterName) == false {
			log.Infof("Cluster name %s can only contain letters, numbers and dashes (-)", clusterName)
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
func checkResources(client kubernetes.Interface) (*clusterResources, error) {
	log.StartWait("Checking cluster resources")
	defer log.StopWait()

	// Check if cluster has active nodes
	nodeList, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list cluster nodes")
	}
	if len(nodeList.Items) == 0 {
		return nil, errors.Errorf("The cluster specified has no nodes, please choose a cluster where at least one node is up and running")
	}

	groupResources, err := client.Discovery().ServerResources()
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

func initializeNamespace(client kubernetes.Interface) error {
	log.StartWait("Initializing namespace")
	defer log.StopWait()

	// Create devspace-cloud namespace
	_, err := client.CoreV1().Namespaces().Get(DevSpaceCloudNamespace, metav1.GetOptions{})
	if err != nil {
		_, err = client.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: DevSpaceCloudNamespace,
			},
		})
		if err != nil {
			return errors.Wrap(err, "create namespace")
		}

		log.Donef("Created namespace '%s'", DevSpaceCloudNamespace)
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

		log.Donef("Created service account '%s'", DevSpaceServiceAccount)
	}

	// Create cluster-admin clusterrole binding
	_, err = client.RbacV1beta1().ClusterRoleBindings().Get(DevSpaceClusterRoleBinding, metav1.GetOptions{})
	if err != nil {
		_, err = client.RbacV1beta1().ClusterRoleBindings().Create(&v1beta1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: DevSpaceClusterRoleBinding,
			},
			Subjects: []v1beta1.Subject{
				{
					Kind:      v1beta1.ServiceAccountKind,
					Name:      DevSpaceServiceAccount,
					Namespace: DevSpaceCloudNamespace,
				},
			},
			RoleRef: v1beta1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
		})
		if err != nil {
			return errors.Wrap(err, "create cluster role binding")
		}

		log.Infof("Created cluster role binding '%s'", DevSpaceClusterRoleBinding)
	}

	return nil
}

// ResetKey resets a cluster key
func (p *Provider) ResetKey(clusterName string, log log.Logger) error {
	cluster, err := p.GetClusterByName(clusterName)
	if err != nil {
		return errors.Wrap(err, "get cluster")
	}
	clusterUser, err := p.GetClusterUser(cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "get cluster user")
	}

	// Get kube context to use
	client, err := kubectl.NewClientBySelect(false, false, log)
	if err != nil {
		return err
	}
	if client.RestConfig.Host != *cluster.Server {
		return errors.Errorf("Selected context does not point to the correct host. Selected %s <> %s", client.RestConfig.Host, *cluster.Server)
	}

	key, err := getKey(p, true, log)
	if err != nil {
		return errors.Wrap(err, "get key")
	}

	token, _, err := getServiceAccountCredentials(client)
	if err != nil {
		return errors.Wrap(err, "get service account credentials")
	}

	encryptedToken, err := EncryptAES([]byte(key), token)
	if err != nil {
		return errors.Wrap(err, "encrypt token")
	}

	// Update token
	log.StartWait("Update token")
	defer log.StopWait()

	// Do the request
	err = p.GrapqhlRequest(`
		mutation($clusterUserID:Int!, $encryptedToken:String!) {
			manager_updateUserClusterUser(
				clusterUserID:$clusterUserID, 
				encryptedToken:$encryptedToken
			)
	  	}
	`, map[string]interface{}{
		"clusterUserID":  clusterUser.ClusterUserID,
		"encryptedToken": encryptedToken,
	}, &struct {
		UpdateClusterUser bool `json:"manager_updateUserClusterUser"`
	}{})
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
