package plugin

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func logErrorf(message string, args ...interface{}) {
	fileLogger := log.GetFileLogger("plugin")
	fileLogger.Errorf(message, args...)
}

const (
	SessionEnv           = "DEVSPACE_PLUGIN_SESSION"
	ExecutionIDEnv       = "DEVSPACE_PLUGIN_EXECUTION_ID"
	KubeContextFlagEnv   = "DEVSPACE_PLUGIN_KUBE_CONTEXT_FLAG"
	KubeNamespaceFlagEnv = "DEVSPACE_PLUGIN_KUBE_NAMESPACE_FLAG"
	ConfigEnv            = "DEVSPACE_PLUGIN_CONFIG"
	ConfigPathEnv        = "DEVSPACE_PLUGIN_CONFIG_PATH"
	ConfigVarsEnv        = "DEVSPACE_PLUGIN_CONFIG_VARS"
	OsArgsEnv            = "DEVSPACE_PLUGIN_OS_ARGS"
	CommandEnv           = "DEVSPACE_PLUGIN_COMMAND"
	CommandLineEnv       = "DEVSPACE_PLUGIN_COMMAND_LINE"
	CommandFlagsEnv      = "DEVSPACE_PLUGIN_COMMAND_FLAGS"
	CommandArgsEnv       = "DEVSPACE_PLUGIN_COMMAND_ARGS"
)

var plugins []Metadata

var pluginContextLock sync.Mutex
var pluginContext map[string]string = map[string]string{}

var pluginsOnce sync.Once

func SetPlugins(p []Metadata) {
	pluginsOnce.Do(func() {
		plugins = p

		pluginContextLock.Lock()
		defer pluginContextLock.Unlock()

		pluginContext[SessionEnv] = uuid.New().String()
		pluginContext[ExecutionIDEnv] = pluginContext[SessionEnv]
	})
}

var kubeContextOnce sync.Once

func SetPluginKubeContext(kubeContext, namespace string) {
	kubeContextOnce.Do(func() {
		pluginContextLock.Lock()
		defer pluginContextLock.Unlock()

		if kubeContext != "" {
			pluginContext[KubeContextFlagEnv] = kubeContext
		}
		if namespace != "" {
			pluginContext[KubeNamespaceFlagEnv] = namespace
		}
	})
}

var commandOnce sync.Once

func SetPluginCommand(cobraCmd *cobra.Command, args []string) {
	commandOnce.Do(func() {
		pluginContextLock.Lock()
		defer pluginContextLock.Unlock()

		if cobraCmd == nil {
			return
		}

		osArgsBytes, err := json.Marshal(os.Args)
		if err != nil {
			logErrorf("marshal os args: %v", err)
			return
		}

		pluginContext[CommandEnv] = cobraCmd.Use
		pluginContext[CommandLineEnv] = cobraCmd.UseLine()
		pluginContext[OsArgsEnv] = string(osArgsBytes)

		// Flags
		flags := []string{}
		cobraCmd.Flags().Visit(func(f *pflag.Flag) {
			flags = append(flags, "--"+f.Name)
			flags = append(flags, f.Value.String())
		})
		if len(flags) > 0 {
			flagsStr, err := json.Marshal(flags)
			if err != nil {
				logErrorf("marshal flags: %v", err)
				return
			}

			pluginContext[CommandFlagsEnv] = string(flagsStr)
		}

		// Args
		if len(args) > 0 {
			argsStr, err := json.Marshal(args)
			if err != nil {
				logErrorf("marshal args: %v", err)
				return
			}
			if string(argsStr) != "" {
				pluginContext[CommandArgsEnv] = string(argsStr)
			}
		}
	})
}

var configOnce sync.Once

func SetPluginConfig(config config.Config) {
	configOnce.Do(func() {
		pluginContextLock.Lock()
		defer pluginContextLock.Unlock()

		if config == nil || config.Config() == nil {
			return
		}

		configBytes, err := yaml.Marshal(config.Config())
		if err != nil {
			logErrorf("error marshalling devspace.yaml: %v", err)
			return
		}

		pluginContext[ConfigEnv] = string(configBytes)
		pluginContext[ConfigPathEnv] = config.Path()

		varsBytes, err := json.Marshal(config.Variables())
		if err != nil {
			logErrorf("error marshalling config vars: %v", err)
			return
		}
		pluginContext[ConfigVarsEnv] = string(varsBytes)
	})
}

