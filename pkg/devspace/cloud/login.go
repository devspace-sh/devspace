package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/skratchdot/open-golang/open"
	"k8s.io/client-go/tools/clientcmd/api"
)

// CheckAuth verifies if the user is logged into the devspace cloud and if not logs the user in
func CheckAuth(provider *Provider, devSpaceID, target string, log log.Logger) (string, string, *api.Cluster, *api.AuthInfo, error) {
	if provider.Token == "" {
		return Login(provider, devSpaceID, target, log)
	}

	return GetClusterConfig(provider, devSpaceID, target, log)
}

// GetClusterConfig retrieves the cluster and authconfig from the devspace cloud
func GetClusterConfig(provider *Provider, devSpaceID, target string, log log.Logger) (string, string, *api.Cluster, *api.AuthInfo, error) {
	log.StartWait("Retrieving auth info from cloud provider...")
	defer log.StopWait()

	client := &http.Client{}
	req, err := http.NewRequest("GET", provider.Host+GetClusterConfigEndpoint, nil)
	if err != nil {
		return "", "", nil, nil, err
	}

	req.Header.Set("Authorization", provider.Token)

	if devSpaceID != "" || target != "" {
		q := req.URL.Query()
		if devSpaceID != "" {
			q.Add("namespace", devSpaceID)
		}
		if target != "" {
			q.Add("target", target)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", nil, nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, nil, err
	} else if resp.StatusCode == http.StatusUnauthorized {
		return Login(provider, devSpaceID, target, log)
	} else if resp.StatusCode != http.StatusOK {
		return "", "", nil, nil, fmt.Errorf("Couldn't retrieve cluster config: %s. Status: %d", body, resp.StatusCode)
	}

	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(body, &objmap)
	if err != nil {
		return "", "", nil, nil, err
	}

	cluster := api.NewCluster()
	err = json.Unmarshal(*objmap["cluster"], cluster)
	if err != nil {
		return "", "", nil, nil, err
	}

	authInfo := api.NewAuthInfo()
	err = json.Unmarshal(*objmap["user"], authInfo)
	if err != nil {
		return "", "", nil, nil, err
	}

	namespace := ""
	err = json.Unmarshal(*objmap["namespace"], &namespace)
	if err != nil {
		return "", "", nil, nil, err
	}

	domain := ""
	err = json.Unmarshal(*objmap["domain"], &domain)
	if err != nil {
		return "", "", nil, nil, err
	}

	return domain, namespace, cluster, authInfo, nil
}

// Login logs the user into the devspace cloud
func Login(provider *Provider, namespace, target string, log log.Logger) (string, string, *api.Cluster, *api.AuthInfo, error) {
	log.StartWait("Logging into cloud provider...")
	defer log.StopWait()

	ctx := context.Background()
	tokenChannel := make(chan string)

	server := startServer(provider.Host+LoginSuccessEndpoint, tokenChannel)
	open.Start(provider.Host + LoginEndpoint)

	token := <-tokenChannel
	close(tokenChannel)

	err := server.Shutdown(ctx)
	if err != nil {
		return "", "", nil, nil, err
	}

	providerConfig, err := ParseCloudConfig()
	if err != nil {
		return "", "", nil, nil, err
	}

	providerConfig[provider.Name].Token = token

	err = SaveCloudConfig(providerConfig)
	if err != nil {
		return "", "", nil, nil, err
	}

	return GetClusterConfig(providerConfig[provider.Name], namespace, target, log)
}

func startServer(redirectURI string, tokenChannel chan string) *http.Server {
	srv := &http.Server{Addr: ":25853"}

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		keys, ok := r.URL.Query()["token"]
		if !ok || len(keys[0]) < 1 {
			log.Fatal("Bad request")
		}

		tokenChannel <- keys[0]
		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
