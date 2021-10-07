package dev

import (
	"github.com/loft-sh/devspace/pkg/devspace/hook"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/devspace/services"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

func UI(servicesClient services.Client, port int) error {
	logger := servicesClient.Log()
	logger.StartWait("Starting the ui server...")
	defer logger.StopWait()

	var defaultPort *int
	if port != 0 {
		defaultPort = &port
	}

	// Create server
	uiLogger := log.GetFileLogger("ui")
	serv, err := server.NewServer(servicesClient.Config(), servicesClient.Dependencies(), "localhost", false, servicesClient.KubeClient().CurrentContext(), servicesClient.KubeClient().Namespace(), defaultPort, uiLogger)
	logger.StopWait()
	if err != nil {
		logger.Warnf("Couldn't start UI server: %v", err)
	} else {
		// Start server
		go func() { _ = serv.ListenAndServe() }()

		logger.WriteString("\n#########################################################\n")
		logger.Infof("DevSpace UI available at: %s", ansi.Color("http://"+serv.Server.Addr, "white+b"))
		logger.WriteString("#########################################################\n\n")
	}
	return nil
}

func Sync(servicesClient services.Client, interrupt chan error, printSyncLog, verbose bool) error {
	pluginErr := hook.ExecuteHooks(servicesClient.KubeClient(), servicesClient.Config(), servicesClient.Dependencies(), map[string]interface{}{}, servicesClient.Log(), "devCommand:before:sync", "dev.beforeSync")
	if pluginErr != nil {
		return pluginErr
	}

	err := servicesClient.StartSync(interrupt, printSyncLog, verbose, services.DefaultPrefixFn)
	if err != nil {
		return errors.Wrap(err, "start sync")
	}

	// start in dependencies
	for _, d := range servicesClient.Dependencies() {
		if d.DependencyConfig().Dev == nil || !d.DependencyConfig().Dev.Sync {
			continue
		}
		err = d.StartSync(servicesClient.KubeClient(), interrupt, printSyncLog, verbose, servicesClient.Log())
		if err != nil {
			return err
		}
	}

	pluginErr = hook.ExecuteHooks(servicesClient.KubeClient(), servicesClient.Config(), servicesClient.Dependencies(), map[string]interface{}{}, servicesClient.Log(), "devCommand:after:sync", "dev.afterSync")
	if pluginErr != nil {
		return pluginErr
	}

	return nil
}

func PortForwarding(servicesClient services.Client, interrupt chan error) error {
	pluginErr := hook.ExecuteHooks(servicesClient.KubeClient(), servicesClient.Config(), servicesClient.Dependencies(), map[string]interface{}{}, servicesClient.Log(), "devCommand:before:portForwarding", "dev.beforePortForwarding")
	if pluginErr != nil {
		return pluginErr
	}

	// start port forwarding
	err := servicesClient.StartPortForwarding(interrupt, services.DefaultPrefixFn)
	if err != nil {
		return errors.Errorf("Unable to start portforwarding: %v", err)
	}
	for _, d := range servicesClient.Dependencies() {
		if d.DependencyConfig().Dev == nil || !d.DependencyConfig().Dev.Ports {
			continue
		}
		err = d.StartPortForwarding(servicesClient.KubeClient(), interrupt, servicesClient.Log())
		if err != nil {
			return err
		}
	}

	pluginErr = hook.ExecuteHooks(servicesClient.KubeClient(), servicesClient.Config(), servicesClient.Dependencies(), map[string]interface{}{}, servicesClient.Log(), "devCommand:after:portForwarding", "dev.afterPortForwarding")
	if pluginErr != nil {
		return pluginErr
	}
	return nil
}

func ReplacePods(servicesClient services.Client) error {
	pluginErr := hook.ExecuteHooks(servicesClient.KubeClient(), servicesClient.Config(), servicesClient.Dependencies(), map[string]interface{}{}, servicesClient.Log(), "devCommand:before:replacePods", "dev.beforeReplacePods")
	if pluginErr != nil {
		return pluginErr
	}

	// replace pods
	err := servicesClient.ReplacePods(services.DefaultPrefixFn)
	if err != nil {
		return err
	}
	for _, d := range servicesClient.Dependencies() {
		if d.DependencyConfig().Dev == nil || !d.DependencyConfig().Dev.ReplacePods {
			continue
		}
		err = d.ReplacePods(servicesClient.KubeClient(), servicesClient.Log())
		if err != nil {
			return err
		}
	}

	pluginErr = hook.ExecuteHooks(servicesClient.KubeClient(), servicesClient.Config(), servicesClient.Dependencies(), map[string]interface{}{}, servicesClient.Log(), "devCommand:after:replacePods", "dev.afterReplacePods")
	if pluginErr != nil {
		return pluginErr
	}
	return nil
}
