package main

import (
	"math/rand"
	"os"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
)

var version string = ""

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
