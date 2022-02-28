package dev

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

func UI(ctx *devspacecontext.Context, port int) error {
	var defaultPort *int
	if port != 0 {
		defaultPort = &port
	}

	// Create server
	uiLogger := log.GetFileLogger("ui")
	serv, err := server.NewServer(ctx.Config, ctx.Dependencies, "localhost", false, ctx.KubeClient.CurrentContext(), ctx.KubeClient.Namespace(), defaultPort, uiLogger)
	if err != nil {
		ctx.Log.Warnf("Couldn't start UI server: %v", err)
	} else {
		// Start server
		go func() { _ = serv.ListenAndServe() }()

		ctx.Log.WriteString(logrus.InfoLevel, "\n#########################################################\n")
		ctx.Log.Infof("DevSpace UI available at: %s", ansi.Color("http://"+serv.Server.Addr, "white+b"))
		ctx.Log.WriteString(logrus.InfoLevel, "#########################################################\n\n")
	}
	return nil
}
