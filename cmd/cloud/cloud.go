package cloud

import "github.com/spf13/cobra"

// Cmd is the cloud command that is exported
var Cmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud specific commands",
	Long: `
#######################################################
################## devspace cloud #####################
#######################################################
`,
	Args: cobra.NoArgs,
}
