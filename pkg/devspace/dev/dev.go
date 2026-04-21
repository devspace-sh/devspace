package dev

import (
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/pipeline/types"
	"github.com/loft-sh/devspace/pkg/devspace/server"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

func UI(ctx devspacecontext.Context, port int, showUI bool, protectUI bool, pipeline types.Pipeline) (*server.Server, error) {
	var defaultPort *int
	if port != 0 {
		defaultPort = &port
	}

	// Create server
	uiLogger := log.GetFileLogger("ui")
	serv, err := server.NewServer(ctx.WithLogger(uiLogger), "localhost", false, defaultPort, pipeline, protectUI)
	if err != nil {
		ctx.Log().Warnf("Couldn't start UI server: %v", err)
	} else {
		// Start server
		go func() {
			_ = serv.ListenAndServe()
		}()

		go func() {
			<-ctx.Context().Done()
			_ = serv.Server.Close()
		}()

		if showUI {
			ctx.Log().WriteString(logrus.InfoLevel, "\n#########################################################\n")
			ctx.Log().Infof("DevSpace UI available at: %s", ansi.Color(serv.BrowserURL(), "white+b"))
			ctx.Log().WriteString(logrus.InfoLevel, "#########################################################\n\n")
		}
	}
	return serv, nil
}
