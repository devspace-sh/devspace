package helm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/covexo/devspace/pkg/util/fsutil"
	"github.com/covexo/devspace/pkg/util/log"

	helminstaller "k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/repo"

	"k8s.io/client-go/kubernetes"

	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/covexo/devspace/pkg/devspace/config/configutil"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	homedir "github.com/mitchellh/go-homedir"
	k8sv1 "k8s.io/api/core/v1"
	k8sv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmchartutil "k8s.io/helm/pkg/chartutil"
	helmdownloader "k8s.io/helm/pkg/downloader"
	k8shelm "k8s.io/helm/pkg/helm"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/helm/portforwarder"
	hapi_release5 "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	helmstoragedriver "k8s.io/helm/pkg/storage/driver"
)

// HelmClientWrapper holds the necessary information for helm
type HelmClientWrapper struct {
	Client       *k8shelm.Client
	Settings     *helmenvironment.EnvSettings
	TillerConfig *v1.TillerConfig
	kubectl      *kubernetes.Clientset
}

// TillerDeploymentName is the string identifier for the tiller deployment
const TillerDeploymentName = "tiller-deploy"
const tillerServiceAccountName = "devspace-tiller"
const tillerRoleName = "devspace-tiller"
const tillerRoleManagerName = "tiller-config-manager"
const stableRepoCachePath = "repository/cache/stable-index.yaml"
const defaultRepositories = `apiVersion: v1
repositories:
- caFile: ""
  cache: ` + stableRepoCachePath + `
  certFile: ""
  keyFile: ""
  name: stable
  url: https://kubernetes-charts.storage.googleapis.com
`

var alreadyExistsRegexp = regexp.MustCompile(".* already exists$")

var defaultPolicyRules = []k8sv1beta1.PolicyRule{
	{
		APIGroups: []string{
			k8sv1beta1.APIGroupAll,
			"extensions",
			"apps",
		},
		Resources: []string{k8sv1beta1.ResourceAll},
		Verbs:     []string{k8sv1beta1.ResourceAll},
	},
}

// NewClient creates a new helm client
func NewClient(kubectlClient *kubernetes.Clientset, upgradeTiller bool) (*HelmClientWrapper, error) {
	config := configutil.GetConfig(false)

	tillerConfig := config.Services.Tiller
	kubeconfig, err := kubectl.GetClientConfig()
	if err != nil {
		return nil, err
	}

	err = ensureTiller(kubectlClient, config, upgradeTiller)
	if err != nil {
		return nil, err
	}

	var tunnel *kube.Tunnel

	tunnelWaitTime := 2 * 60 * time.Second
	tunnelCheckInterval := 5 * time.Second

	log.StartWait("Waiting for tiller to become ready")
	defer log.StopWait()

	// Next we wait till we can establish a tunnel to the running pod
	for tunnelWaitTime > 0 {
		tunnel, err = portforwarder.New(*tillerConfig.Release.Namespace, kubectlClient, kubeconfig)
		if err == nil {
			break
		}

		if tunnelWaitTime <= 0 {
			return nil, err
		}

		tunnelWaitTime = tunnelWaitTime - tunnelCheckInterval
		time.Sleep(tunnelCheckInterval)
	}

	helmWaitTime := 2 * 60 * time.Second
	helmCheckInterval := 5 * time.Second

	helmOptions := []k8shelm.Option{
		k8shelm.Host("127.0.0.1:" + strconv.Itoa(tunnel.Local)),
		k8shelm.ConnectTimeout(int64(helmCheckInterval)),
	}

	client := k8shelm.NewClient(helmOptions...)
	var tillerError error

	for helmWaitTime > 0 {
		_, tillerError = client.ListReleases(k8shelm.ReleaseListLimit(1))

		if tillerError == nil || helmWaitTime < 0 {
			break
		}

		helmWaitTime = helmWaitTime - helmCheckInterval
		time.Sleep(helmCheckInterval)
	}

	log.StopWait()

	if tillerError != nil {
		return nil, tillerError
	}

	homeDir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	helmHomePath := homeDir + "/.devspace/helm"
	repoPath := helmHomePath + "/repository"
	repoFile := repoPath + "/repositories.yaml"
	stableRepoCachePathAbs := helmHomePath + "/" + stableRepoCachePath

	os.MkdirAll(helmHomePath+"/cache", os.ModePerm)
	os.MkdirAll(repoPath, os.ModePerm)
	os.MkdirAll(filepath.Dir(stableRepoCachePathAbs), os.ModePerm)

	_, repoFileNotFound := os.Stat(repoFile)

	if repoFileNotFound != nil {
		err = fsutil.WriteToFile([]byte(defaultRepositories), repoFile)
		if err != nil {
			return nil, err
		}
	}

	wrapper := &HelmClientWrapper{
		Client: client,
		Settings: &helmenvironment.EnvSettings{
			Home: helmpath.Home(helmHomePath),
		},
		TillerConfig: tillerConfig,
		kubectl:      kubectlClient,
	}

	_, err = os.Stat(stableRepoCachePathAbs)
	if err != nil {
		err = wrapper.updateRepos()
		if err != nil {
			return nil, err
		}
	}

	return wrapper, nil
}

