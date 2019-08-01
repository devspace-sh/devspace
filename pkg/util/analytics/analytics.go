package analytics

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"path/filepath"
	"strings"
	"time"
	"regexp"
	"errors"
	"sync"
	"strconv"

	"github.com/devspace-cloud/devspace/pkg/util/randutil"
	"github.com/devspace-cloud/devspace/pkg/util/yamlutil"
	"github.com/google/uuid"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/shirou/gopsutil/process"
)

var token string
var analyticsConfigFile string
var analyticsInstance *analyticsConfig
var loadAnalyticsOnce sync.Once

// Analytics is an interface for sending data to an analytics service
type Analytics interface {
	Disable() error
	Enable() error
	SendEvent(eventName string, eventData map[string]interface{}) error
	SendCommandEvent(commandError error) error
	ReportPanics()
	Identify(identifier string) error
	SetVersion(version string)
}

type analyticsConfig struct {
	DistinctID string `yaml:"distinctId,omitempty"`
	Identifier string `yaml:"identifier,omitempty"`
	Disabled   bool   `yaml:"disabled,omitempty"`

	version string
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
			_ = a.resetDistinctID()
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

func (a *analyticsConfig) SendCommandEvent(commandError error) error {
	executable, _ := os.Executable()
	command := strings.Join(os.Args, " ")
	command = strings.Replace(command, executable, "devspace", 1)

	expr := regexp.MustCompile(`^(devspace\s+login\s.*--key=?\s*)(.*)(\s.*|$)`)
	command = expr.ReplaceAllString(command, `$1[REDACTED]$3`)

	commandData := map[string]interface{}{
		"command": command,
		"runtime_os": runtime.GOOS,
		"runtime_arch": runtime.GOARCH,
		"cli_version": a.version,
	}
	
	if commandError != nil {
		commandData["error"] = commandError.Error()
	}
	
	pid := os.Getpid()
	p, err := process.NewProcess(int32(pid))
	if err == nil {
		procCreateTime, err := p.CreateTime()
		if err == nil {
			commandData["command_duration"] = strconv.FormatInt(time.Now().UnixNano() / int64(time.Millisecond) - procCreateTime, 10) + "ms"
		}
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

func (a *analyticsConfig) ReportPanics() {
	if r := recover(); r != nil {
		err := fmt.Errorf("Panic: %v", r)

		a.SendCommandEvent(err)
		fmt.Println(err)
	}
}

func (a *analyticsConfig) SetVersion(version string) {
	a.version = version
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
		return errors.New("Received error from analytics request")
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
		return fmt.Errorf("Couldn't determine config file: %v", err)
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

func GetAnalytics() (Analytics, error) {
	var err error

	loadAnalyticsOnce.Do(func() {
		analyticsInstance = &analyticsConfig{}
	
		analyticsConfigFilePath, err := analyticsInstance.getAnalyticsConfigFilePath()
		if err != nil {
			err = fmt.Errorf("Couldn't determine config file: %v", err)
			return
		}
		_, err = os.Stat(analyticsConfigFilePath)
		if err == nil {
			err := yamlutil.ReadYamlFromFile(analyticsConfigFilePath, analyticsInstance)
			if err != nil {
				err = fmt.Errorf("Couldn't read analytics config file %s: %v", analyticsConfigFilePath, err)
				return
			}
		} else {
			err = analyticsInstance.resetDistinctID()
			if err != nil {
				err = fmt.Errorf("Couldn't reset analytics distinct id: %v", err)
				return
			}
	
			err = analyticsInstance.save()
			if err != nil {
				err = fmt.Errorf("Couldn't save analytics config: %v", err)
				return
			}
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		go func(){
			<-c

			analyticsInstance.SendCommandEvent(errors.New("Interrupted"))
			signal.Stop(c)

			pid := os.Getpid()
			sigterm(pid)
		}()
	})
	return analyticsInstance, err
}

func SetConfigPath(path string) {
	analyticsConfigFile = path
}
