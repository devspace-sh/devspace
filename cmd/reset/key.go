package reset

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/spf13/cobra"
)

type keyCmd struct {
	Provider string
}

func newKeyCmd(f factory.Factory) *cobra.Command {
	cmd := &keyCmd{}

	keyCmd := &cobra.Command{
		Use:   "key",
		Short: "Resets a cluster key",
		Long: `
#######################################################
############### devspace reset key ####################
#######################################################
Resets a key for a given cluster. Useful if the key 
cannot be obtained anymore. Needs cluster access scope

Examples:
devspace reset key my-cluster
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunResetkey(f, cobraCmd, args)
		}}

	keyCmd.Flags().StringVar(&cmd.Provider, "provider", "", "The cloud provider to use")

	return keyCmd
}

// RunResetkey executes the reset key command logic
func (cmd *keyCmd) RunResetkey(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Get provider
	log := f.GetLog()
	provider, err := f.GetProvider(cmd.Provider, log)
	if err != nil {
		return err
	}

	// Reset the key
	err = provider.ResetKey(args[0])
	if err != nil {
		return err
	}

	log.Donef("Successfully reseted key for cluster %s", args[0])
	return nil
}
