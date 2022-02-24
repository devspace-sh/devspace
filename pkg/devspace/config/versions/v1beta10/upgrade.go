package v1beta10

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/v1beta11"
	"github.com/loft-sh/devspace/pkg/util/log"
	"strings"
)

// Upgrade upgrades the config
func (c *Config) Upgrade(log log.Logger) (config.Config, error) {
	nextConfig := &next.Config{}
	err := util.Convert(c, nextConfig)
	if err != nil {
		return nil, err
	}

	// dependencies
	for i, d := range c.Dependencies {
		if d.OverwriteVars == nil || *d.OverwriteVars {
			nextConfig.Dependencies[i].OverwriteVars = true
		}
	}

	// commands
	for i, c := range c.Commands {
		if c.AppendArgs == nil || *c.AppendArgs {
			nextConfig.Commands[i].AppendArgs = true
		}
	}

	// replace pod
	for i, rp := range c.Dev.ReplacePods {
		if rp.ImageName != "" {
			nextConfig.Dev.ReplacePods[i].ImageSelector = fmt.Sprintf("image(%s):tag(%s)", rp.ImageName, rp.ImageName)
		}
	}

	// port forwarding
	for i, rp := range c.Dev.Ports {
		if rp.ImageName != "" {
			nextConfig.Dev.Ports[i].ImageSelector = fmt.Sprintf("image(%s):tag(%s)", rp.ImageName, rp.ImageName)
		}
	}

	// sync
	for i, rp := range c.Dev.Sync {
		if rp.ImageName != "" {
			nextConfig.Dev.Sync[i].ImageSelector = fmt.Sprintf("image(%s):tag(%s)", rp.ImageName, rp.ImageName)
		}
	}

	// logs
	if c.Dev.Logs != nil {
		for _, rp := range c.Dev.Logs.Images {
			if nextConfig.Dev.Logs == nil {
				nextConfig.Dev.Logs = &next.LogsConfig{}
			}
			if nextConfig.Dev.Logs.Selectors == nil {
				nextConfig.Dev.Logs.Selectors = []next.LogsSelector{}
			}

			nextConfig.Dev.Logs.Selectors = append(nextConfig.Dev.Logs.Selectors, next.LogsSelector{
				ImageSelector: fmt.Sprintf("image(%s):tag(%s)", rp, rp),
			})
		}
	}

	// terminal
	if c.Dev.Terminal != nil && c.Dev.Terminal.ImageName != "" {
		if nextConfig.Dev.Terminal == nil {
			nextConfig.Dev.Terminal = &next.Terminal{}
		}

		nextConfig.Dev.Terminal.ImageSelector = fmt.Sprintf("image(%s):tag(%s)", c.Dev.Terminal.ImageName, c.Dev.Terminal.ImageName)
	}

	// hooks
	for i, h := range c.Hooks {
		if h.Where.Container != nil {
			newContainer := &next.HookContainer{}
			err = util.Convert(h.Where.Container, newContainer)
			if err != nil {
				return nil, err
			}

			nextConfig.Hooks[i].Container = newContainer
		}

		events := []string{}
		if h.When != nil {
			if h.When.After != nil {
				events = append(events, getEventsFrom("after:build", h.When.After.Images, false)...)
				events = append(events, getEventsFrom("after:deploy", h.When.After.Deployments, false)...)
				events = append(events, getEventsFrom("after:purge", h.When.After.PurgeDeployments, false)...)
				if h.When.After.PullSecrets != "" {
					events = append(events, "after:createPullSecrets")
				}
				if h.When.After.Dependencies != "" {
					events = append(events, "after:deployDependencies")
				}
				events = append(events, getEventsFrom("after:initialSync", h.When.After.InitialSync, true)...)
			}
			if h.When.Before != nil {
				events = append(events, getEventsFrom("before:build", h.When.Before.Images, false)...)
				events = append(events, getEventsFrom("before:deploy", h.When.Before.Deployments, false)...)
				events = append(events, getEventsFrom("before:purge", h.When.Before.PurgeDeployments, false)...)
				if h.When.Before.PullSecrets != "" {
					events = append(events, "before:createPullSecrets")
				}
				if h.When.Before.Dependencies != "" {
					events = append(events, "before:deployDependencies")
				}
				events = append(events, getEventsFrom("before:initialSync", h.When.Before.InitialSync, true)...)
			}
			if h.When.OnError != nil {
				events = append(events, getEventsFrom("error:build", h.When.OnError.Images, true)...)
				events = append(events, getEventsFrom("error:deploy", h.When.OnError.Deployments, true)...)
				events = append(events, getEventsFrom("error:purge", h.When.OnError.PurgeDeployments, true)...)
				if h.When.OnError.PullSecrets != "" {
					events = append(events, "error:createPullSecrets")
				}
				if h.When.OnError.Dependencies != "" {
					events = append(events, "error:deployDependencies")
				}
				events = append(events, getEventsFrom("error:initialSync", h.When.OnError.InitialSync, true)...)
			}
		}
		nextConfig.Hooks[i].Events = events
	}

	return nextConfig, nil
}

func getEventsFrom(base string, val string, each bool) []string {
	if val == "" {
		return []string{}
	}
	if val == "all" && each {
		return []string{base + ":*"}
	}
	if val == "all" {
		return []string{base}
	}

	s := strings.Split(val, ",")
	out := []string{}
	for _, v := range s {
		out = append(out, base+":"+strings.TrimSpace(v))
	}

	return out
}
