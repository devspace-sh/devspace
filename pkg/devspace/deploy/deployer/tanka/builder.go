package tanka

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sort"

	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
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
	targetFlags  []string
	rootDir      string
	stdout       io.Writer
	stderr       io.Writer
}

// NewTankaEnvironment generates a new tanka environment which allows show/diff/apply operations.
func NewTankaEnvironment(config *latest.TankaConfig) TankaEnvironment {
	args := []string{}
	flags := []string{}
	targetFlags := []string{}
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
	for _, v := range config.Targets {
		targetFlags = append(targetFlags, fmt.Sprintf("--target=%s", v))
	}
	sort.Strings(flags)
	tkPath := config.TankaBinaryPath
	if tkPath == "" {
		tkPath = tkDefaultCommand // fallback to default binary; resolved by the OS
	}
	jbPath := config.JsonnetBundlerBinaryPath
	if jbPath == "" {
		jbPath = jbDefaultCommand // fallback to default binary; resolved by the OS
	}
	return &tankaEnvironmentImpl{
		tkBinaryPath: tkPath,
		jbBinaryPath: jbPath,
		args:         args,
		flags:        flags,
		targetFlags:  targetFlags,
		rootDir:      config.Path,

		// Extract those fields from the wellknown configuration
		name:      config.ExternalStringVariables[ExtVarName],
		namespace: config.ExternalStringVariables[ExtVarNamespace],

		// Pass stdout/stderr -> can be replaced for testing
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

func eval(ctx devspacecontext.Context, arg string) string {
	resolver := runtimevar.NewRuntimeResolver(ctx.WorkingDir(), true)
	_, newArg, err := resolver.FillRuntimeVariablesWithRebuild(ctx.Context(), arg, ctx.Config(), ctx.Dependencies())
	if err != nil {
		ctx.Log().Warnf("Could not be able to resolve variables from '%v': %v", arg, err)
		return arg
	}
	return fmt.Sprint(newArg)
}

// Build arguments with Runtime vars
func buildArgs(ctx devspacecontext.Context, arguments []string) []string {
	newArgs := []string{}

	for _, arg := range arguments {
		newArgs = append(newArgs, eval(ctx, arg))
	}
	return newArgs
}

// Apply implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Apply(ctx devspacecontext.Context) error {
	var err error

	out := new(bytes.Buffer)

	applyArgs := append([]string{"apply"}, t.args...)
	applyArgs = append(applyArgs, t.flags...)
	applyArgs = append(applyArgs, t.targetFlags...)
	applyArgs = append(applyArgs, "--auto-approve=always")

	applyArgs = buildArgs(ctx, applyArgs)

	err = t.Show(ctx, out)
	if err != nil {
		return err
	}

	if out.String() == "" {
		ctx.Log().Warnf("Warning: No manifests detected. Skipping apply: %v", applyArgs)
		return nil
	}
	out.Reset()

	ctx.Log().Debugf("Tanka apply arguments: %v", applyArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, applyArgs...)
	cmd.Stderr = out
	cmd.Stdout = out
	cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))

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
	diffArgs = append(diffArgs, t.targetFlags...)
	diffArgs = append(diffArgs, []string{"--exit-zero", "--summarize"}...)
	diffArgs = buildArgs(ctx, diffArgs)

	ctx.Log().Debugf("Tanka diff arguments: %v", diffArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, diffArgs...)
	cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))

	out, err := cmd.CombinedOutput()

	return string(out), err
}

// Show implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Show(ctx devspacecontext.Context, out io.Writer) error {
	showArgs := append([]string{"show"}, t.args...)
	showArgs = append(showArgs, t.flags...)
	showArgs = append(showArgs, t.targetFlags...)
	showArgs = append(showArgs, "--dangerous-allow-redirect")
	showArgs = buildArgs(ctx, showArgs)

	ctx.Log().Debugf("Tanka show arguments: %v", showArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, showArgs...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))

	return cmd.Run()
}

// Prune implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Prune(ctx devspacecontext.Context) error {
	pruneArgs := append([]string{"prune"}, t.args...)
	pruneArgs = append(pruneArgs, "--auto-approve=always")
	pruneArgs = append(pruneArgs, t.flags...)
	pruneArgs = buildArgs(ctx, pruneArgs)

	ctx.Log().Debugf("Tanka prune arguments: %v", pruneArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, pruneArgs...)
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))

	return cmd.Run()
}

// Delete implements TankaEnvironment.
func (t *tankaEnvironmentImpl) Delete(ctx devspacecontext.Context) error {
	deleteArgs := append([]string{"delete"}, t.args...)
	deleteArgs = append(deleteArgs, "--auto-approve=always")
	deleteArgs = append(deleteArgs, t.flags...)
	deleteArgs = append(deleteArgs, t.targetFlags...)
	deleteArgs = buildArgs(ctx, deleteArgs)

	ctx.Log().Debugf("Tanka delete arguments: %v", deleteArgs)
	cmd := exec.CommandContext(ctx.Context(), t.tkBinaryPath, deleteArgs...)
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))

	return cmd.Run()
}

// those functions run in a per-execution global sync.Once, as the called binary modifies the file structure
// and is not thread safe

func (t *tankaEnvironmentImpl) Install(ctx devspacecontext.Context) error {
	var err error = nil
	installArgs := []string{"install"}

	GetOnce("install", path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))).Do(func() {
		ctx.Log().Debugf("Jb install")
		cmd := exec.CommandContext(ctx.Context(), t.jbBinaryPath, installArgs...)
		cmd.Stdout = t.stdout
		cmd.Stderr = t.stderr
		cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))
		err = cmd.Run()
	})

	return err
}

func (t *tankaEnvironmentImpl) Update(ctx devspacecontext.Context) error {
	var err error = nil
	installArgs := []string{"update"}

	GetOnce("update", path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))).Do(func() {
		ctx.Log().Debugf("Jb update")
		cmd := exec.CommandContext(ctx.Context(), t.jbBinaryPath, installArgs...)
		cmd.Stdout = t.stdout
		cmd.Stderr = t.stderr
		cmd.Dir = path.Join(ctx.WorkingDir(), eval(ctx, t.rootDir))
		err = cmd.Run()
	})

	return err
}
