package helm

import (
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/covexo/devspace/pkg/util/log"
	helmdownloader "k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/repo"
)

// stringArraySorter
type stringArraySorter [][]string

// Len returns the length of this scoreSorter.
func (s stringArraySorter) Len() int { return len(s) }

// Swap performs an in-place swap.
func (s stringArraySorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Less compares a to b, and returns true if a is less than b.
func (s stringArraySorter) Less(a, b int) bool { return s[a][0] < s[b][0] }

// PrintAllAvailableCharts prints all available charts
func (helmClientWrapper *ClientWrapper) PrintAllAvailableCharts() {
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
func (helmClientWrapper *ClientWrapper) SearchChart(chartName, chartVersion, appVersion string) (*repo.Entry, *repo.ChartVersion, error) {
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
func (helmClientWrapper *ClientWrapper) BuildDependencies(chartPath string) error {
	man := &helmdownloader.Manager{
		Out:       ioutil.Discard,
		ChartPath: chartPath,
		HelmHome:  helmClientWrapper.Settings.Home,
		Getters:   getter.All(*helmClientWrapper.Settings),
	}

	return man.Build()
}

// UpdateDependencies updates the dependencies
func (helmClientWrapper *ClientWrapper) UpdateDependencies(chartPath string) error {
	man := &helmdownloader.Manager{
		Out:       ioutil.Discard,
		ChartPath: chartPath,
		HelmHome:  helmClientWrapper.Settings.Home,
		Getters:   getter.All(*helmClientWrapper.Settings),
	}

	return man.Update()
}
