package penv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-ps"
)

var (
	bashShell = &shell{
		configFileName: filepath.Join(os.Getenv("HOME"), ".bashrc"),
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
				"export %s=%s",
				nv.Name, sh.quote(nv.Value),
			)
		},
		mkAppend: func(sh *shell, nv NameValue) string {
			return fmt.Sprintf(
				"export %s=${%s}${%s:+:}%s",
				nv.Name, nv.Name, nv.Name, sh.quote(nv.Value),
			)
		},
		mkUnset: func(sh *shell, nv NameValue) string {
			return fmt.Sprintf(
				"unset %s",
				nv.Name,
			)
		},
	}
)

type (
	bashOp struct {
		op        string
		nameValue NameValue
	}
	// BashDAO is a data access object for bash
	BashDAO struct{}
)

func init() {
	// For nitrous store the settings in .bash_profile since autoparts overrides
	// the settings
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".nitrousboxrc.sample")); err == nil {
		bashShell.configFileName = filepath.Join(os.Getenv("HOME"), ".bash_profile")
	}

	RegisterDAO(1000, func() bool {
		pid := os.Getpid()
		for pid > 0 {
			p, err := ps.FindProcess(pid)
			if err != nil || p == nil {
				break
			}
			if p.Executable() == "fish" {
				return false
			}
			if p.Executable() == "bash" {
				return true
			}
			pid = p.PPid()
		}
		return false
	}, bashShell)
}