func ensureTiller(kubectlClient *kubernetes.Clientset, config *v1.Config, upgrade bool) error {
	tillerConfig := config.Services.Tiller
	tillerNamespace := *tillerConfig.Release.Namespace
	tillerSA := &k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tillerServiceAccountName,
			Namespace: tillerNamespace,
		},
	}
	tillerOptions := &helminstaller.Options{
		Namespace:      tillerNamespace,
		MaxHistory:     10,
		ImageSpec:      "gcr.io/kubernetes-helm/tiller:v2.9.1",
		ServiceAccount: tillerSA.ObjectMeta.Name,
	}

	// Check if tiller namespace exists
	_, err := kubectlClient.CoreV1().Namespaces().Get(tillerNamespace, metav1.GetOptions{})
	if err != nil {
		// Create tiller namespace
		_, err := kubectlClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tillerNamespace,
			},
		})

		if err != nil {
			return err
		}
	}

	_, tillerCheckErr := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})

	// Tiller is not there
	if tillerCheckErr != nil {
		log.StartWait("Installing Tiller server")
		defer log.StopWait()

		_, err := kubectlClient.CoreV1().ServiceAccounts(tillerSA.Namespace).Get(tillerSA.Name, metav1.GetOptions{})
		if err != nil {
			_, err := kubectlClient.CoreV1().ServiceAccounts(tillerSA.Namespace).Create(tillerSA)
			if err != nil {
				return err
			}
		}
		serviceAccountSubject := []k8sv1beta1.Subject{
			{
				Kind:      k8sv1beta1.ServiceAccountKind,
				Name:      tillerServiceAccountName,
				Namespace: tillerNamespace,
			},
		}

		err = ensureRoleBinding(kubectlClient, tillerConfig, tillerRoleManagerName, tillerNamespace, []k8sv1beta1.PolicyRule{
			{
				APIGroups: []string{
					k8sv1beta1.APIGroupAll,
					"extensions",
					"apps",
				},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{k8sv1beta1.ResourceAll},
			},
		}, serviceAccountSubject)
		if err != nil {
			return err
		}

		err = helminstaller.Install(kubectlClient, tillerOptions)
		if err != nil {
			return err
		}

		appNamespaces := []*string{
			config.DevSpace.Release.Namespace,
		}

		if config.Services.InternalRegistry != nil && config.Services.InternalRegistry.Release.Namespace != nil {
			appNamespaces = append(appNamespaces, config.Services.InternalRegistry.Release.Namespace)
		}

		tillerConfig.AppNamespaces = &appNamespaces
		for _, appNamespace := range *tillerConfig.AppNamespaces {
			if *appNamespace == tillerRoleManagerName {
				continue
			}

			err = ensureRoleBinding(kubectlClient, tillerConfig, tillerRoleName, *appNamespace, defaultPolicyRules, serviceAccountSubject)
			if err != nil {
				return err
			}
		}

		log.StopWait()
		log.Done("Tiller started")

		//Upgrade of tiller is necessary
	} else if upgrade {
		log.StartWait("Upgrading tiller")

		tillerOptions.ImageSpec = ""
		err := helminstaller.Upgrade(kubectlClient, tillerOptions)

		log.StopWait()

		if err != nil {
			return err
		}
	}

	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second

	log.StartWait("Waiting for tiller to start")

	for tillerWaitingTime > 0 {
		tillerDeployment, err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})

		if err != nil {
			continue
		}

		if tillerDeployment.Status.ReadyReplicas == tillerDeployment.Status.Replicas {
			break
		}

		time.Sleep(tillerCheckInterval)
		tillerWaitingTime = tillerWaitingTime - tillerCheckInterval
	}

	log.StopWait()

	return nil
}

