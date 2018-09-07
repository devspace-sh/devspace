package cmd

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// RunStatusSync executes the devspace status sync commad logic
func (cmd *StatusCmd) RunStatusSync(cobraCmd *cobra.Command, args []string) {
	// config := configutil.GetConfig(false)

	// Read syncLog
	/*cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	syncLogPath := path.Join(cwd, ".devspace", "logs", "syncLog.log")
	data, err := ioutil.ReadFile(syncLogPath)
	if err != nil {
		log.Fatalf("Couldn't read %s. Do you have a sync path configured?", syncLogPath)
	}

	_ = strings.Split(string(data), "\n")*/
}
