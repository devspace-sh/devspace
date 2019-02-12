package status

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

var syncStopped = regexp.MustCompile(`^\[Sync\] Sync stopped$`)
var downstreamChanges = regexp.MustCompile(`^\[Downstream\] Successfully processed (\d+) change\(s\)$`)
var upstreamChanges = regexp.MustCompile(`^\[Upstream\] Successfully processed (\d+) change\(s\)$`)

type syncStatus struct {
	Status    string
	Pod       string
	Local     string
	Container string

	LastActivity     string
	LastActivityTime string
	Error            string

	TotalChanges int
}

// RunStatusSync executes the devspace status sync commad logic
func (cmd *StatusCmd) RunStatusSync(cobraCmd *cobra.Command, args []string) {
	// Read syncLog
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	syncLogPath := filepath.Join(cwd, ".devspace", "logs", "sync.log")
	data, err := ioutil.ReadFile(syncLogPath)
	if err != nil {
		log.Fatalf("Couldn't read %s. Do you have a sync path configured? (check `devspace list sync`)", syncLogPath)
	}

	syncMap := make(map[string]*syncStatus)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		jsonMap := make(map[string]string)
		err = json.Unmarshal([]byte(line), &jsonMap)
		if err != nil {
			log.Fatal(err)
		}
		if isSyncJSONMapInvalid(jsonMap) {
			log.Fatalf("Error parsing %s: Json object is invalid %s", syncLogPath, line)
		}

		err = updateSyncMap(syncMap, jsonMap)
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(syncMap) == 0 {
		log.Info("No sync activity found. Did you run `devspace up`?")
		return
	}

	// Print table
	header := []string{
		"Status",
		"Pod",
		"Local",
		"Container",
		"Latest Activity",
		"Total Changes",
	}

	values := make([][]string, 0, len(syncMap))

	for _, status := range syncMap {
		latestActivity := status.LastActivity

		if status.Error != "" {
			latestActivity = status.Error
		}

		parsedTime, _ := time.Parse(time.RFC3339, status.LastActivityTime)
		if parsedTime.Unix() == 0 {
			parsedTime = time.Now()
		}

		latestActivity += " (" + intToTimeString(int(time.Now().Unix()-parsedTime.Unix())) + " ago)"

		syncStatus := status.Status
		if syncStatus == "" {
			syncStatus = "Active"
		}

		if len(status.Pod) > 15 {
			status.Pod = status.Pod[:15] + "..."
		}
		if len(status.Local) > 20 {
			status.Local = "..." + status.Local[len(status.Local)-20:len(status.Local)]
		}
		if len(status.Container) > 20 {
			status.Container = "..." + status.Container[len(status.Container)-20:len(status.Container)]
		}

		values = append(values, []string{
			syncStatus,
			status.Pod,
			status.Local,
			status.Container,
			latestActivity,
			strconv.Itoa(status.TotalChanges),
		})
	}

	log.PrintTable(header, values)
}

func intToTimeString(timeDifference int) string {
	days := math.Floor(float64(timeDifference) / (60.0 * 60.0 * 24.0))
	if days > 0 {
		if days == 1 {
			return "1d"
		}

		return strconv.Itoa(int(days)) + "d"
	}

	hours := math.Floor(float64(timeDifference) / (60.0 * 60.0))
	if hours > 0 {
		if hours == 1 {
			return "1h"
		}

		return strconv.Itoa(int(hours)) + "h"
	}

	minutes := math.Floor(float64(timeDifference) / 60.0)
	if minutes > 0 {
		if minutes == 1 {
			return "1m"
		}

		return strconv.Itoa(int(minutes)) + "m"
	}

	if timeDifference > 0 {
		if timeDifference == 1 {
			return "1s"
		}

		return strconv.Itoa(timeDifference) + "s"
	}

	return "0s"
}

func isSyncJSONMapInvalid(jsonMap map[string]string) bool {
	return jsonMap["container"] == "" || jsonMap["local"] == "" || jsonMap["pod"] == "" || jsonMap["level"] == "" || jsonMap["time"] == "" || jsonMap["msg"] == ""
}

func updateSyncMap(syncMap map[string]*syncStatus, jsonMap map[string]string) error {
	pod := jsonMap["pod"]
	local := jsonMap["local"]
	container := jsonMap["container"]
	message := jsonMap["msg"]
	level := jsonMap["level"]
	time := jsonMap["time"]

	identifier := pod + ":" + local + ":" + container

	if syncMap[identifier] == nil {
		syncMap[identifier] = &syncStatus{
			Pod:       pod,
			Container: container,
			Local:     local,
		}
	}

	if level == "error" {
		syncMap[identifier].Status = "Error"
		syncMap[identifier].Error = message
		syncMap[identifier].LastActivityTime = time
	} else if matches := downstreamChanges.FindStringSubmatch(message); len(matches) == 2 {
		syncMap[identifier].LastActivity = "Downloaded " + matches[1] + " changes"
		syncMap[identifier].LastActivityTime = time

		changes, _ := strconv.Atoi(matches[1])
		syncMap[identifier].TotalChanges += changes
	} else if matches := upstreamChanges.FindStringSubmatch(message); len(matches) == 2 {
		syncMap[identifier].LastActivity = "Uploaded " + matches[1] + " changes"
		syncMap[identifier].LastActivityTime = time

		changes, _ := strconv.Atoi(matches[1])
		syncMap[identifier].TotalChanges += changes
	} else if syncStopped.MatchString(message) {
		syncMap[identifier].Status = "Stopped"
		syncMap[identifier].LastActivity = "Sync stopped"
		syncMap[identifier].LastActivityTime = time
	}

	return nil
}
