package tanka

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
)

type TankaEnvironment interface {
	Show(ctx devspacecontext.Context, out io.Writer) error
	Diff(ctx devspacecontext.Context) (string, error)
	Apply(ctx devspacecontext.Context) error
	Prune(ctx devspacecontext.Context) error
	Delete(ctx devspacecontext.Context) error
	Update(ctx devspacecontext.Context) error
	Install(ctx devspacecontext.Context) error
}

type tankaEnvironmentImpl struct {
	name         string
	namespace    string
	tkBinaryPath string
	jbBinaryPath string
	args         []string
	flags        []string
	rootDir      string
	stdout       io.Writer
	stderr       io.Writer
}

// NewTankaEnvironment generates a new tanka environment which allows show/diff/apply operations.
func NewTankaEnvironment(config *latest.TankaConfig) TankaEnvironment {
	args := []string{}
	flags := []string{}
	// Map configuration to CLI arguments and flags
	if config.EnvironmentPath != "" {
		args = append(args, config.EnvironmentPath)
	} else {
		args = append(args, ".")
	}
	if config.EnvironmentName != "" {
		flags = append(flags, fmt.Sprintf("--name=%s", config.EnvironmentName))
	}
	for k, v := range config.ExternalCodeVariables {
		flags = append(flags, fmt.Sprintf("--ext-code=%s=%s", k, v))
	}
	for k, v := range config.ExternalStringVariables {
		flags = append(flags, fmt.Sprintf("--ext-str=%s=%s", k, v))
	}
	for _, v := range config.TopLevelCode {
		flags = append(flags, fmt.Sprintf("--tla-code=%s", v))
	}
	for _, v := range config.TopLevelString {
		flags = append(flags, fmt.Sprintf("--tla-str=%s", v))
	}
	if config.Target != "" {
		flags = append(flags, fmt.Sprintf("--target=%s", config.Target))
	}
	tkPath := config.TankaBinaryPath
	if tkPath == "" {
		tkPath = "tk" // fallback to default binary; resolved by the OS
	}
	jbPath := config.JsonnetBundlerBinaryPath
	if jbPath == "" {
		jbPath = "jb" // fallback to default binary; resolved by the OS
	}
	return &tankaEnvironmentImpl{
		tkBinaryPath: tkPath,
		jbBinaryPath: jbPath,
		args:         args,
		flags:        flags,
		rootDir:      config.Path,

		// Extract those fields from the wellknown configuration
		name:      config.ExternalStringVariables[ExtVarName],
		namespace: config.ExternalStringVariables[ExtVarNamespace],

		// Pass stdout/stderr -> can be replaced for testing
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// Apply implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Apply(ctx devspacecontext.Context) error {
	var err error

	out := new(bytes.Buffer)

	applyArgs := append([]string{"apply"}, t.args...)
	applyArgs = append(applyArgs, t.flags...)
	applyArgs = append(applyArgs, "--auto-approve=always")

	ctx.Log().Debugf("Tanka apply arguments: %v", applyArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, applyArgs...)
	cmd.Stderr = out
	cmd.Stdout = out
	cmd.Dir = t.rootDir

	err = cmd.Run()

	// Proxy output to stderr
	// Ignore if this fails or not. The output is wrapped in an error anyways
	_, _ = t.stderr.Write(out.Bytes())

	if err != nil {
		return fmt.Errorf(out.String())
	}

	return nil
}

// Diff implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Diff(ctx devspacecontext.Context) (string, error) {
	diffArgs := append([]string{"diff"}, t.args...)
	diffArgs = append(diffArgs, t.flags...)
	diffArgs = append(diffArgs, []string{"--exit-zero", "--summarize"}...)
	ctx.Log().Debugf("Tanka diff arguments: %v", diffArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, diffArgs...)
	cmd.Dir = t.rootDir

	out, err := cmd.CombinedOutput()

	return string(out), err
}

// Show implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Show(ctx devspacecontext.Context, out io.Writer) error {
	showArgs := append([]string{"show"}, t.args...)
	showArgs = append(showArgs, t.flags...)
	showArgs = append(showArgs, "--dangerous-allow-redirect")

	ctx.Log().Debugf("Tanka show arguments: %v", showArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, showArgs...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = t.rootDir

	return cmd.Run()
}

// Prune implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Prune(ctx devspacecontext.Context) error {
	pruneArgs := append([]string{"prune"}, t.args...)
	pruneArgs = append(pruneArgs, "--auto-approve=always")
	pruneArgs = append(pruneArgs, t.flags...)

	ctx.Log().Debugf("Tanka prune arguments: %v", pruneArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, pruneArgs...)
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	cmd.Dir = t.rootDir

	return cmd.Run()
}

// Delete implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Delete(ctx devspacecontext.Context) error {
	deleteArgs := append([]string{"delete"}, t.args...)
	deleteArgs = append(deleteArgs, "--auto-approve=always")
	deleteArgs = append(deleteArgs, t.flags...)

	ctx.Log().Debugf("Tanka delete arguments: %v", deleteArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, deleteArgs...)
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	cmd.Dir = t.rootDir

	return cmd.Run()
}

func (t *tankaEnvironmentImpl) Install(ctx devspacecontext.Context) error {
	installArgs := []string{"install"}

	ctx.Log().Debugf("Jb install")
	cmd := exec.CommandContext(ctx.Context(), t.jbBinaryPath, installArgs...)
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	cmd.Dir = t.rootDir

	return cmd.Run()
}

func (t *tankaEnvironmentImpl) Update(ctx devspacecontext.Context) error {
	installArgs := []string{"update"}

	ctx.Log().Debugf("Jb update")
	cmd := exec.CommandContext(ctx.Context(), t.jbBinaryPath, installArgs...)
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	cmd.Dir = t.rootDir

	return cmd.Run()
}
