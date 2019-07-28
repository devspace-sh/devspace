package analytics

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"path/filepath"
	"strings"
	"time"
	"regexp"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"github.com/google/uuid"
	homedir "github.com/mitchellh/go-homedir"
)

var token string
var analyticsConfigFile = constants.DefaultHomeDevSpaceFolder + "/analytics.yaml"
var analyticsInstance *analyticsConfig

type Analytics interface {
	Disable() error
	Enable() error
	SendEvent(eventName string, eventData map[string]interface{}) error
	SendCommandEvent(errorMessage string) error
	Identify(identifier string) error
}

type analyticsConfig struct {
	DistinctID string `yaml:"distinctId,omitempty"`
	Identifier string `yaml:"identifier,omitempty"`
	Disabled   bool   `yaml:"disabled,omitempty"`

	devSpaceVersion string
}

func (a *analyticsConfig) Disable() error {
	if !a.Disabled {
		a.Disabled = true
		return a.save()
	}
	return nil
}

func (a *analyticsConfig) Enable() error {
	if a.Disabled {
		a.Disabled = false
		return a.save()
	}
	return nil
}

func (a *analyticsConfig) Identify(identifier string) error {
	if identifier != a.Identifier {
		if a.Identifier != "" {
			// different user is logged in now => RESET DISTINCT ID
			a.resetDistinctID()
		}
		a.Identifier = identifier

		// Save a.Identifier as alias for a.DistinctID
		err := a.createAlias()
		if err != nil {
			return err
		}
		return a.save()
	}
	return nil
}

func (a *analyticsConfig) SendCommandEvent(errorMessage string) error {
	executable, _ := os.Executable()
	command := strings.Join(os.Args, " ")
	command = strings.Replace(command, executable, "devspace", 1)

	expr := regexp.MustCompile(`^(devspace\s+login\s.*--key=?\s*)(.*)(\s.*|$)`)
	command = expr.ReplaceAllString(command, `$1[REDACTED]$3`)

	commandData := map[string]interface{}{
		"command": command,
		"error":   errorMessage,
		"runtime_os": runtime.GOOS,
		"runtime_arch": runtime.GOARCH,
		"devspace_version": a.devSpaceVersion,
	}
	return a.SendEvent("command", commandData)
}

func (a *analyticsConfig) SendEvent(eventName string, eventData map[string]interface{}) error {
	if !a.Disabled {
		insertID, err := randutil.GenerateRandomString(16)
		if err != nil {
			return fmt.Errorf("Couldn't generate random insert_id for analytics: %v", err)
		}
		eventData["$insert_id"] = insertID
		eventData["token"] = token

		if a.Identifier != "" {
			eventData["distinct_id"] = a.Identifier
		} else {
			eventData["distinct_id"] = a.DistinctID
		}
		
		if _, ok := eventData["time"]; !ok {
			eventData["time"] = time.Now()
		}
		data := map[string]interface{}{}

		data["event"] = eventName
		data["properties"] = eventData
		return a.sendRequest("track", data)
	}
	return nil
}

func (a *analyticsConfig) UpdateUser(userData map[string]interface{}) error {
	if !a.Disabled {
		data := map[string]interface{}{}
		data["$token"] = token
		data["$distinct_id"] = a.Identifier

		if _, ok := data["$time"]; !ok {
			data["$time"] = time.Now()
		}
		data["$set"] = userData

		return a.sendRequest("engage", data)
	}
	return nil
}

func (a *analyticsConfig) createAlias() error {
	if !a.Disabled {
		data := map[string]interface{}{
			"event": "$create_alias",
			"properties": map[string]interface{}{
				"distinct_id": a.Identifier,
				"alias": a.DistinctID,
				"token": token,
			},
		}
		identifierSections := strings.Split(a.Identifier, "/")

		a.UpdateUser(map[string]interface{}{
			"provider": identifierSections[0],
			"account_id": identifierSections[1],
		})
		return a.sendRequest("track", data)
	}
	return nil
}

func (a *analyticsConfig) sendRequest(endpointPath string, data map[string]interface{}) error {
	if !a.Disabled {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("Couldn't marshal analytics data to json: %v", err)
		}

		requestURL := "https://api.mixpanel.com/" + endpointPath + "/?data=" + base64.StdEncoding.EncodeToString(jsonData)

		response, err := http.Get(requestURL)
		if err != nil {
			return fmt.Errorf("Couldn't make request to analytics endpoint: %v", err)
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("Couldn't get http response from analytics request: %v", err)
		}

		if string(body) == "1" {
			return nil
		}
		return fmt.Errorf("Received error from analytics request: %v", err)
	}
	return nil
}

func (a *analyticsConfig) resetDistinctID() error {
	DistinctID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("Couldn't create UUID: %v", err)
	}
	a.DistinctID = DistinctID.String()

	return nil
}

func (a *analyticsConfig) save() error {
	analyticsConfigFilePath, err := a.getAnalyticsConfigFilePath()
	if err != nil {
		return fmt.Errorf("Couldn't determine config file %s: %v", err)
	}

	err = yamlutil.WriteYamlToFile(a, analyticsConfigFilePath)
	if err != nil {
		return fmt.Errorf("Couldn't save analytics config file %s: %v", analyticsConfigFilePath, err)
	}
	return nil
}

func (a *analyticsConfig) getAnalyticsConfigFilePath() (string, error) {
	homedir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homedir, analyticsConfigFile), nil
}

func GetAnalytics() Analytics {
	return analyticsInstance
}

func SetVersion(version string) error {
	analyticsInstance = &analyticsConfig{}

	analyticsConfigFilePath, err := analyticsInstance.getAnalyticsConfigFilePath()
	if err != nil {
		return fmt.Errorf("Couldn't determine config file %s: %v", err)
	}
	_, err = os.Stat(analyticsConfigFilePath)
	if err == nil {
		err := yamlutil.ReadYamlFromFile(analyticsConfigFilePath, analyticsInstance)
		if err != nil {
			return fmt.Errorf("Couldn't read analytics config file %s: %v", analyticsConfigFilePath, err)
		}
	} else {
		err = analyticsInstance.resetDistinctID()
		if err != nil {
			//TODO
		}

		err = analyticsInstance.save()
		if err != nil {
			//TODO
		}
	}
	analyticsInstance.devSpaceVersion = version

	return nil
}