func addAppNamespaces(appNamespaces *[]*string, namespaces []*string) {
	newAppNamespaces := *appNamespaces

	for _, ns := range namespaces {
		isExisting := false

		for _, existingNS := range newAppNamespaces {
			if ns == existingNS {
				isExisting = true
				break
			}
		}

		if !isExisting {
			newAppNamespaces = append(newAppNamespaces, ns)
		}
	}

	appNamespaces = &newAppNamespaces
}

// IsTillerDeployed determines if we could connect to a tiller server
func IsTillerDeployed(kubectlClient *kubernetes.Clientset, tillerConfig *v1.TillerConfig) bool {
	tillerNamespace := *tillerConfig.Release.Namespace
	deployment, err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Get(TillerDeploymentName, metav1.GetOptions{})

	if err != nil {
		return false
	}

	if deployment == nil {
		return false
	}

	return true
}

// DeleteTiller clears the tiller server, the service account and role binding
func DeleteTiller(kubectlClient *kubernetes.Clientset, tillerConfig *v1.TillerConfig) error {
	tillerNamespace := *tillerConfig.Release.Namespace
	errs := make([]error, 0, 1)
	propagationPolicy := metav1.DeletePropagationForeground

	err := kubectlClient.ExtensionsV1beta1().Deployments(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
		errs = append(errs, err)
	}

	err = kubectlClient.CoreV1().Services(tillerNamespace).Delete(TillerDeploymentName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
		errs = append(errs, err)
	}

	err = kubectlClient.CoreV1().ServiceAccounts(tillerNamespace).Delete(tillerServiceAccountName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
	if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
		errs = append(errs, err)
	}

	roleNamespace := append(*tillerConfig.AppNamespaces, &tillerNamespace)
	for _, appNamespace := range roleNamespace {
		err = kubectlClient.RbacV1beta1().Roles(*appNamespace).Delete(tillerRoleName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
			errs = append(errs, err)
		}

		err = kubectlClient.RbacV1beta1().RoleBindings(*appNamespace).Delete(tillerRoleName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
			errs = append(errs, err)
		}

		err = kubectlClient.RbacV1beta1().Roles(*appNamespace).Delete(tillerRoleManagerName, &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
			errs = append(errs, err)
		}

		err = kubectlClient.RbacV1beta1().RoleBindings(*appNamespace).Delete(tillerRoleManagerName+"-binding", &metav1.DeleteOptions{PropagationPolicy: &propagationPolicy})
		if err != nil && strings.HasSuffix(err.Error(), "not found") == false {
			errs = append(errs, err)
		}
	}

	// Merge errors
	errorText := ""

	for _, value := range errs {
		errorText += value.Error() + "\n"
	}

	if errorText == "" {
		return nil
	}
	return errors.New(errorText)
}

// func (helmClientWrapper *HelmClientWrapper) ensureAuth(namespace string) error {
//	 return ensureRoleBinding(helmClientWrapper.kubectl, tillerRoleName, namespace, helmClientWrapper.Settings.TillerNamespace, defaultPolicyRules)
// }

func ensureRoleBinding(kubectlClient *kubernetes.Clientset, tillerConfig *v1.TillerConfig, name, namespace string, rules []k8sv1beta1.PolicyRule, subjects []k8sv1beta1.Subject) error {
	role := &k8sv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}
	rolebinding := &k8sv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-binding",
			Namespace: namespace,
		},
		Subjects: subjects,
		RoleRef: k8sv1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}

	_, roleErr := kubectlClient.RbacV1beta1().Roles(namespace).Create(role)
	if roleErr != nil && alreadyExistsRegexp.Match([]byte(roleErr.Error())) == false {
		return roleErr
	}

	_, roleBindingErr := kubectlClient.RbacV1beta1().RoleBindings(namespace).Create(rolebinding)
	if roleBindingErr != nil && alreadyExistsRegexp.Match([]byte(roleBindingErr.Error())) == false {
		return roleBindingErr
	}

	return nil
}

