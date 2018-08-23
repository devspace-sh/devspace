package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/covexo/devspace/pkg/util/logutil"

	yaml "gopkg.in/yaml.v2"

	"github.com/covexo/devspace/pkg/util/fsutil"

	helminstaller "k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/repo"

	"k8s.io/client-go/kubernetes"

	"github.com/Sirupsen/logrus"
	"github.com/covexo/devspace/pkg/devspace/clients/kubectl"
	"github.com/covexo/devspace/pkg/devspace/config"
	"github.com/covexo/devspace/pkg/devspace/config/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8sv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmchartutil "k8s.io/helm/pkg/chartutil"
	helmdownloader "k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/helm"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/helm/portforwarder"
	helmstoragedriver "k8s.io/helm/pkg/storage/driver"
)

type HelmClientWrapper struct {
	Client   *helm.Client
	Settings *helmenvironment.EnvSettings
	kubectl  *kubernetes.Clientset
}

const tillerServiceAccountName = "devspace-tiller"
const tillerRoleName = "devspace-tiller"
const tillerDeploymentName = "tiller-deploy"

var privateConfig = &v1.PrivateConfig{}
var log *logrus.Logger
var defaultPolicyRules = []k8sv1beta1.PolicyRule{
	k8sv1beta1.PolicyRule{
		APIGroups: []string{
			k8sv1beta1.APIGroupAll,
			"extensions",
			"apps",
		},
		Resources: []string{k8sv1beta1.ResourceAll},
		Verbs:     []string{k8sv1beta1.ResourceAll},
	},
}

func NewClient(kubectlClient *kubernetes.Clientset, upgradeTiller bool) (*HelmClientWrapper, error) {
	log = logutil.GetLogger("default", true)
	config.LoadConfig(privateConfig)

	kubeconfig, err := kubectl.GetClientConfig()

	if err != nil {
		return nil, err
	}

	tillerErr := ensureTiller(kubectlClient, upgradeTiller)

	if tillerErr != nil {
		return nil, tillerErr
	}
	var tunnelErr error
	var tunnel *kube.Tunnel

	tunnelWaitTime := 2 * 60 * time.Second
	tunnelCheckInterval := 5 * time.Second

	for tunnelWaitTime > 0 {
		tunnel, tunnelErr = portforwarder.New(privateConfig.Cluster.TillerNamespace, kubectlClient, kubeconfig)

		if tunnelErr == nil {
			break
		}
		log.Info("Waiting for port forwarding to start")

		tunnelWaitTime = tunnelWaitTime - tunnelCheckInterval
	}

	if tunnelErr != nil {
		return nil, tunnelErr
	}
	helmOptions := []helm.Option{
		helm.Host("127.0.0.1:" + strconv.Itoa(tunnel.Local)),
	}
	var tillerError error

	client := helm.NewClient(helmOptions...)
	helmWaitTime := 2 * 60 * time.Second
	helmCheckInterval := 5 * time.Second

	for helmWaitTime > 0 {
		_, tillerError := client.GetVersion()

		if tillerError == nil {
			break
		}
		log.Info("Waiting for helm client getting connection")

		helmWaitTime = helmWaitTime - helmCheckInterval
	}

	if tillerError != nil {
		return nil, tillerError
	}
	helmHomePath := os.ExpandEnv("$HOME/.devspace/helm")
	_, helmHomeNotFoundErr := os.Stat(helmHomePath)

	if helmHomeNotFoundErr != nil {
		os.MkdirAll(helmHomePath, os.ModePerm)
		helmHomeTemplates := filepath.Join(fsutil.GetCurrentGofileDir(), "assets")
		copyErr := fsutil.Copy(helmHomeTemplates, helmHomePath)

		if copyErr != nil {
			return nil, copyErr
		}
	}
	wrapper := &HelmClientWrapper{
		Client: client,
		Settings: &helmenvironment.EnvSettings{
			Home: helmpath.Home(helmHomePath),
		},
		kubectl: kubectlClient,
	}

	if helmHomeNotFoundErr != nil {
		wrapper.updateRepos()
	}
	return wrapper, nil
}

func ensureTiller(kubectlClient *kubernetes.Clientset, upgrade bool) error {
	tillerSA := &k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tillerServiceAccountName,
			Namespace: privateConfig.Cluster.TillerNamespace,
		},
	}
	tillerOptions := &helminstaller.Options{
		Namespace:      privateConfig.Cluster.TillerNamespace,
		MaxHistory:     10,
		ImageSpec:      "gcr.io/kubernetes-helm/tiller:v2.9.1",
		ServiceAccount: tillerSA.ObjectMeta.Name,
	}
	_, tillerCheckErr := kubectlClient.ExtensionsV1beta1().Deployments(privateConfig.Cluster.TillerNamespace).Get(tillerDeploymentName, metav1.GetOptions{})

	if tillerCheckErr != nil {
		log.Info("Installing Tiller server")

		_, saNotFoundErr := kubectlClient.CoreV1().ServiceAccounts(tillerSA.Namespace).Get(tillerSA.Name, metav1.GetOptions{})

		if saNotFoundErr != nil {
			_, saCreateErr := kubectlClient.CoreV1().ServiceAccounts(tillerSA.Namespace).Create(tillerSA)

			if saCreateErr != nil {
				return saCreateErr
			}
		}
		roleBindingErr := ensureRoleBinding(kubectlClient, "tiller-config-manager", privateConfig.Cluster.TillerNamespace, privateConfig.Cluster.TillerNamespace, []k8sv1beta1.PolicyRule{
			k8sv1beta1.PolicyRule{
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
		})

		if roleBindingErr != nil {
			return roleBindingErr
		}
		helminstaller.Install(kubectlClient, tillerOptions)

		roleBindingErr = ensureRoleBinding(kubectlClient, tillerRoleName, privateConfig.Release.Namespace, privateConfig.Cluster.TillerNamespace, defaultPolicyRules)

		if roleBindingErr != nil {
			return roleBindingErr
		}
	} else if upgrade {
		log.Info("Upgrading Tiller server")

		tillerOptions.ImageSpec = ""

		helminstaller.Upgrade(kubectlClient, tillerOptions)
	}
	tillerWaitingTime := 2 * 60 * time.Second
	tillerCheckInterval := 5 * time.Second

	for tillerWaitingTime > 0 {
		tillerDeployment, _ := kubectlClient.ExtensionsV1beta1().Deployments(privateConfig.Cluster.TillerNamespace).Get(tillerDeploymentName, metav1.GetOptions{})

		if tillerDeployment.Status.ReadyReplicas == tillerDeployment.Status.Replicas {
			break
		}
		log.Info("Waiting for Tiller server to start")

		time.Sleep(tillerCheckInterval)

		tillerWaitingTime = tillerWaitingTime - tillerCheckInterval
	}
	return nil
}

