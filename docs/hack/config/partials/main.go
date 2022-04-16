package main

import (
	"github.com/loft-sh/devspace/docs/hack/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

const configPartialBasePath = "docs/pages/configuration/_partials/"

func main() {
	repository := "github.com/loft-sh/devspace"
	configGoPackage := "./pkg/devspace/config/versions/latest"
	versionedConfigBasePath := configPartialBasePath + latest.Version + "/"

	schema := util.GenerateSchema(latest.Config{}, repository, configGoPackage)
	util.GenerateReference(schema, versionedConfigBasePath)

	schema = util.GenerateSchema(latest.ComponentConfig{}, repository, configGoPackage)
	util.GenerateReference(schema, versionedConfigBasePath+"deployments/helm/componentChart/")
}
