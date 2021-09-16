package v1beta10

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/config"
	next "github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
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
				events = append(events, getEventsFrom("after:build", h.When.After.Images)...)
				events = append(events, getEventsFrom("after:deploy", h.When.After.Deployments)...)
				events = append(events, getEventsFrom("after:purge", h.When.After.PurgeDeployments)...)
				events = append(events, "after:createAllPullSecrets")
				events = append(events, getEventsFrom("after:initialSync", h.When.After.InitialSync)...)
			}
			if h.When.Before != nil {
				events = append(events, getEventsFrom("before:build", h.When.Before.Images)...)
				events = append(events, getEventsFrom("before:deploy", h.When.Before.Deployments)...)
				events = append(events, getEventsFrom("before:purge", h.When.Before.PurgeDeployments)...)
				events = append(events, "before:createAllPullSecrets")
				events = append(events, getEventsFrom("before:initialSync", h.When.Before.InitialSync)...)
			}
			if h.When.OnError != nil {
				events = append(events, getEventsFrom("error:build", h.When.OnError.Images)...)
				events = append(events, getEventsFrom("error:deploy", h.When.OnError.Deployments)...)
				events = append(events, getEventsFrom("error:purge", h.When.OnError.PurgeDeployments)...)
				events = append(events, "error:createAllPullSecrets")
				events = append(events, getEventsFrom("error:initialSync", h.When.OnError.InitialSync)...)
			}
		}
		nextConfig.Hooks[i].Events = events
	}

	return nextConfig, nil
}

func getEventsFrom(base string, val string) []string {
	if val == "" {
		return []string{}
	}
	if val == "all" && strings.HasSuffix(base, "initialSync") {
		return []string{base + ":*"}
	}
	if val == "all" {
		return []string{base + "All"}
	}

	s := strings.Split(val, ",")
	out := []string{}
	for _, v := range s {
		out = append(out, base+":"+strings.TrimSpace(v))
	}

	return out
}
