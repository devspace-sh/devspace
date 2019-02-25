package main

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"os"

	"github.com/covexo/devspace/cmd"
	"github.com/covexo/devspace/pkg/devspace/upgrade"
)

var version string

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
