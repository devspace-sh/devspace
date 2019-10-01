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
	if err != nil {
		return
	}

	analytics.ReportPanics()
}

// SendCommandEvent sends a new event to the analytics provider
func SendCommandEvent(commandErr error) {
	analytics, err := analytics.GetAnalytics()
	if err != nil {
		return
	}

	// Ignore analytics error
	_ = analytics.SendCommandEvent(commandErr)
}

// Start initializes the analytics
func Start(version string) {
	analytics.SetConfigPath(constants.DefaultHomeDevSpaceFolder + "/analytics.yaml")

	analytics, err := analytics.GetAnalytics()
	if err != nil {
		return
	}

	analytics.SetVersion(version)
	analytics.SetIdentifyProvider(GetIdentity)
}

// GetIdentity return the cloud identifier
func GetIdentity() string {
	providerConfig, err := config.ParseProviderConfig()
	if err != nil {
		return ""
	}

	providerName := config.DevSpaceCloudProviderName

	// Choose cloud provider
	if providerConfig.Default != "" {
		providerName = providerConfig.Default
	}

	provider := config.GetProvider(providerConfig, providerName)
	if provider == nil || provider.Host == "" || provider.Token == "" {
		return ""
	}

	parsedURL, err := url.Parse(provider.Host)
	if err != nil {
		return ""
	}

	identifier, err := token.GetAccountID(provider.Token)
	if err != nil {
		return ""
	}

	stringIdentifier := strconv.Itoa(identifier)

	// Ignore if identify fails
	return parsedURL.Host + "/" + stringIdentifier
}
