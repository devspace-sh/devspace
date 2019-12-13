package set

import (
	"github.com/devspace-cloud/devspace/pkg/util/analytics"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type analyticsCmd struct{}

func newAnalyticsCmd() *cobra.Command {
	cmd := &analyticsCmd{}

	return &cobra.Command{
		Use:   "analytics",
		Short: "Update analytics settings",
		Long: `
#######################################################
############### devspace set analytics ################
#######################################################
Example:
devspace set analytics disabled true
#######################################################
	`,
		Args: cobra.RangeArgs(1, 2),
		RunE: cmd.RunAnalyticsConfig,
	}
}

// RunAnalyticsConfig executes the "devspace set analytics" logic
func (*analyticsCmd) RunAnalyticsConfig(cobraCmd *cobra.Command, args []string) error {
	analytics, err := analytics.GetAnalytics()
	if err != nil {
		return errors.Wrap(err, "get analytics config")
	}

	if args[0] == "disabled" {
		if len(args) == 2 && (args[1] == "false" || args[1] == "0") {
			err = analytics.Enable()
		} else {
			err = analytics.Disable()
		}
	}

	if err != nil {
		return errors.Wrap(err, "set analytics config")
	}

	log.GetInstance().Infof("Successfully updated analytics config")
	return nil
}
