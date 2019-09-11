package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

const cliDocsDir = "./docs/pages/cli/commands"
const headerTemplate = `---
title: "%s"
sidebar_label: %s
---

`

var fixSynopsisRegexp = regexp.MustCompile("(?smi)(## devspace.*?\n)(.*?)#(## Synopsis\n*\\s*)(.*?)(\\s*\n\n)((```)(.*?))?#(## Options)(.*?)#(## SEE ALSO)(\\s*\\* \\[devspace\\][^\n]*)?")

// Run executes the command logic
func (cmd *GenDocsCmd) Run(cobraCmd *cobra.Command, args []string) {
	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		command := strings.Split(base, "_")
		title := strings.Join(command, " ")
		sidebarLabel := title
		l := len(command)

		if l > 2 {
			sidebarLabel = command[l-1]
		}

		return fmt.Sprintf(headerTemplate, "Command: "+title, sidebarLabel)
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/docs/cli/commands/" + strings.ToLower(base)
	}

	err := doc.GenMarkdownTreeCustom(rootCmd, cliDocsDir, filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}

	err = filepath.Walk(cliDocsDir, func(path string, info os.FileInfo, err error) error {
		stat, err := os.Stat(path)
		if stat.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}

		newContents := fixSynopsisRegexp.ReplaceAllString(string(content), "$2$3$7$8```\n$4\n```\n$9$10## See Also")

		err = ioutil.WriteFile(path, []byte(newContents), 0)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