func (helmClientWrapper *HelmClientWrapper) updateRepos() error {
	allRepos, err := repo.LoadRepositoriesFile(helmClientWrapper.Settings.Home.RepositoryFile())
	if err != nil {
		return err
	}

	repos := []*repo.ChartRepository{}

	for _, repoData := range allRepos.Repositories {
		repo, err := repo.NewChartRepository(repoData, getter.All(*helmClientWrapper.Settings))
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

			err := re.DownloadIndexFile(helmClientWrapper.Settings.Home.String())
			if err != nil {
				log.With(err).Error("Unable to download repo index")

				//TODO
			}
		}(re)
	}

	wg.Wait()

	return nil
}

// ReleaseExists checks if the given release name exists
func (helmClientWrapper *HelmClientWrapper) ReleaseExists(releaseName string) (bool, error) {
	_, err := helmClientWrapper.Client.ReleaseHistory(releaseName, k8shelm.WithMaxHistory(1))
	if err != nil {
		if strings.Contains(err.Error(), helmstoragedriver.ErrReleaseNotFound(releaseName).Error()) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// InstallChartByPath installs the given chartpath und the releasename in the releasenamespace
func (helmClientWrapper *HelmClientWrapper) InstallChartByPath(releaseName string, releaseNamespace string, chartPath string, values *map[interface{}]interface{}) (*hapi_release5.Release, error) {
	chart, err := helmchartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	chartDependencies := chart.GetDependencies()

	if len(chartDependencies) > 0 {
		_, err = helmchartutil.LoadRequirements(chart)

		if err != nil {
			return nil, err
		}
		chartDownloader := &helmdownloader.Manager{
			/*		Out:        i.out,
					ChartPath:  i.chartPath,
					HelmHome:   settings.Home,
					Keyring:    defaultKeyring(),
					SkipUpdate: false,
					Getters:    getter.All(settings),
			*/
		}
		err = chartDownloader.Update()

		if err != nil {
			return nil, err
		}
		chart, err = helmchartutil.Load(chartPath)

		if err != nil {
			return nil, err
		}
	}
	releaseExists, err := helmClientWrapper.ReleaseExists(releaseName)

	if err != nil {
		return nil, err
	}

	deploymentTimeout := int64(10 * 60)
	overwriteValues := []byte("")

	if values != nil {
		unmarshalledValues, err := yaml.Marshal(values)

		if err != nil {
			return nil, err
		}
		overwriteValues = unmarshalledValues
	}

	var release *hapi_release5.Release

	if releaseExists {
		upgradeResponse, err := helmClientWrapper.Client.UpdateRelease(
			releaseName,
			chartPath,
			k8shelm.UpgradeTimeout(deploymentTimeout),
			k8shelm.UpdateValueOverrides(overwriteValues),
			k8shelm.ReuseValues(false),
			k8shelm.UpgradeWait(true),
		)

		if err != nil {
			return nil, err
		}

		release = upgradeResponse.GetRelease()
	} else {
		installResponse, err := helmClientWrapper.Client.InstallReleaseFromChart(
			chart,
			releaseNamespace,
			k8shelm.InstallTimeout(deploymentTimeout),
			k8shelm.ValueOverrides(overwriteValues),
			k8shelm.ReleaseName(releaseName),
			k8shelm.InstallReuseName(false),
			k8shelm.InstallWait(true),
		)

		if err != nil {
			return nil, err
		}

		release = installResponse.GetRelease()
	}
	return release, nil
}

// InstallChartByName installs the given chart by name under the releasename in the releasenamespace
func (helmClientWrapper *HelmClientWrapper) InstallChartByName(releaseName string, releaseNamespace string, chartName string, chartVersion string, values *map[interface{}]interface{}) (*hapi_release5.Release, error) {
	if len(chartVersion) == 0 {
		chartVersion = ">0.0.0-0"
	}
	getter := getter.All(*helmClientWrapper.Settings)
	chartDownloader := downloader.ChartDownloader{
		HelmHome: helmClientWrapper.Settings.Home,
		Out:      os.Stdout,
		Getters:  getter,
		Verify:   downloader.VerifyNever,
	}
	os.MkdirAll(helmClientWrapper.Settings.Home.Archive(), os.ModePerm)

	chartPath, _, err := chartDownloader.DownloadTo(chartName, chartVersion, helmClientWrapper.Settings.Home.Archive())
	if err != nil {
		return nil, err
	}

	return helmClientWrapper.InstallChartByPath(releaseName, releaseNamespace, chartPath, values)
}

// stringArraySorter
type stringArraySorter [][]string

// Len returns the length of this scoreSorter.
func (s stringArraySorter) Len() int { return len(s) }

// Swap performs an in-place swap.
func (s stringArraySorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less compares a to b, and returns true if a is less than b.
func (s stringArraySorter) Less(a, b int) bool { return s[a][0] < s[b][0] }

// PrintAllAvailableCharts prints all available charts
func (helmClientWrapper *HelmClientWrapper) PrintAllAvailableCharts() {
	var values stringArraySorter
	var header = []string{
		"NAME",
		"CHART VERSION",
		"APP VERSION",
		"DESCRIPTION",
	}

	allRepos, err := repo.LoadRepositoriesFile(helmClientWrapper.Settings.Home.RepositoryFile())
	if err != nil {
		log.Fatal(err)
	}

	for _, re := range allRepos.Repositories {
		n := re.Name
		f := helmClientWrapper.Settings.Home.CacheIndex(n)

		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			continue
		}

		// Sort versions
		ind.SortEntries()

		for _, versions := range ind.Entries {
			if len(versions) == 0 {
				continue
			}

			description := versions[0].Description
			if len(description) > 45 {
				description = description[:45] + "..."
			}

			values = append(values, []string{
				versions[0].GetName(),
				versions[0].GetVersion(),
				versions[0].GetAppVersion(),
				description,
			})
		}
	}

	sort.Sort(values)
	log.PrintTable(header, values)
}

// SearchChart searches the chart name in all repositories
func (helmClientWrapper *HelmClientWrapper) SearchChart(chartName, chartVersion, appVersion string) (*repo.Entry, *repo.ChartVersion, error) {
	allRepos, err := repo.LoadRepositoriesFile(helmClientWrapper.Settings.Home.RepositoryFile())
	if err != nil {
		return nil, nil, err
	}

	for _, re := range allRepos.Repositories {
		n := re.Name
		f := helmClientWrapper.Settings.Home.CacheIndex(n)

		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			continue
		}

		// Sort versions
		ind.SortEntries()

		// Check if chart exists
		if versions, ok := ind.Entries[chartName]; ok {
			if len(versions) == 0 {
				// Skip chart names that have zero releases.
				continue
			}

			if chartVersion != "" {
				for _, version := range versions {
					if version.GetVersion() == chartVersion {
						return re, version, nil
					}
				}

				return nil, nil, fmt.Errorf("Chart %s with chart version %s not found", chartName, chartVersion)
			}

			if appVersion != "" {
				for _, version := range versions {
					if version.GetAppVersion() == appVersion {
						return re, version, nil
					}
				}

				return nil, nil, fmt.Errorf("Chart %s with app version %s not found", chartName, appVersion)
			}

			return re, versions[0], nil
		}
	}

	return nil, nil, fmt.Errorf("Chart %s not found", chartName)
}

// BuildDependencies builds the dependencies
func (helmClientWrapper *HelmClientWrapper) BuildDependencies(chartPath string) error {
	man := &helmdownloader.Manager{
		Out:       ioutil.Discard,
		ChartPath: chartPath,
		HelmHome:  helmClientWrapper.Settings.Home,
		Getters:   getter.All(*helmClientWrapper.Settings),
	}

	return man.Build()
}

// DeleteRelease deletes a helm release and optionally purges it
func (helmClientWrapper *HelmClientWrapper) DeleteRelease(releaseName string, purge bool) (*rls.UninstallReleaseResponse, error) {
	return helmClientWrapper.Client.DeleteRelease(releaseName, k8shelm.DeletePurge(purge))
}
