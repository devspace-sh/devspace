package types

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

func GetDefaultPipeline(pipeline string) (*latest.Pipeline, error) {
	switch pipeline {
	case "deploy":
		return DefaultDeployPipeline, nil
	case "dev":
		return DefaultDevPipeline, nil
	case "purge":
		return DefaultPurgePipeline, nil
	case "build":
		return DefaultBuildPipeline, nil
	}

	return nil, fmt.Errorf("couldn't find pipeline %v", pipeline)
}

var DefaultDeployPipeline = &latest.Pipeline{
	Name: "deploy",
	Run: `run_dependencies --all
ensure_pull_secrets --all
build_images --all
create_deployments --all`,
}

var DefaultDevPipeline = &latest.Pipeline{
	Name: "dev",
	Run: `run_dependencies --all
ensure_pull_secrets --all
build_images --all
create_deployments --all
start_dev --all`,
}

var DefaultPurgePipeline = &latest.Pipeline{
	Name: "purge",
	Run: `stop_dev --all
purge_deployments --all
run_dependencies --all --pipeline purge`,
}

var DefaultBuildPipeline = &latest.Pipeline{
	Name: "build",
	Run: `run_dependencies --all --pipeline build
build_images --all`,
}
