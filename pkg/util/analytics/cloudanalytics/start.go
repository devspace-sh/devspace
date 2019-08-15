package cloudanalytics

import (
	"net/url"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/analytics"
)

// ReportPanics resolves a panic
func ReportPanics() {
	analytics, err := analytics.GetAnalytics()
	if err == nil {
		analytics.ReportPanics()
	}
}

// SendCommandEvent sends a new event to the analytics provider
func SendCommandEvent(commandErr error) {
	analytics, err := analytics.GetAnalytics()
	if err == nil {
		// Ignore analytics error
		_ = analytics.SendCommandEvent(commandErr)
	}
}

// Start initializes the analytics
func Start(version string) {
	analytics.SetConfigPath(constants.DefaultHomeDevSpaceFolder + "/analytics.yaml")

	analytics, err := analytics.GetAnalytics()
	if err == nil {
		analytics.SetVersion(version)

		providerConfig, err := config.ParseProviderConfig()
		if err == nil {
			providerName := config.DevSpaceCloudProviderName

			// Choose cloud provider
			if providerConfig.Default != "" {
				providerName = providerConfig.Default
			}

			provider := config.GetProvider(providerConfig, providerName)
			if provider != nil && provider.Host != "" && provider.Token != "" {
				parsedURL, err := url.Parse(provider.Host)
				if err == nil {
					identifier, err := token.GetAccountID(provider.Token)
					if err == nil {
						stringIdentifier := strconv.Itoa(identifier)

						// Ignore if identify fails
						_ = analytics.Identify(parsedURL.Host + "/" + stringIdentifier)
					}
				}
			}
		}
	}
}
