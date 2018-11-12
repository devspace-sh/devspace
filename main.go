package main

import (
	"fmt"

	"github.com/covexo/devspace/pkg/util/stdinutil"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var version string

func main() {
	/*upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)*/
	useDevSpaceCloud := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:               "Do you want to use the DevSpace Cloud? (free ready-to-use Kubernetes) (yes | no)",
		DefaultValue:           "yes",
		ValidationRegexPattern: "^(yes)|(no)$",
	})

	fmt.Println(useDevSpaceCloud)
}
