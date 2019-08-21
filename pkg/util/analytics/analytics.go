package analytics

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

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
	DistinctID   string `yaml:"distinctId,omitempty"`
	Identifier   string `yaml:"identifier,omitempty"`
	Disabled     bool   `yaml:"disabled,omitempty"`
	LatestUpdate int64  `yaml:"latestUpdate,omitempty"`

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

		// Save a.Identifier as alias for a.DistinctID
		err := a.createAlias(identifier)
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

	expr := regexp.MustCompile(`^.*\s+(login\s.*--key=?\s*)(.*)(\s.*|$)`)
	command = expr.ReplaceAllString(command, `devspace $1[REDACTED]$3`)

	commandData := map[string]interface{}{
		"command":      command,
		"$os":          runtime.GOOS,
		"runtime_arch": runtime.GOARCH,
		"cli_version":  a.version,
	}

	if commandError != nil {
		commandData["error"] = commandError.Error()
	}

	if a.Identifier != "" {
		commandData["$user_id"] = a.Identifier
	}

	pid := os.Getpid()
	p, err := process.NewProcess(int32(pid))
	if err == nil {
		procCreateTime, err := p.CreateTime()
		if err == nil {
			commandData["command_duration"] = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond)-procCreateTime, 10) + "ms"
		}
	}

	if regexp.MustCompile(`^.*\s+(use\s+space\s.*--get-token((\s*)|$))`).MatchString(command) {
		return a.SendEvent("context", commandData)
	}
	return a.SendEvent("command", commandData)
}

func (a *analyticsConfig) SendEvent(eventName string, eventData map[string]interface{}) error {
	if !a.Disabled && token != "" {
		now := time.Now()

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
			eventData["time"] = now
		}
		data := map[string]interface{}{}

		data["event"] = eventName
		data["properties"] = eventData

		if time.Unix(a.LatestUpdate, 0).Add(time.Minute * 5).Before(now) {
			// ignore if user update fails
			_ = a.UpdateUser(map[string]interface{}{})
		}
		return a.sendRequest("track", data)
	}
	return nil
}

func (a *analyticsConfig) UpdateUser(userData map[string]interface{}) error {
	if !a.Disabled && token != "" {
		a.LatestUpdate = time.Now().Unix()

		// ignore if config save fails
		_ = a.save()

		data := map[string]interface{}{}
		data["$token"] = token

		if a.Identifier != "" {
			data["$distinct_id"] = a.Identifier
		} else {
			data["$distinct_id"] = a.DistinctID
		}

		if _, ok := userData["cli_version"]; !ok {
			userData["cli_version"] = a.version
		}

		if _, ok := userData["runtime_os"]; !ok {
			userData["runtime_os"] = runtime.GOOS
		}

		if _, ok := userData["runtime_arch"]; !ok {
			userData["runtime_arch"] = runtime.GOARCH
		}

		if _, ok := userData["duplicate"]; !ok {
			userData["duplicate"] = false
		}
		data["$set"] = userData

		return a.sendRequest("engage", data)
	}
	return nil
}

func (a *analyticsConfig) ReportPanics() {
	if r := recover(); r != nil {
		err := fmt.Errorf("Panic: %v\n%v", r, string(debug.Stack()))

		a.SendCommandEvent(err)
	}
}

func (a *analyticsConfig) SetVersion(version string) {
	a.version = version
}

func (a *analyticsConfig) createAlias(identifier string) error {
	if !a.Disabled {
		identifierSections := strings.Split(identifier, "/")

		// mark current distinct_id as duplicate
		// => future calls will reset this for the correct distinct_id
		// ignore if user update fails
		_ = a.UpdateUser(map[string]interface{}{
			"provider":   identifierSections[0],
			"account_id": identifierSections[1],
			"duplicate":  true,
		})

		a.Identifier = identifier

		// ignore if update user fails
		_ = a.UpdateUser(map[string]interface{}{
			"provider":   identifierSections[0],
			"account_id": identifierSections[1],
		})

		data := map[string]interface{}{
			"event": "$create_alias",
			"properties": map[string]interface{}{
				"distinct_id": a.DistinctID,
				"alias":       a.Identifier,
				"token":       token,
			},
		}
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

		}

		if analyticsInstance.DistinctID == "" {
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

		go func() {
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
