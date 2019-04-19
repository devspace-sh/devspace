package helm

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/devspace/analyze"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	yaml "gopkg.in/yaml.v2"
	helmchartutil "k8s.io/helm/pkg/chartutil"
	helmdownloader "k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	k8shelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release5 "k8s.io/helm/pkg/proto/hapi/release"
)

// DeploymentTimeout is the timeout to wait for helm to deploy
const DeploymentTimeout = int64(80)

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
func (helmClientWrapper *ClientWrapper) InstallChartByPath(releaseName, releaseNamespace, chartPath string, values *map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*hapi_release5.Release, error) {
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
	} else if err != helmchartutil.ErrRequirementsNotFound {
		return nil, fmt.Errorf("cannot load requirements: %v", err)
	}

	releaseExists, err := helmClientWrapper.ReleaseExists(releaseName)
	if err != nil {
		return nil, err
	}

	overwriteValues := []byte("")

	if values != nil {
		unmarshalledValues, err := yaml.Marshal(values)

		if err != nil {
			return nil, err
		}
		overwriteValues = unmarshalledValues
	}

	// Set wait and timeout
	waitTimeout := DeploymentTimeout
	if helmConfig.Timeout != nil {
		waitTimeout = *helmConfig.Timeout
	}

	wait := true
	if helmConfig.Wait != nil {
		wait = *helmConfig.Wait
	}

	rollback := true
	if helmConfig.Rollback != nil {
		rollback = *helmConfig.Rollback
	}

	if releaseExists {
		upgradeResponse, err := helmClientWrapper.Client.UpdateRelease(
			releaseName,
			chartPath,
			k8shelm.UpgradeWait(wait),
			k8shelm.UpgradeTimeout(waitTimeout),
			k8shelm.UpdateValueOverrides(overwriteValues),
			k8shelm.ReuseValues(false),
			k8shelm.UpgradeForce(ptr.ReverseBool(helmConfig.Force)),
		)

		if err != nil {
			err = helmClientWrapper.analyzeError(fmt.Errorf("helm upgrade: %v", err), releaseNamespace)
			if err != nil {
				if rollback {
					log.Warn("Try to roll back back chart because of previous error")
					_, rollbackError := helmClientWrapper.Client.RollbackRelease(releaseName, k8shelm.RollbackTimeout(180))
					if rollbackError != nil {
						return nil, err
					}
				}

				return nil, err
			}

			return nil, nil
		}

		return upgradeResponse.GetRelease(), nil
	}

	installResponse, err := helmClientWrapper.Client.InstallReleaseFromChart(
		chart,
		releaseNamespace,
		k8shelm.InstallWait(wait),
		k8shelm.InstallTimeout(waitTimeout),
		k8shelm.ValueOverrides(overwriteValues),
		k8shelm.ReleaseName(releaseName),
		k8shelm.InstallReuseName(true),
	)
	if err != nil {
		err = helmClientWrapper.analyzeError(fmt.Errorf("helm install: %v", err), releaseNamespace)
		if err != nil {
			if rollback {
				// Try to delete and ignore errors, because otherwise we have a broken release laying around and always get the no deployed resources error
				helmClientWrapper.DeleteRelease(releaseName, true)
			}

			return nil, err
		}

		return nil, nil
	}

	return installResponse.GetRelease(), nil
}

// analyzeError calls analyze and tries to find the issue
func (helmClientWrapper *ClientWrapper) analyzeError(srcErr error, releaseNamespace string) error {
	errMessage := srcErr.Error()

	// Only check if the error is time out
	if strings.Index(errMessage, "timed out waiting") != -1 {
		config, err := kubectl.GetClientConfig()
		if err != nil {
			log.Warnf("Error loading kubectl config: %v", err)
			return srcErr
		}

		report, err := analyze.CreateReport(config, releaseNamespace, false)
		if err != nil {
			log.Warnf("Error creating analyze report: %v", err)
			return srcErr
		}
		if len(report) == 0 {
			return nil
		}

		return errors.New(analyze.ReportToString(report))
	}

	return srcErr
}

// InstallChart installs the given chart by name under the releasename in the releasenamespace
func (helmClientWrapper *ClientWrapper) InstallChart(releaseName string, releaseNamespace string, values *map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*hapi_release5.Release, error) {
	chart := helmConfig.Chart
	chartPath, err := locateChartPath(helmClientWrapper.Settings, ptr.ReverseString(chart.RepoURL), ptr.ReverseString(chart.Username), ptr.ReverseString(chart.Password), ptr.ReverseString(chart.Name), ptr.ReverseString(chart.Version), false, "", "", "", "")
	if err != nil {
		return nil, errors.Wrap(err, "locate chart path")
	}

	return helmClientWrapper.InstallChartByPath(releaseName, releaseNamespace, chartPath, values, helmConfig)
}
