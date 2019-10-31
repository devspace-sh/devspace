package cloudanalytics

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
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

// SendCommandEventBackground sends a new event to the analytics provider in the background
func SendCommandEventBackground(commandErr error) {
	commandDuration := analytics.GetProcessDuration()
	commandErrMessage := ""
	if commandErr != nil {
		commandErrMessage = commandErr.Error()
	}

	args := []string{"send", "event", strconv.FormatInt(commandDuration, 10), commandErrMessage}
	args = append(args, os.Args...)

	cmd := exec.Command("devspace", args...)
	cmd.Start()
}

// HandleSendCommand sends an event
func HandleSendCommand() {
	if len(os.Args) < 6 || os.Args[1] != "send" || os.Args[2] != "event" {
		return
	}

	a, err := analytics.GetAnalytics()
	if err != nil {
		panic(err)
	}

	commandDuration, err := strconv.ParseInt(os.Args[3], 10, 64)
	if err != nil {
		panic(err)
	}

	var commandErr error
	if os.Args[4] != "" {
		commandErr = errors.New(os.Args[4])
	}

	commandArgs := os.Args[5:]
	err = a.SendCommandEvent(commandArgs, commandErr, commandDuration)
	if err != nil {
		panic(err)
	}

	os.Exit(0)
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
}

// getIdentity return the cloud identifier
func getIdentity() string {
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
