package tanka

import (
	"fmt"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

// validateConfigs checks the tanka config for sanity
func validateConfig(cfg *latest.DeploymentConfig) error {
	var errors []string

	if cfg.Tanka == nil {
		errors = append(errors, "tanka is nil")
	} else if cfg.Tanka.Path == "" && cfg.Tanka.EnvironmentPath == "" {
		errors = append(errors, "neither tanka.path nor tanka.environmentPath is configured")
	} else if cfg.Tanka.EnvironmentName == "" && cfg.Tanka.EnvironmentPath == "" {
		errors = append(errors, "neither tanka.environmentName nor tanka.environmentPath is configured")
	}

	if len(errors) != 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}
