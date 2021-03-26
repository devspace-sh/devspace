package plugin

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/exit"
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	yaml2 "gopkg.in/yaml.v2"

	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var encoding = base32.StdEncoding.WithPadding('0')

const pluginYaml = "plugin.yaml"

var PluginBinary = "binary"

const PluginCommandAnnotation = "devspace.sh/is-plugin"

const (
	KubeContextFlagEnv   = "DEVSPACE_PLUGIN_KUBE_CONTEXT_FLAG"
	KubeNamespaceFlagEnv = "DEVSPACE_PLUGIN_KUBE_NAMESPACE_FLAG"
	ConfigEnv            = "DEVSPACE_PLUGIN_CONFIG"
	OsArgsEnv            = "DEVSPACE_PLUGIN_OS_ARGS"
	CommandEnv           = "DEVSPACE_PLUGIN_COMMAND"
	CommandLineEnv       = "DEVSPACE_PLUGIN_COMMAND_LINE"
	CommandFlagsEnv      = "DEVSPACE_PLUGIN_COMMAND_FLAGS"
	CommandArgsEnv       = "DEVSPACE_PLUGIN_COMMAND_ARGS"
)

func init() {
	if runtime.GOOS == "windows" {
		PluginBinary += ".exe"
	}
}

type NewestVersionError struct {
	version string
}

func (n *NewestVersionError) Error() string {
	return "Current binary is the latest version: " + n.version
}

type Interface interface {
	Add(path, version string) (*Metadata, error)
	GetByName(name string) (string, *Metadata, error)
	Update(name, version string) (*Metadata, error)
	Remove(name string) error

	List() ([]Metadata, error)
}

type client struct {
	installer Installer

	log log.Logger
}

func NewClient(log log.Logger) Interface {
	return &client{
		installer: NewInstaller(),
		log:       log,
	}
}

func (c *client) PluginFolder() (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, constants.DefaultHomeDevSpaceFolder, PluginFolder), nil
}

func (c *client) Add(path, version string) (*Metadata, error) {
	metadata, err := c.Get(path)
	if err != nil {
		return nil, err
	} else if metadata != nil {
		return nil, fmt.Errorf("plugin %s already exists", path)
	}

	return c.install(path, version)
}

func (c *client) install(path, version string) (*Metadata, error) {
	metadata, err := c.installer.DownloadMetadata(path, version)
	if err != nil {
		return nil, errors.Wrap(err, "download metadata")
	}

	// find binary for system
	found := false
	binaryPath := ""
	for _, binary := range metadata.Binaries {
		if binary.OS == runtime.GOOS && binary.Arch == runtime.GOARCH {
			found = true
			binaryPath = binary.Path
			break
		}
	}
	if found == false {
		return nil, fmt.Errorf("plugin %s does not support %s/%s", metadata.Name, runtime.GOOS, runtime.GOARCH)
	}

	pluginFolder, err := c.PluginFolder()
	if err != nil {
		return nil, err
	}

	pluginFolder = filepath.Join(pluginFolder, Encode(path))
	err = os.MkdirAll(pluginFolder, 0755)
	if err != nil {
		return nil, err
	}

	out, err := yaml.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(filepath.Join(pluginFolder, pluginYaml), out, 0666)
	if err != nil {
		return nil, err
	}

	outBinaryPath := filepath.Join(pluginFolder, PluginBinary)
	err = c.installer.DownloadBinary(path, version, binaryPath, outBinaryPath)
	if err != nil {
		return nil, errors.Wrap(err, "download plugin binary")
	}

	// make the file executable
	_ = os.Chmod(outBinaryPath, 0755)
	metadata.PluginFolder = pluginFolder
	return metadata, nil
}

