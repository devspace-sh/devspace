package set

import (
	"github.com/devspace-cloud/devspace/pkg/util/analytics"
	"github.com/devspace-cloud/devspace/pkg/util/factory"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type analyticsCmd struct{}

func newAnalyticsCmd(f factory.Factory) *cobra.Command {
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunAnalyticsConfig(f, cobraCmd, args)
		},
	}
}

// RunAnalyticsConfig executes the "devspace set analytics" logic
func (*analyticsCmd) RunAnalyticsConfig(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	log := f.GetLog()
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

	log.Infof("Successfully updated analytics config")
	return nil
}