func LogExecutePluginHookWithContext(extraEnv map[string]interface{}, events ...string) {
	err := ExecutePluginHookWithContext(extraEnv, events...)
	if err != nil {
		logErrorf("%v", err)
	}
}

func ExecutePluginHookWithContext(extraEnv map[string]interface{}, events ...string) error {
	if len(plugins) == 0 {
		return nil
	}

	// apply global plugin context
	newEnv := map[string]string{}
	pluginContextLock.Lock()
	for k, v := range pluginContext {
		newEnv[k] = v
	}
	pluginContextLock.Unlock()

	// apply extra context
	convertedExtraEnv := ConvertExtraEnv("DEVSPACE_PLUGIN", extraEnv)
	for k, v := range convertedExtraEnv {
		newEnv[k] = v
	}

	for _, plugin := range plugins {
		for _, e := range events {
			newEnv["DEVSPACE_PLUGIN_EVENT"] = e
			err := executePluginHookAt(plugin, e, newEnv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ConvertExtraEnv(base string, extraEnv map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range extraEnv {
		if v == nil {
			continue
		}

		k = strings.TrimSpace(strings.ToUpper(base + "_" + k))
		switch t := v.(type) {
		case string:
			out[k] = t
		case int:
			out[k] = fmt.Sprintf("%d", t)
		case error:
			out[k] = fmt.Sprintf("%v", t)
		default:
			outBytes, err := json.Marshal(yamlutil.Convert(t))
			if err != nil {
				logErrorf("error marshal extra env %s: %v", k, err)
			}
			out[k] = string(outBytes)
		}
	}
	return out
}

func ExecutePluginHookAt(plugin Metadata, events ...string) error {
	for _, e := range events {
		// apply global plugin context
		newEnv := map[string]string{}
		pluginContextLock.Lock()
		for k, v := range pluginContext {
			newEnv[k] = v
		}
		pluginContextLock.Unlock()

		err := executePluginHookAt(plugin, e, newEnv)
		if err != nil {
			return err
		}
	}

	return nil
}

func executePluginHookAt(plugin Metadata, event string, env map[string]string) error {
	pluginFolder := plugin.PluginFolder
	for _, pluginHook := range plugin.Hooks {
		if strings.TrimSpace(pluginHook.Event) == event {
			var err error
			if pluginHook.Background {
				err = CallPluginExecutableInBackground(filepath.Join(pluginFolder, PluginBinary), pluginHook.BaseArgs, env)
			} else {
				err = CallPluginExecutable(filepath.Join(pluginFolder, PluginBinary), pluginHook.BaseArgs, env, os.Stdout)
			}
			if err != nil {
				return fmt.Errorf("error calling plugin hook %s at event %s: %v", plugin.Name, event, err)
			}
		}
	}
	return nil
}

func CallPluginExecutableInBackground(main string, argv []string, extraEnvVars map[string]string) error {
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

	stderrOut := &bytes.Buffer{}
	prog := exec.Command(main, argv...)
	prog.Env = env
	prog.Stderr = stderrOut
	if err := prog.Start(); err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("the plugin's binary was not found (%v). Please reinstall the plugin and make sure there are no other conflicting plugins installed (run 'devspace list plugins' to see all installed plugins)", err)
		}

		return err
	}

	go func() {
		err := prog.Wait()
		if err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				os.Stderr.Write([]byte(fmt.Sprintf("Plugin Hook %s failed (code: %d): %s", main+" "+strings.Join(argv, " "), eerr.ExitCode(), stderrOut.String())))
			}
		}
	}()
	return nil
}

// CallPluginExecutable is used to setup the environment for the plugin and then
// call the executable specified by the parameter 'main'
func CallPluginExecutable(main string, argv []string, extraEnvVars map[string]string, out io.Writer) error {
	env := os.Environ()
	for k, v := range extraEnvVars {
		env = append(env, k+"="+v)
	}

	prog := exec.Command(main, argv...)
	prog.Env = env
	prog.Stdin = os.Stdin
	prog.Stdout = out
	prog.Stderr = os.Stderr
	if err := prog.Run(); err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			os.Stderr.Write(eerr.Stderr)
			return &exit.ReturnCodeError{ExitCode: eerr.ExitCode()}
		} else if strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("the plugin's binary was not found (%v). Please uninstall and reinstall the plugin and make sure there are no other conflicting plugins installed (run 'devspace list plugins' to see all installed plugins)", err)
		}

		return err
	}

	return nil
}
