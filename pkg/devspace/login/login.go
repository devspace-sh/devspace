package login

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	"github.com/covexo/devspace/pkg/util/kubeconfig"
	"github.com/covexo/devspace/pkg/util/yamlutil"

	"github.com/covexo/devspace/pkg/util/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/skratchdot/open-golang/open"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// DevSpaceCloudConfigPath holds the path to the cloud config file
const DevSpaceCloudConfigPath = ".devspace/cloudConfig.yaml"

// DevSpaceCloudLogin is the endpoint to login an user
const DevSpaceCloudLogin = "https://cloud.devspace.covexo.com/login"

// DevSpaceCloudGetClusterConfig is the endpoint to retrieve the cluster configuration
const DevSpaceCloudGetClusterConfig = "https://cloud.devspace.covexo.com/clusterConfig"

// DevSpaceCloudContextName is the name for the kube config context
const DevSpaceCloudContextName = "devspace-cloud"

// DevSpaceCloudConfig describes the struct to hold the cloud configuration
type DevSpaceCloudConfig struct {
	Token string `yaml:"token"`
}

// CheckAuth verifies if the user is logged into the devspace cloud and if not logs the user in
func CheckAuth() (string, *api.Cluster, *api.AuthInfo, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return "", nil, nil, err
	}

	data, err := ioutil.ReadFile(filepath.Join(homedir, DevSpaceCloudConfigPath))
	if os.IsNotExist(err) {
		return Login()
	} else if err != nil {
		return "", nil, nil, errors.Wrapf(err, "Error reading file %q", filepath.Join(homedir, DevSpaceCloudConfigPath))
	}

	cloudConfig := &DevSpaceCloudConfig{}
	err = yaml.Unmarshal(data, cloudConfig)
	if err != nil {
		return "", nil, nil, err
	}

	return GetClusterConfig(cloudConfig)
}

// GetClusterConfig retrieves the cluster and authconfig from the devspace cloud
func GetClusterConfig(cfg *DevSpaceCloudConfig) (string, *api.Cluster, *api.AuthInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", DevSpaceCloudGetClusterConfig, nil)
	if err != nil {
		return "", nil, nil, err
	}

	req.Header.Set("Authorization", cfg.Token)

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, nil, err
	}

	if resp.StatusCode == 401 {
		return Login()
	}
	if resp.StatusCode != 200 {
		return "", nil, nil, fmt.Errorf("Couldn't retrieve cluster config: %s", body)
	}

	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return "", nil, nil, err
	}

	cluster := api.NewCluster()
	err = json.Unmarshal(*objmap["cluster"], cluster)
	if err != nil {
		return "", nil, nil, err
	}

	authInfo := api.NewAuthInfo()
	err = json.Unmarshal(*objmap["user"], authInfo)
	if err != nil {
		return "", nil, nil, err
	}

	namespace := ""
	err = json.Unmarshal(*objmap["namespace"], &namespace)
	if err != nil {
		return "", nil, nil, err
	}

	return namespace, cluster, authInfo, nil
}

// Login logs the user into the devspace cloud
func Login() (string, *api.Cluster, *api.AuthInfo, error) {
	tokenChannel := make(chan string)
	homedir, err := homedir.Dir()
	if err != nil {
		return "", nil, nil, err
	}

	cfgPath := filepath.Join(homedir, DevSpaceCloudConfigPath)

	log.StartWait("Logging into DevSpace cloud " + DevSpaceCloudLogin + " ...")
	server := startServer(tokenChannel)

	open.Start(DevSpaceCloudLogin)

	token := <-tokenChannel
	close(tokenChannel)

	err = server.Shutdown(nil)
	if err != nil {
		return "", nil, nil, err
	}

	cfg := DevSpaceCloudConfig{
		Token: token,
	}

	err = os.MkdirAll(filepath.Dir(cfgPath), 0755)
	if err != nil {
		return "", nil, nil, err
	}

	err = yamlutil.WriteYamlToFile(cfg, cfgPath)
	if err != nil {
		return "", nil, nil, err
	}

	return GetClusterConfig(&cfg)
}

// Update updates the devspace cloud information if necessary
func Update(config *v1.Config, switchContext bool) error {
	// Don't update anything if we don't use the devspace cloud
	if *config.Cluster.DevSpaceCloud == false {
		return nil
	}

	namespace, cluster, authInfo, err := CheckAuth()
	if err != nil {
		return err
	}

	config.DevSpace.Release.Namespace = &namespace
	config.Services.Tiller.Release.Namespace = &namespace

	if *config.Cluster.UseKubeConfig {
		err = UpdateKubeConfig(cluster, authInfo, switchContext)
		if err != nil {
			return err
		}

		config.Cluster.KubeContext = configutil.String(DevSpaceCloudContextName)
	} else {
		config.Cluster.APIServer = &cluster.Server
		config.Cluster.CaCert = configutil.String(string(cluster.CertificateAuthorityData))

		config.Cluster.User = &v1.ClusterUser{
			ClientCert: configutil.String(string(authInfo.ClientCertificateData)),
			ClientKey:  configutil.String(string(authInfo.ClientKeyData)),
		}
	}

	return err
}

// UpdateKubeConfig adds the devspace-cloud context if necessary and switches the current context
func UpdateKubeConfig(cluster *api.Cluster, authInfo *api.AuthInfo, switchContext bool) error {
	config, err := kubeconfig.ReadKubeConfig(clientcmd.RecommendedHomeFile)
	if err != nil {
		return err
	}

	// Switch context if necessary
	if switchContext && config.CurrentContext != DevSpaceCloudContextName {
		config.CurrentContext = DevSpaceCloudContextName
	}

	config.Clusters[DevSpaceCloudContextName] = cluster
	config.AuthInfos[DevSpaceCloudContextName] = authInfo

	// Check if we need to add the context
	if _, ok := config.Contexts[DevSpaceCloudContextName]; !ok {
		context := api.NewContext()
		context.Cluster = DevSpaceCloudContextName
		context.AuthInfo = DevSpaceCloudContextName

		config.Contexts[DevSpaceCloudContextName] = context
	}

	return kubeconfig.WriteKubeConfig(config, clientcmd.RecommendedHomeFile)
}

func startServer(tokenChannel chan string) *http.Server {
	srv := &http.Server{Addr: ":25853"}

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<script type=\"text/javascript\">window.close();</script>")

		keys, ok := r.URL.Query()["token"]
		if !ok || len(keys[0]) < 1 {
			log.Fatal("Bad request")
		}

		log.StopWait()
		tokenChannel <- keys[0]
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
