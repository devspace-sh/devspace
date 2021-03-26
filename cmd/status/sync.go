package status

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var syncStopped = regexp.MustCompile(`^\[Sync\] Sync stopped$`)
var downstreamChanges = regexp.MustCompile(`^\[Downstream\] Successfully processed (\d+) change\(s\)$`)
var upstreamChanges = regexp.MustCompile(`^\[Upstream\] Successfully processed (\d+) change\(s\)$`)

type syncStatus struct {
	Level   string
	Message string
}

type syncCmd struct{}

func newSyncCmd(f factory.Factory) *cobra.Command {
	cmd := &syncCmd{}

	return &cobra.Command{
		Use:   "sync",
		Short: "Shows the sync status",
		Long: `
#######################################################
################ devspace status sync #################
#######################################################
Shows the sync status
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunStatusSync(f, cobraCmd, args)
		}}
}

// RunStatusSync executes the devspace status sync commad logic
func (cmd *syncCmd) RunStatusSync(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	logger := f.GetLog()
	configLoader := f.NewConfigLoader("")
	configExists, err := configLoader.SetDevSpaceRoot(logger)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	logger.Warn("This command is deprecated and will be removed in a future DevSpace version. Please take a look at the sync logs at .devspace/logs/sync.log instead")

	// Read syncLog
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	syncLogPath := filepath.Join(cwd, ".devspace", "logs", "sync.log")
	data, err := ioutil.ReadFile(syncLogPath)
	if err != nil {
		return errors.Errorf("Couldn't read %s. Do you have a sync path configured? (check `devspace list sync`)", syncLogPath)
	}

	// Prepare table
	header := []string{
		"Level",
		"Message",
		"Time",
	}

	values := [][]string{}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		jsonMap := make(map[string]string)
		err = json.Unmarshal([]byte(line), &jsonMap)
		if err != nil {
			return err
		}
		if isSyncJSONMapInvalid(jsonMap) {
			return errors.Errorf("Error parsing %s: Json object is invalid %s", syncLogPath, line)
		}

		values = append(values, []string{jsonMap["level"], jsonMap["msg"], jsonMap["time"]})
	}

	if len(values) == 0 {
		logger.Info("No sync activity found. Did you run `devspace dev`?")
		return nil
	}

	log.PrintTable(logger, header, values)
	return nil
}

func isSyncJSONMapInvalid(jsonMap map[string]string) bool {
	return jsonMap["level"] == "" || jsonMap["time"] == "" || jsonMap["msg"] == ""
}
