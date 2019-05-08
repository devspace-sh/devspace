package penv

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-ps"
)

var (
	fishShell = &shell{
		configFileName: filepath.Join(os.Getenv("HOME"), ".config", "fish", "config.fish"),
		commentSigil:   " #",
		quote: func(value string) string {
			r := strings.NewReplacer(
				"\\", "\\\\",
				"'", "\\'",
				"\n", `'"\n"'`,
				"\r", `'"\r"'`,
			)
			return "'" + r.Replace(value) + "'"
		},
		mkSet: func(sh *shell, nv NameValue) string {
			return fmt.Sprintf(
				"set -Ux %s %s",
				nv.Name, sh.quote(nv.Value),
			)
		},
		mkAppend: func(sh *shell, nv NameValue) string {
			return fmt.Sprintf(
				"set -Ux %s $%s %s",
				nv.Name, nv.Name, sh.quote(nv.Value),
			)
		},
		mkUnset: func(sh *shell, nv NameValue) string {
			return fmt.Sprintf(
				"set -Ue %s",
				nv.Name,
			)
		},
	}
)

type fishReloader struct{ DAO }

func (fw fishReloader) Save(env *Environment) error {
	err := fw.DAO.Save(env)
	if err != nil {
		return err
	}
	return exec.Command("fish", "-c", "echo hi").Run()
}

func init() {
	RegisterDAO(1000, func() bool {
		pid := os.Getpid()
		for pid > 0 {
			p, err := ps.FindProcess(pid)
			if err != nil || p == nil {
				break
			}
			switch p.Executable() {
			case "fish":
				return true
			case "bash":
				return false
			case "zsh":
				return false
			}
			pid = p.PPid()
		}
		return false
	}, fishReloader{fishShell})
}
