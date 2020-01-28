package v3

import (
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Template runs `helm template`
func (client *v3Client) Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	if releaseNamespace == "" {
		releaseNamespace = client.kubectl.Namespace()
	}

	var (
		settings  = cli.New()
		chartName = strings.TrimSpace(helmConfig.Chart.Name)
		chartRepo = helmConfig.Chart.RepoURL
		getter    = genericclioptions.NewConfigFlags(true)
		cfg       = &action.Configuration{
			RESTClientGetter: getter,
			Releases:         storage.Init(driver.NewMemory()),
			KubeClient:       kube.New(getter),
			Log: func(msg string, params ...interface{}) {
				// We don't log helm messages
			},
		}
	)

	if strings.HasPrefix(chartName, "stable/") && chartRepo == "" {
		chartName = chartName[7:]
		chartRepo = stableChartRepo
	}

	// Get values
	vals := yamlutil.Convert(values).(map[string]interface{})

	// If a release does not exist, install it. If another error occurs during
	// the check, ignore the error and continue with the upgrade.
	instClient := action.NewInstall(cfg)
	instClient.ChartPathOptions.Version = helmConfig.Chart.Version
	instClient.ChartPathOptions.RepoURL = chartRepo
	instClient.ChartPathOptions.Username = helmConfig.Chart.Username
	instClient.ChartPathOptions.Password = helmConfig.Chart.Password
	instClient.DryRun = true
	instClient.ClientOnly = true
	instClient.DisableHooks = helmConfig.DisableHooks
	if helmConfig.Timeout != nil {
		instClient.Timeout = time.Duration(*helmConfig.Timeout)
	}
	instClient.Wait = helmConfig.Wait || helmConfig.Atomic
	instClient.Namespace = releaseNamespace
	instClient.Atomic = helmConfig.Atomic

	chartPath, err := instClient.ChartPathOptions.LocateChart(chartName, settings)
	if err != nil {
		return "", err
	}

	rel, err := install(releaseName, releaseNamespace, chartPath, instClient, vals, settings)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(rel.Manifest), nil
}