func (c *client) Update(name, version string) (*Metadata, error) {
	path, metadata, err := c.GetByName(name)
	if err != nil {
		return nil, err
	} else if metadata == nil {
		return nil, fmt.Errorf("couldn't find plugin %s", name)
	}

	oldVersion, err := c.parseVersion(metadata.Version)
	if err != nil {
		return nil, errors.Wrap(err, "parse old version")
	}

	newMetadata, err := c.installer.DownloadMetadata(path, version)
	if err != nil {
		return nil, err
	}

	newVersion, err := c.parseVersion(newMetadata.Version)
	if err != nil {
		return nil, errors.Wrap(err, "parse new version")
	}

	if oldVersion.EQ(newVersion) {
		return nil, &NewestVersionError{newVersion.String()}
	} else if oldVersion.GT(newVersion) {
		return nil, fmt.Errorf("new version is older than existing version")
	}

	c.log.Infof("Updating plugin %s to version %s", name, newMetadata.Version)
	return c.install(path, version)
}

func (c *client) parseVersion(version string) (semver.Version, error) {
	if len(version) == 0 {
		return semver.Version{}, fmt.Errorf("version is empty")
	}
	if version[0] == 'v' {
		version = version[1:]
	}

	return semver.Parse(version)
}

func (c *client) Remove(name string) error {
	path, metadata, err := c.GetByName(name)
	if err != nil {
		return err
	} else if metadata == nil {
		return fmt.Errorf("couldn't find plugin %s", name)
	}

	pluginFolder, err := c.PluginFolder()
	if err != nil {
		return err
	}

	return os.RemoveAll(filepath.Join(pluginFolder, Encode(path)))
}

func (c *client) List() ([]Metadata, error) {
	pluginFolder, err := c.PluginFolder()
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(pluginFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return []Metadata{}, nil
		}

		return nil, err
	}

	plugins, err := ioutil.ReadDir(pluginFolder)
	if err != nil {
		return nil, err
	}

	retMetadatas := []Metadata{}
	for _, plugin := range plugins {
		pFolder := filepath.Join(pluginFolder, plugin.Name())
		metadataFileContents, err := ioutil.ReadFile(filepath.Join(pFolder, pluginYaml))
		if os.IsNotExist(err) {
			_ = os.RemoveAll(filepath.Join(pluginFolder, plugin.Name()))
			continue
		}

		metadata := Metadata{}
		err = yaml.Unmarshal(metadataFileContents, &metadata)
		if err != nil {
			c.log.Warnf("Error parsing plugin.yaml for plugin %s: %v", plugin, err)
			continue
		}

		metadata.PluginFolder = pFolder
		retMetadatas = append(retMetadatas, metadata)
	}

	return retMetadatas, nil
}

func (c *client) GetByName(name string) (string, *Metadata, error) {
	pluginFolder, err := c.PluginFolder()
	if err != nil {
		return "", nil, err
	}

	plugins, err := ioutil.ReadDir(pluginFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}

		return "", nil, err
	}

	for _, plugin := range plugins {
		metadataFileContents, err := ioutil.ReadFile(filepath.Join(pluginFolder, plugin.Name(), pluginYaml))
		if os.IsNotExist(err) {
			_ = os.RemoveAll(filepath.Join(pluginFolder, plugin.Name()))
			continue
		}

		metadata := Metadata{}
		err = yaml.Unmarshal(metadataFileContents, &metadata)
		if err != nil {
			c.log.Warnf("Error parsing plugin.yaml for plugin %s: %v", plugin, err)
			continue
		}

		if metadata.Name == name {
			decoded, err := Decode(plugin.Name())
			if err != nil {
				return "", nil, errors.Wrap(err, "decode plugin path")
			}

			metadata.PluginFolder = filepath.Join(pluginFolder, plugin.Name())
			return string(decoded), &metadata, nil
		}
	}

	return "", nil, nil
}

