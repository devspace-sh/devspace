package main

/*
import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/pkg/devspace/upgrade"
)

var version string

func main() {
	upgrade.SetVersion(version)

	cmd.Execute()
	os.Exit(0)
}
*/

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func main() {
	log.Info("Start bois")

	client, err := docker.NewClient(nil, false, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}

	authConfig, err := docker.GetAuthConfig(client, "", true)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("%#v", *authConfig)
}
