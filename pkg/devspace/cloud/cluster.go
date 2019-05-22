package cloud

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

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
	"k8s.io/client-go/rest"
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
func (p *Provider) ConnectCluster(options *ConnectClusterOptions) error {
	var (
		config *rest.Config
	)

	// Get cluster name
	clusterName, err := getClusterName(options.ClusterName)
	if err != nil {
		return err
	}

	// Check what kube context to use
	if options.KubeContext == "" {
		// Get kube context to use
		config, err = kubectl.GetClientConfigBySelect(false, true)
		if err != nil {
			return errors.Wrap(err, "new kubectl client")
		}
	} else {
		// Get kube context to use
		config, err = kubectl.GetClientConfigFromContext(options.KubeContext)
		if err != nil {
			return errors.Wrap(err, "new kubectl client")
		}
	}

	// Get client from config
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "get kubernetes client")
	}

	// Check available cluster resources
	availableResources, err := checkResources(client)
	if err != nil {
		return errors.Wrap(err, "check resource availability")
	}

	// Initialize namespace
	err = initializeNamespace(client)
	if err != nil {
		return errors.Wrap(err, "init namespace")
	}

	if options.Key == "" {
		options.Key, err = getKey(p, false)
		if err != nil {
			return errors.Wrap(err, "get key")
		}
	}

	token, caCert, err := getServiceAccountCredentials(client)
	if err != nil {
		return errors.Wrap(err, "get service account credentials")
	}

	encryptedToken, err := EncryptAES([]byte(options.Key), token)
	if err != nil {
		return errors.Wrap(err, "encrypt token")
	}

	// Create cluster remotely
	log.StartWait("Initialize cluster")
	defer log.StopWait()
	clusterID, err := p.CreateUserCluster(clusterName, config.Host, caCert, base64.StdEncoding.EncodeToString(encryptedToken), availableResources.NetworkPolicy)
	if err != nil {
		return errors.Wrap(err, "create cluster")
	}
	log.StopWait()

	// Save key
	p.ClusterKey[clusterID] = options.Key
	err = p.Save()
	if err != nil {
		return errors.Wrap(err, "save key")
	}

	// Initialize roles and pod security policies
	err = p.initCore(clusterID, options.Key, availableResources.PodPolicy)
	if err != nil {
		// Try to delete cluster if core initialization has failed
		deleteCluster(p, clusterID, options.Key)

		return errors.Wrap(err, "initialize core")
	}

	// Ask if we should use the host network
	if options.UseHostNetwork == nil {
		options.UseHostNetwork = ptr.Bool(survey.Question(&survey.QuestionOptions{
			Question:     "Should the ingress controller use a LoadBalancer or the host network?",
			DefaultValue: loadBalancerOption,
			Options: []string{
				loadBalancerOption,
				hostNetworkOption,
			},
		}) == hostNetworkOption)
	}

	// Deploy admission controller, ingress controller and cert manager
	err = p.deployServices(clusterID, availableResources, options)
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

func (p *Provider) specifyDomain(clusterID int, options *ConnectClusterOptions) error {
	if options.Domain == "" {
		options.Domain = survey.Question(&survey.QuestionOptions{
			Question:               "DevSpace will automatically create an ingress for each space, which base domain do you want to use for the created spaces? (e.g. users.test.com)",
			ValidationRegexPattern: "^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$",
			ValidationMessage:      "Please enter a valid hostname (e.g. users.my-domain.com)",
		})
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
		log.Donef("Please create an A dns record for '*.%s' that points to external ip of the loadbalancer service 'devspace-cloud/nginx-ingress-controller'.\n Run `%s` to view the service", options.Domain, ansi.Color("kubectl get svc nginx-ingress-controller -n devspace-cloud", "white+b"))
	} else {
		log.Donef("Please create an A dns record for '*.%s' that points to the external ip of one of your cluster nodes.\n Run `%s` to view your cluster nodes and their ip adresses. \n Please make also sure the ports 80 and 443 can be accessed on these nodes from the internet", options.Domain, ansi.Color("kubectl get nodes -o wide", "white+b"))
	}

	return nil
}

func (p *Provider) deployServices(clusterID int, availableResources *clusterResources, options *ConnectClusterOptions) error {
	defer log.StopWait()

	// Ingress controller
	if options.DeployIngressController {
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

func getServiceAccountCredentials(client kubernetes.Interface) ([]byte, string, error) {
	log.StartWait("Retrieving service account credentials")
	defer log.StopWait()

	// Create main service account
	sa, err := client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	beginTimeStamp := time.Now()
	timeout := time.Second * 90

	for len(sa.Secrets) == 0 && time.Since(beginTimeStamp) < timeout {
		time.Sleep(time.Second)

		sa, err = client.CoreV1().ServiceAccounts(DevSpaceCloudNamespace).Get(DevSpaceServiceAccount, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
	}

	if time.Since(beginTimeStamp) >= timeout {
		return nil, "", errors.New("ServiceAccount did not receive secret in time")
	}

	// Get secret
	secret, err := client.CoreV1().Secrets(DevSpaceCloudNamespace).Get(sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}

	return secret.Data["token"], base64.StdEncoding.EncodeToString(secret.Data["ca.crt"]), nil
}

func getKey(provider *Provider, forceQuestion bool) (string, error) {
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
		firstKey := survey.Question(&survey.QuestionOptions{
			Question:               "Please enter a secure encryption key for your cluster credentials",
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		})

		secondKey := survey.Question(&survey.QuestionOptions{
			Question:               "Please re-enter the key",
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		})

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

func getClusterName(clusterName string) (string, error) {
	if clusterName != "" && ClusterNameValidationRegEx.MatchString(clusterName) == false {
		return "", fmt.Errorf("Cluster name %s can only contain letters, numbers and dashes (-)", clusterName)
	}

	// Ask for cluster name
	for true {
		clusterName = survey.Question(&survey.QuestionOptions{
			Question:     "Please enter a cluster name (e.g. my-cluster)",
			DefaultValue: "my-cluster",
		})

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
		return nil, fmt.Errorf("The cluster specified has no nodes, please choose a cluster where at least one node is up and running")
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
func (p *Provider) ResetKey(clusterName string) error {
	cluster, err := p.GetClusterByName(clusterName)
	if err != nil {
		return errors.Wrap(err, "get cluster")
	}
	clusterUser, err := p.GetClusterUser(cluster.ClusterID)
	if err != nil {
		return errors.Wrap(err, "get cluster user")
	}

	// Get kube context to use
	config, err := kubectl.GetClientConfigBySelect(false, false)
	if err != nil {
		return errors.Wrap(err, "new kubectl client")
	}
	if config.Host != *cluster.Server {
		return fmt.Errorf("Selected context does not point to the correct host. Selected %s <> %s", config.Host, *cluster.Server)
	}

	// Get client from config
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "get kubernetes client")
	}

	key, err := getKey(p, true)
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
