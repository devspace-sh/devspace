package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// GenDocsCmd is a struct that defines a command call for "enter"
type GenDocsCmd struct{}

// newGenDocsCmd creates a new gen-docs command
func newGenDocsCmd() *cobra.Command {
	cmd := &GenDocsCmd{}

	genDocsCmd := &cobra.Command{
		Use:   "gen-docs",
		Short: "Generates docs pages for CLI commands",
		Long: `
#######################################################
################# devspace gen-docs ###################
#######################################################
Run this command to generate the documentation for all
CLI commands.

This command is not available in production.
#######################################################`,
		Run: cmd.Run,
	}

	return genDocsCmd
}

const headerTemplate = `---
title: "%s"
sidebar_title: %s
---

`

// Run executes the command logic
func (cmd *GenDocsCmd) Run(cobraCmd *cobra.Command, args []string) {
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		command := strings.Split(base, "_")
		title := strings.Join(command, " ")
		sidebarTitle := title
		l := len(command)

		if l > 0 {
			sidebarTitle = command[l-1]
		}

		return fmt.Sprintf(headerTemplate, "Command: "+title, sidebarTitle)
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/docs/cli/commands/" + strings.ToLower(base)
	}

	err := doc.GenMarkdownTreeCustom(rootCmd, "./tmp", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}
