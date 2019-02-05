package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/covexo/devspace/pkg/devspace/config/configutil"

	yaml "gopkg.in/yaml.v2"
	helmchartutil "k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	helmdownloader "k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	k8shelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release5 "k8s.io/helm/pkg/proto/hapi/release"
)

func checkDependencies(ch *chart.Chart, reqs *helmchartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}

// InstallChartByPath installs the given chartpath und the releasename in the releasenamespace
func (helmClientWrapper *ClientWrapper) InstallChartByPath(releaseName, releaseNamespace, chartPath string, values *map[string]interface{}, wait bool) (*hapi_release5.Release, error) {
	if releaseNamespace == "" {
		config := configutil.GetConfig()

		// Use default namespace here
		defaultNamespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return nil, err
		}

		releaseNamespace = defaultNamespace
	}

	chart, err := helmchartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	if req, err := helmchartutil.LoadRequirements(chart); err == nil {
		// If checkDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/kubernetes/helm/issues/2209
		if err := checkDependencies(chart, req); err != nil {
			man := &helmdownloader.Manager{
				Out:       ioutil.Discard,
				ChartPath: chartPath,
				HelmHome:  helmClientWrapper.Settings.Home,
				Getters:   getter.All(*helmClientWrapper.Settings),
			}
			if err := man.Update(); err != nil {
				return nil, err
			}

			// Update all dependencies which are present in /charts.
			chart, err = helmchartutil.Load(chartPath)
			if err != nil {
				return nil, err
			}
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

	if releaseExists {
		waitOption := k8shelm.UpgradeWait(wait)

		upgradeResponse, err := helmClientWrapper.Client.UpdateRelease(
			releaseName,
			chartPath,
			k8shelm.UpgradeTimeout(deploymentTimeout),
			k8shelm.UpdateValueOverrides(overwriteValues),
			k8shelm.ReuseValues(false),
			k8shelm.UpgradeForce(true),
			waitOption,
		)

		if err != nil {
			// Delete release and redeploy
			if strings.Index(err.Error(), "cannot re-use a name that is still in use") != -1 {
				// Try to delete and ignore errors, because otherwise we have a broken release laying around and always get the no deployed resources error
				_, err := helmClientWrapper.DeleteRelease(releaseName, true)
				if err != nil {
					return nil, fmt.Errorf("Error deleting release %s: %v", releaseName, err)
				}
			} else {
				return nil, err
			}
		} else {
			return upgradeResponse.GetRelease(), nil
		}
	}

	waitOption := k8shelm.InstallWait(wait)
	installResponse, err := helmClientWrapper.Client.InstallReleaseFromChart(
		chart,
		releaseNamespace,
		k8shelm.InstallTimeout(deploymentTimeout),
		k8shelm.ValueOverrides(overwriteValues),
		k8shelm.ReleaseName(releaseName),
		k8shelm.InstallReuseName(true),
		waitOption,
	)
	if err != nil {
		// Try to delete and ignore errors, because otherwise we have a broken release laying around and always get the no deployed resources error
		helmClientWrapper.DeleteRelease(releaseName, true)

		return nil, err
	}

	return installResponse.GetRelease(), nil
}

// InstallChartByName installs the given chart by name under the releasename in the releasenamespace
func (helmClientWrapper *ClientWrapper) InstallChartByName(releaseName string, releaseNamespace string, chartName string, chartVersion string, values *map[string]interface{}, wait bool) (*hapi_release5.Release, error) {
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

	return helmClientWrapper.InstallChartByPath(releaseName, releaseNamespace, chartPath, values, wait)
}
