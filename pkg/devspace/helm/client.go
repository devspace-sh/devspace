package helm

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/repo"

	"k8s.io/client-go/kubernetes"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	homedir "github.com/mitchellh/go-homedir"
	k8shelm "k8s.io/helm/pkg/helm"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/helm/portforwarder"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	helmstoragedriver "k8s.io/helm/pkg/storage/driver"
)

// Client holds the necessary information for helm
type Client struct {
	Settings  *helmenvironment.EnvSettings
	Namespace string

	helm    *k8shelm.Client
	kubectl kubernetes.Interface
}

// NewClient creates a new helm client
// NOTE: This is not safe to use in goroutines and could cause multiple creation of the same client
func NewClient(tillerNamespace string, log log.Logger, upgradeTiller bool) (*Client, error) {
	client, err := createNewClient(tillerNamespace, log, upgradeTiller)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func createNewClient(tillerNamespace string, log log.Logger, upgradeTiller bool) (*Client, error) {
	// Get kube config
	kubeconfig, err := kubectl.GetClientConfig()
	if err != nil {
		return nil, err
	}

	// Create client from config
	kubectlClient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	// Create tiller if necessary
	err = ensureTiller(kubectlClient, tillerNamespace, upgradeTiller)
	if err != nil {
		return nil, err
	}

	var tunnel *kube.Tunnel
	var helmClient *k8shelm.Client

	tunnelWaitTime := 2 * 60 * time.Second
	tunnelCheckInterval := 5 * time.Second

	log.StartWait("Waiting for " + tillerNamespace + "/tiller-deploy to become ready")
	defer log.StopWait()

	for true {
		// Next we wait till we can establish a tunnel to the running pod
		for true {
			tunnel, err = portforwarder.New(tillerNamespace, kubectlClient, kubeconfig)
			if err == nil && tunnel != nil {
				break
			}
			if tunnelWaitTime <= 0 {
				return nil, err
			}

			tunnelWaitTime = tunnelWaitTime - tunnelCheckInterval
			time.Sleep(tunnelCheckInterval)
		}

		helmOptions := []k8shelm.Option{
			k8shelm.Host("127.0.0.1:" + strconv.Itoa(tunnel.Local)),
			k8shelm.ConnectTimeout(int64(5 * time.Second)),
		}

		helmClient = k8shelm.NewClient(helmOptions...)
		_, err = helmClient.ListReleases(k8shelm.ReleaseListLimit(1))

		if err == nil {
			break
		}

		tunnel.Close()

		tunnelWaitTime = tunnelWaitTime - tunnelCheckInterval
		time.Sleep(tunnelCheckInterval)

		if tunnelWaitTime < 0 {
			return nil, errors.New("Waiting for tiller timed out")
		}
	}

	log.StopWait()

	homeDir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	helmHomePath := homeDir + "/.helm"
	repoPath := helmHomePath + "/repository"
	repoFile := repoPath + "/repositories.yaml"
	stableRepoCachePathAbs := helmHomePath + "/" + stableRepoCachePath

	os.MkdirAll(helmHomePath+"/cache", os.ModePerm)
	os.MkdirAll(repoPath, os.ModePerm)
	os.MkdirAll(filepath.Dir(stableRepoCachePathAbs), os.ModePerm)

	repoFileStat, repoFileNotFound := os.Stat(repoFile)
	if repoFileNotFound != nil || repoFileStat.Size() == 0 {
		err = fsutil.WriteToFile([]byte(defaultRepositories), repoFile)
		if err != nil {
			return nil, err
		}
	}

	wrapper := &Client{
		Settings: &helmenvironment.EnvSettings{
			Home: helmpath.Home(helmHomePath),
		},
		Namespace: tillerNamespace,
		helm:      helmClient,
		kubectl:   kubectlClient,
	}

	_, err = os.Stat(stableRepoCachePathAbs)
	if err != nil {
		err = wrapper.UpdateRepos()
		if err != nil {
			return nil, err
		}
	}

	return wrapper, nil
}

// UpdateRepos will update the helm repositories
func (client *Client) UpdateRepos() error {
	allRepos, err := repo.LoadRepositoriesFile(client.Settings.Home.RepositoryFile())
	if err != nil {
		return err
	}

	repos := []*repo.ChartRepository{}
	for _, repoData := range allRepos.Repositories {
		repo, err := repo.NewChartRepository(repoData, getter.All(*client.Settings))
		if err != nil {
			return err
		}

		repos = append(repos, repo)
	}

	wg := sync.WaitGroup{}
	for _, re := range repos {
		wg.Add(1)

		go func(re *repo.ChartRepository) {
			defer wg.Done()

			if re.Config.Name == "local" {
				return
			}

			err := re.DownloadIndexFile(client.Settings.Home.String())
			if err != nil {
				log.Errorf("Unable to download repo index: %v", err)
			}
		}(re)
	}

	wg.Wait()
	return nil
}

// ReleaseExists checks if the given release name exists
func (client *Client) ReleaseExists(releaseName string) (bool, error) {
	_, err := client.helm.ReleaseHistory(releaseName, k8shelm.WithMaxHistory(1))
	if err != nil {
		if strings.Contains(err.Error(), helmstoragedriver.ErrReleaseNotFound(releaseName).Error()) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// DeleteRelease deletes a helm release and optionally purges it
func (client *Client) DeleteRelease(releaseName string, purge bool) (*rls.UninstallReleaseResponse, error) {
	return client.helm.DeleteRelease(releaseName, k8shelm.DeletePurge(purge))
}

// ListReleases lists all helm releases
func (client *Client) ListReleases() (*rls.ListReleasesResponse, error) {
	return client.helm.ListReleases()
}
