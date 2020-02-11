/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/helm/pkg/chartutil"
	helmenvironment "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/manifest"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/timeconv"
)

// Template executes a `helm template`
func (client *client) Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	helmHomePath := homeDir + "/.helm"
	chartPath, err := locateChartPath(&helmenvironment.EnvSettings{Home: helmpath.Home(helmHomePath)}, helmConfig.Chart.RepoURL, helmConfig.Chart.Username, helmConfig.Chart.Password, helmConfig.Chart.Name, helmConfig.Chart.Version, false, "", "", "", "")
	if err != nil {
		return "", errors.Wrap(err, "locate chart path")
	}

	unmarshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return "", err
	}

	kubeVersion := fmt.Sprintf("%s.%s", chartutil.DefaultKubeVersion.Major, chartutil.DefaultKubeVersion.Minor)

	// verify chart path exists
	if _, err := os.Stat(chartPath); err == nil {
		if chartPath, err = filepath.Abs(chartPath); err != nil {
			return "", err
		}
	} else {
		return "", err
	}

	// get combined values and create config
	config := &chart.Config{Raw: string(unmarshalledValues), Values: map[string]*chart.Value{}}

	if msgs := validation.IsDNS1123Subdomain(releaseName); releaseName != "" && len(msgs) > 0 {
		return "", fmt.Errorf("release name %s is invalid: %s", releaseName, strings.Join(msgs, ";"))
	}

	// Check chart requirements to make sure all dependencies are present in /charts
	c, err := chartutil.Load(chartPath)
	if err != nil {
		return "", err
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      releaseName,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: releaseNamespace,
		},
		KubeVersion: kubeVersion,
		APIVersions: []string{},
	}

	renderedTemplates, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return "", err
	}

	listManifests := manifest.SplitManifests(renderedTemplates)
	manifestsToRender := listManifests

	out := ""
	for _, m := range tiller.SortByKind(manifestsToRender) {
		data := m.Content
		b := filepath.Base(m.Name)
		if b == "NOTES.txt" {
			continue
		}
		if strings.HasPrefix(b, "_") {
			continue
		}

		out += fmt.Sprintf("---\n# Source: %s\n", m.Name)
		out += fmt.Sprintln(data)
	}

	return out, nil
}
