package v2

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/repo"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	homedir "github.com/mitchellh/go-homedir"
	k8shelm "k8s.io/helm/pkg/helm"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/helm/portforwarder"
)

// client holds the necessary information for helm
type client struct {
	Settings  *helmenvironment.EnvSettings
	Namespace string

	helm     k8shelm.Interface
	kubectl  kubectl.Client
	analyzer analyze.Analyzer

	config *latest.Config
	log    log.Logger
}

// NewClient creates a new helm client
func NewClient(config *latest.Config, kubeClient kubectl.Client, tillerNamespace string, log log.Logger, upgradeTiller bool) (types.Client, error) {
	return createNewClient(config, kubeClient, tillerNamespace, log, upgradeTiller)
}

func createNewClient(config *latest.Config, kubeClient kubectl.Client, tillerNamespace string, log log.Logger, upgradeTiller bool) (*client, error) {
	// Create tiller if necessary
	err := ensureTiller(config, kubeClient, tillerNamespace, upgradeTiller, log)
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
			tunnel, err = portforwarder.New(tillerNamespace, kubeClient.KubeClient(), kubeClient.RestConfig())
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

	return create(config, tillerNamespace, helmClient, kubeClient, true, log)
}

func create(config *latest.Config, tillerNamespace string, helmClient k8shelm.Interface, kubeClient kubectl.Client, updateRepo bool, log log.Logger) (*client, error) {
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

	wrapper := &client{
		Settings: &helmenvironment.EnvSettings{
			Home: helmpath.Home(helmHomePath),
		},
		Namespace: tillerNamespace,
		helm:      helmClient,
		kubectl:   kubeClient,
		config:    config,
		analyzer:  analyze.NewAnalyzer(kubeClient, log),
		log:       log,
	}

	if updateRepo {
		_, err = os.Stat(stableRepoCachePathAbs)
		if err != nil {
			err = wrapper.UpdateRepos()
			if err != nil {
				return nil, err
			}
		}
	}

	return wrapper, nil
}

// UpdateRepos will update the helm repositories
func (client *client) UpdateRepos() error {
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

			err := re.DownloadIndexFile(client.Settings.Home.String())
			if err != nil {
				client.log.Errorf("Unable to download repo index: %v", err)
			}
		}(re)
	}

	wg.Wait()
	return nil
}

// ReleaseExists checks if the given release name exists
func ReleaseExists(helm helm.Interface, releaseName string) bool {
	releases, err := helm.ListReleases()
	if err != nil {
		return false
	}

	if releases != nil {
		for _, release := range releases.Releases {
			if release.Name == releaseName {
				return true
			}
		}
	}

	return false
}

// DeleteRelease deletes a helm release and optionally purges it
func (client *client) DeleteRelease(releaseName string, purge bool) error {
	_, err := client.helm.DeleteRelease(releaseName, k8shelm.DeletePurge(purge))
	return err
}

// ListReleases lists all helm releases
func (client *client) ListReleases() ([]*types.Release, error) {
	releases, err := client.helm.ListReleases()
	if err != nil {
		return nil, err
	} else if releases == nil {
		return nil, nil
	}

	retReleases := make([]*types.Release, len(releases.Releases))
	for i, release := range releases.Releases {
		retReleases[i] = &types.Release{
			Name:         release.GetName(),
			Namespace:    release.GetNamespace(),
			Version:      release.Version,
			Status:       release.Info.Status.Code.String(),
			LastDeployed: time.Unix(release.Info.LastDeployed.Seconds, 0),
		}
	}

	return retReleases, nil
}
