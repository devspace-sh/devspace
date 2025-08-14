package main

import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
)

var version = ""

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
