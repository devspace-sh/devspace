package set

import (
	"github.com/devspace-cloud/devspace/pkg/util/factory"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type encryptionkeyCmd struct {
	Cluster string
}

func newEncryptionKeyCmd(f factory.Factory) *cobra.Command {
	cmd := &encryptionkeyCmd{}

	encryptionkeyCmd := &cobra.Command{
		Use:   "encryptionkey",
		Short: "Sets the encryption",
		Long: `
#######################################################
############## devspace set encryptionkey #############
#######################################################
Sets an encryption key for a given cluster

Examples:
devspace set encryptionkey mykey --cluster mycluster 
devspace set encryptionkey --cluster mycluster --reset
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunSetEncryptionKey(f, cobraCmd, args)
		}}

	encryptionkeyCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to apply this key for")

	return encryptionkeyCmd
}

// RunSetEncryptionKey executes the set encryptionkey command logic
func (cmd *encryptionkeyCmd) RunSetEncryptionKey(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	if cmd.Cluster == "" {
		return errors.Errorf("--cluster has to be specified. You can view the available clusters with `devspace list clusters`")
	}

	// Get provider configuration
	provider, err := f.GetProvider("", f.GetLog())
	if err != nil {
		return err
	}

	cluster, err := provider.Client().GetClusterByName(cmd.Cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster")
	}

	hashedKey, err := hash.Password(args[0])
	if err != nil {
		return errors.Wrap(err, "hash key")
	}

	valid, err := provider.Client().VerifyKey(cluster.ClusterID, hashedKey)
	if err != nil {
		return errors.Wrap(err, "is key valid")
	} else if !valid {
		return errors.Errorf("Provided key is not valid for cluster %s", args[0])
	}

	provider.GetConfig().ClusterKey[cluster.ClusterID] = hashedKey
	err = provider.Save()
	if err != nil {
		return errors.Wrap(err, "save provider")
	}

	f.GetLog().Infof("Successfully set encryption key for cluster %s", args[0])
	return nil
}
