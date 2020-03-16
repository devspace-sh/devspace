package v3

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"

	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type v3Client struct {
	helmDriver string
	kubectl kubectl.Client
	log     log.Logger
}

const stableChartRepo = "https://kubernetes-charts.storage.googleapis.com"

// NewClient creates a new helm v3 client
func NewClient(kubeClient kubectl.Client, helmDriver string, log log.Logger) (types.Client, error) {
	return &v3Client{
		helmDriver: helmDriver,
		kubectl: kubeClient,
		log:     log,
	}, nil
}

func (client *v3Client) initHelmConfig(namespace string) (*action.Configuration, error) {
	getter := genericclioptions.NewConfigFlags(true)

	var store *storage.Storage
	if client.kubectl != nil {
		if namespace == "" {
			namespace = client.kubectl.Namespace()
		}

		getter.Namespace = ptr.String(namespace)
		getter.Context = ptr.String(client.kubectl.CurrentContext())

		switch client.helmDriver {
		case "secret", "secrets", "":
			d := driver.NewSecrets(client.kubectl.KubeClient().CoreV1().Secrets(namespace))
			store = storage.Init(d)
		case "configmap", "configmaps":
			d := driver.NewConfigMaps(client.kubectl.KubeClient().CoreV1().ConfigMaps(namespace))
			store = storage.Init(d)
		case "memory":
			d := driver.NewMemory()
			store = storage.Init(d)
		default:
			// Not sure what to do here.
			return nil, errors.New("Unknown driver in HELM_DRIVER: " + client.helmDriver)
		}
	} else {
		d := driver.NewMemory()
		store = storage.Init(d)
	}

	return &action.Configuration{
		RESTClientGetter: getter,
		Releases:         store,
		KubeClient:       kube.New(getter),
		Log: func(msg string, params ...interface{}) {
			// We don't log helm messages
			// log.Infof(msg, params...)
		},
	}, nil
}

func (client *v3Client) InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
	if releaseNamespace == "" {
		releaseNamespace = client.kubectl.Namespace()
	}

	// Init the client
	cfg, err := client.initHelmConfig(releaseNamespace)
	if err != nil {
		return nil, err
	}

	var (
		settings  = cli.New()
		chartName = strings.TrimSpace(helmConfig.Chart.Name)
		chartRepo = helmConfig.Chart.RepoURL
	)

	// makes sure repos are not being updated
	settings.RepositoryConfig = ""

	if strings.HasPrefix(chartName, "stable/") && chartRepo == "" {
		chartName = chartName[7:]
		chartRepo = stableChartRepo
	}

	upgrade := action.NewUpgrade(cfg)
	upgrade.Install = true
	upgrade.Namespace = releaseNamespace

	upgrade.Force = helmConfig.Force
	upgrade.DisableHooks = helmConfig.DisableHooks
	upgrade.Recreate = helmConfig.Recreate
	upgrade.CleanupOnFail = helmConfig.CleanupOnFail
	upgrade.ReuseValues = false
	upgrade.Atomic = helmConfig.Atomic
	upgrade.Wait = helmConfig.Wait || helmConfig.Atomic
	if helmConfig.Timeout != nil {
		upgrade.Timeout = time.Duration(*helmConfig.Timeout)
	}

	upgrade.ChartPathOptions.Version = helmConfig.Chart.Version
	upgrade.ChartPathOptions.RepoURL = chartRepo
	upgrade.ChartPathOptions.Username = helmConfig.Chart.Username
	upgrade.ChartPathOptions.Password = helmConfig.Chart.Password

	chartPath, err := upgrade.ChartPathOptions.LocateChart(chartName, settings)
	if err != nil {
		return nil, err
	}

	vals := yamlutil.Convert(values).(map[string]interface{})
	if upgrade.Install {
		// If a release does not exist, install it. If another error occurs during
		// the check, ignore the error and continue with the upgrade.
		histClient := action.NewHistory(cfg)
		histClient.Max = 1
		if _, err := histClient.Run(releaseName); err == driver.ErrReleaseNotFound {
			instClient := action.NewInstall(cfg)
			instClient.ChartPathOptions = upgrade.ChartPathOptions
			instClient.DryRun = upgrade.DryRun
			instClient.DisableHooks = upgrade.DisableHooks
			instClient.Timeout = upgrade.Timeout
			instClient.Wait = upgrade.Wait
			instClient.Devel = upgrade.Devel
			instClient.Namespace = upgrade.Namespace
			instClient.Atomic = upgrade.Atomic

			rel, err := install(releaseName, releaseNamespace, chartPath, instClient, vals, settings)
			if err != nil {
				return nil, err
			}

			return &types.Release{
				Name:         rel.Name,
				Namespace:    rel.Namespace,
				Status:       rel.Info.Status.String(),
				LastDeployed: rel.Info.LastDeployed.Time,
			}, nil
		}
	}

	// Check chart dependencies to make sure all are present in /charts
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, err
	}
	if req := ch.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(ch, req); err != nil {
			return nil, err
		}
	}

	rel, err := upgrade.Run(releaseName, ch, vals)
	if err != nil {
		return nil, errors.Wrap(err, "UPGRADE FAILED")
	}

	return &types.Release{
		Name:         rel.Name,
		Namespace:    rel.Namespace,
		Status:       rel.Info.Status.String(),
		LastDeployed: rel.Info.LastDeployed.Time,
	}, nil
}

func install(releaseName string, releaseNamespace string, chartName string, install *action.Install, values map[string]interface{}, settings *cli.EnvSettings) (*release.Release, error) {
	if install.Version == "" && install.Devel {
		install.Version = ">0.0.0-0"
	}

	name, chart, err := install.NameAndChart([]string{releaseName, chartName})
	if err != nil {
		return nil, err
	}
	install.ReleaseName = name

	cp, err := install.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return nil, err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if install.DependencyUpdate {
				man := &downloader.Manager{
					Out:              ioutil.Discard,
					ChartPath:        cp,
					Keyring:          install.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(settings),
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	install.Namespace = releaseNamespace
	return install.Run(chartRequested, values)
}

// isChartInstallable validates if a chart can be installed
//
// Application chart type is only installable
func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func (client *v3Client) DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error {
	cfg, err := client.initHelmConfig(releaseNamespace)
	if err != nil {
		return err
	}

	_, err = action.NewUninstall(cfg).Run(releaseName)
	return err
}

func (client *v3Client) ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error) {
	cfg, err := client.initHelmConfig("")
	if err != nil {
		return nil, err
	}

	list, err := action.NewList(cfg).Run()
	if err != nil {
		return nil, err
	}

	retReleases := make([]*types.Release, len(list))
	for i, release := range list {
		retReleases[i] = &types.Release{
			Name:         release.Name,
			Namespace:    release.Namespace,
			Status:       release.Info.Status.String(),
			LastDeployed: release.Info.LastDeployed.Time,
		}
	}

	return retReleases, nil
}
