package main

import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
)

var version string

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
