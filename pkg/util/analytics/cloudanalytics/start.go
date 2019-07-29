package cloudanalytics

import (
	"github.com/devspace-cloud/devspace/pkg/util/analytics"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"net/url"
	"strconv"
)

func Start(version string) {
	analytics.SetConfigPath(constants.DefaultHomeDevSpaceFolder + "/analytics.yaml")

	analytics, err := analytics.GetAnalytics()
	if err == nil {
		analytics.SetVersion(version)

		providerConfig, err := config.ParseProviderConfig()
		if err == nil {
			var provider *latest.Provider

			providerName := config.DevSpaceCloudProviderName
			
			// Choose cloud provider
			if providerConfig.Default != "" {
				providerName = providerConfig.Default
			}
			
			for _, providerEntry := range providerConfig.Providers {
				if providerEntry.Name == providerName {
					provider = providerEntry
				}
			}

			if provider != nil && provider.Host != "" && provider.Token != "" {
				parsedURL, _ := url.Parse(provider.Host)

				identifier, err := token.GetAccountID(provider.Token)

				if err != nil {
					stringIdentifier := strconv.Itoa(identifier)
					
					// Ignore if identify fails
					_ = analytics.Identify(parsedURL.Host + "/" + stringIdentifier)
				}
			}
		}
	}
}