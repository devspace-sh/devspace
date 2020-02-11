package cloudanalytics

import (
	"net/url"
	"os"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/analytics"
)

// ReportPanics resolves a panic
func ReportPanics() {
	defer func() {
		if r := recover(); r != nil {
			// Fail silently
		}
	}()

	a, err := analytics.GetAnalytics()
	if err != nil {
		return
	}

	a.ReportPanics()
}

// SendCommandEvent sends a new event to the analytics provider
func SendCommandEvent(commandErr error) {
	defer func() {
		if r := recover(); r != nil {
			// Fail silently
		}
	}()

	a, err := analytics.GetAnalytics()
	if err != nil {
		return
	}

	a.SendCommandEvent(os.Args, commandErr, analytics.GetProcessDuration())
}

// Start initializes the analytics
func Start(version string) {
	defer func() {
		if r := recover(); r != nil {
			// Fail silently
		}
	}()

	analytics.SetConfigPath(constants.DefaultHomeDevSpaceFolder + "/analytics.yaml")

	analytics, err := analytics.GetAnalytics()
	if err != nil {
		return
	}

	analytics.SetVersion(version)
	analytics.SetIdentifyProvider(getIdentity)
	err = analytics.HandleDeferredRequest()
	if err != nil {
		// ignore error
	}
}

// getIdentity return the cloud identifier
func getIdentity() string {
	loader := config.NewLoader()
	providerConfig, err := loader.Load()
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