func (helmClientWrapper *HelmClientWrapper) EnsureAuth(namespace string) error {
	return ensureRoleBinding(helmClientWrapper.kubectl, tillerRoleName, namespace, helmClientWrapper.Settings.TillerNamespace, defaultPolicyRules)
}

func ensureRoleBinding(kubectlClient *kubernetes.Clientset, name, namespace string, tillerNamespace string, rules []k8sv1beta1.PolicyRule) error {
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
		Subjects: []k8sv1beta1.Subject{
			k8sv1beta1.Subject{
				Kind:      k8sv1beta1.ServiceAccountKind,
				Name:      tillerServiceAccountName,
				Namespace: tillerNamespace,
			},
		},
		RoleRef: k8sv1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	kubectlClient.RbacV1beta1().Roles(namespace).Create(role)
	kubectlClient.RbacV1beta1().RoleBindings(namespace).Create(rolebinding)

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
				fmt.Println("upanble to download repo index")
				fmt.Println(err)
				//TODO
			}
		}(re)
	}
	wg.Wait()

	return nil
}

func (helmClientWrapper *HelmClientWrapper) ReleaseExists(releaseName string) (bool, error) {
	_, releaseHistoryErr := helmClientWrapper.Client.ReleaseHistory(releaseName, helm.WithMaxHistory(1))

	if releaseHistoryErr != nil {
		if strings.Contains(releaseHistoryErr.Error(), helmstoragedriver.ErrReleaseNotFound(releaseName).Error()) {
			return false, nil
		}
		return false, releaseHistoryErr
	}
	return true, nil
}

func (helmClientWrapper *HelmClientWrapper) InstallChartByPath(releaseName string, releaseNamespace string, chartPath string, values *map[interface{}]interface{}) error {
	chart, chartLoadingErr := helmchartutil.Load(chartPath)

	if chartLoadingErr != nil {
		return chartLoadingErr
	}
	chartDependencies := chart.GetDependencies()

	if len(chartDependencies) > 0 {
		_, chartReqError := helmchartutil.LoadRequirements(chart)

		if chartReqError != nil {
			return chartReqError
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
		chartDownloadErr := chartDownloader.Update()

		if chartDownloadErr != nil {
			return chartDownloadErr
		}
		chart, chartLoadingErr = helmchartutil.Load(chartPath)

		if chartLoadingErr != nil {
			return chartLoadingErr
		}
	}
	releaseExists, releaseExistsErr := helmClientWrapper.ReleaseExists(releaseName)

	if releaseExistsErr != nil {
		return releaseExistsErr
	}
	deploymentTimeout := int64(10 * 60)
	overwriteValues := []byte("")

	if values != nil {
		unmarshalledValues, yamlErr := yaml.Marshal(*values)

		if yamlErr != nil {
			return yamlErr
		}
		overwriteValues = unmarshalledValues
	}

	if releaseExists {
		_, releaseUpgradeErr := helmClientWrapper.Client.UpdateRelease(
			releaseName,
			chartPath,
			helm.UpgradeTimeout(deploymentTimeout),
			helm.UpdateValueOverrides(overwriteValues),
			helm.ReuseValues(false),
			helm.UpgradeWait(true),
		)

		if releaseUpgradeErr != nil {
			return releaseUpgradeErr
		}
	} else {
		_, releaseInstallErr := helmClientWrapper.Client.InstallReleaseFromChart(
			chart,
			releaseNamespace,
			helm.InstallTimeout(deploymentTimeout),
			helm.ValueOverrides(overwriteValues),
			helm.ReleaseName(releaseName),
			helm.InstallReuseName(false),
			helm.InstallWait(true),
		)

		if releaseInstallErr != nil {
			return releaseInstallErr
		}
	}
	return nil
}

func (helmClientWrapper *HelmClientWrapper) InstallChartByName(releaseName string, releaseNamespace string, chartName string, chartVersion string, values *map[interface{}]interface{}) error {
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

	chartPath, _, chartDownloadErr := chartDownloader.DownloadTo(chartName, chartVersion, helmClientWrapper.Settings.Home.Archive())

	if chartDownloadErr != nil {
		return chartDownloadErr
	}
	return helmClientWrapper.InstallChartByPath(releaseName, releaseNamespace, chartPath, values)
}