func (c *client) Get(path string) (*Metadata, error) {
	pluginFolder, err := c.PluginFolder()
	if err != nil {
		return nil, err
	}

	out, err := ioutil.ReadFile(filepath.Join(pluginFolder, Encode(path), pluginYaml))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	metadata := Metadata{}
	err = yaml.Unmarshal(out, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func Encode(path string) string {
	return encoding.EncodeToString([]byte(path))
}

func Decode(encoded string) ([]byte, error) {
	return encoding.DecodeString(encoded)
}

func AddPluginCommands(base *cobra.Command, plugins []Metadata, subCommand string) {
	for _, plugin := range plugins {
		pluginFolder := plugin.PluginFolder
		for _, pluginCommand := range plugin.Commands {
			if pluginCommand.SubCommand == subCommand {
				md := pluginCommand
				if md.Usage == "" {
					md.Usage = fmt.Sprintf("the %q plugin", plugin.Name)
				}

				c := &cobra.Command{
					Use:   md.Name,
					Short: md.Usage,
					Long:  md.Description,
					RunE: func(cmd *cobra.Command, args []string) error {
						newArgs := []string{}
						newArgs = append(newArgs, md.BaseArgs...)
						newArgs = append(newArgs, args...)
						return CallPluginExecutable(filepath.Join(pluginFolder, PluginBinary), newArgs, nil, os.Stdout)
					},
					// This passes all the flags to the subcommand.
					DisableFlagParsing: true,
					Annotations: map[string]string{
						PluginCommandAnnotation: "true",
					},
				}

				base.AddCommand(c)
			}
		}
	}
}

func ExecutePluginHook(plugins []Metadata, cobraCmd *cobra.Command, args []string, event, kubeContext, namespace string, config *latest.Config) error {
	configStr := ""
	if config != nil {
		configBytes, err := yaml2.Marshal(config)
		if err != nil {
			return err
		}

		configStr = string(configBytes)
	}

	osArgsBytes, err := json.Marshal(os.Args)
	if err != nil {
		return err
	}

	// build environment variables
	env := map[string]string{
		CommandEnv:     cobraCmd.Use,
		CommandLineEnv: cobraCmd.UseLine(),
		OsArgsEnv:      string(osArgsBytes),
	}
	if kubeContext != "" {
		env[KubeContextFlagEnv] = kubeContext
	}
	if namespace != "" {
		env[KubeNamespaceFlagEnv] = namespace
	}
	if configStr != "" {
		env[ConfigEnv] = configStr
	}

	// Flags
	flags := []string{}
	cobraCmd.Flags().Visit(func(f *pflag.Flag) {
		flags = append(flags, "--"+f.Name)
		flags = append(flags, f.Value.String())
	})
	if len(flags) > 0 {
		flagsStr, err := json.Marshal(flags)
		if err != nil {
			return err
		}

		env[CommandFlagsEnv] = string(flagsStr)
	}

	// Args
	if len(args) > 0 {
		argsStr, err := json.Marshal(args)
		if err != nil {
			return err
		}
		if string(argsStr) != "" {
			env[CommandArgsEnv] = string(argsStr)
		}
	}

	for _, plugin := range plugins {
		pluginFolder := plugin.PluginFolder
		for _, pluginHook := range plugin.Hooks {
			if strings.TrimSpace(pluginHook.Event) == event {
				if pluginHook.Background {
					err = CallPluginExecutableInBackground(filepath.Join(pluginFolder, PluginBinary), pluginHook.BaseArgs, env)
				} else {
					err = CallPluginExecutable(filepath.Join(pluginFolder, PluginBinary), pluginHook.BaseArgs, env, os.Stdout)
				}
				if err != nil {
					return err
				}
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
		if strings.Index(err.Error(), "no such file or directory") != -1 {
			return fmt.Errorf("the plugin's binary was not found (%v). Please uninstall and reinstall the plugin and make sure there are no other conflicting plugins installed (run 'devspace list plugins' to see all installed plugins)", err)
		}

		return err
	}

	go func() {
		err := prog.Wait()
		if err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				os.Stderr.Write([]byte(fmt.Sprintf("Hook %s failed (code: %d): %s", main+" "+strings.Join(argv, " "), eerr.ExitCode(), stderrOut.String())))
			}
		}
	}()
	return nil
}

// This function is used to setup the environment for the plugin and then
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
		} else if strings.Index(err.Error(), "no such file or directory") != -1 {
			return fmt.Errorf("the plugin's binary was not found (%v). Please uninstall and reinstall the plugin and make sure there are no other conflicting plugins installed (run 'devspace list plugins' to see all installed plugins)", err)
		}

		return err
	}

	return nil
}
