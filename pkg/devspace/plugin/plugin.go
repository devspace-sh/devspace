package plugin

import (
	"encoding/base32"
	"fmt"
	"github.com/blang/semver"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/util/exit"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

func init() {
	if runtime.GOOS == "windows" {
		PluginBinary += ".exe"
	}
}

type Interface interface {
	Add(path, version string) error
	Update(name, version string) error
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

func (c *client) Add(path, version string) error {
	metadata, err := c.Get(path)
	if err != nil {
		return err
	} else if metadata != nil {
		return fmt.Errorf("plugin %s already exists", path)
	}

	return c.install(path, version)
}

func (c *client) install(path, version string) error {
	metadata, err := c.installer.DownloadMetadata(path, version)
	if err != nil {
		return errors.Wrap(err, "download metadata")
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
		return fmt.Errorf("plugin %s does not support %s/%s", metadata.Name, runtime.GOOS, runtime.GOARCH)
	}

	pluginFolder, err := c.PluginFolder()
	if err != nil {
		return err
	}

	pluginFolder = filepath.Join(pluginFolder, Encode(path))
	err = os.MkdirAll(pluginFolder, 0755)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(pluginFolder, pluginYaml), out, 0666)
	if err != nil {
		return err
	}

	outBinaryPath := filepath.Join(pluginFolder, PluginBinary)
	err = c.installer.DownloadBinary(path, version, binaryPath, outBinaryPath)
	if err != nil {
		return errors.Wrap(err, "download plugin binary")
	}

	// make the file executable
	_ = os.Chmod(outBinaryPath, 0755)
	return nil
}

func (c *client) Update(name, version string) error {
	path, metadata, err := c.GetByName(name)
	if err != nil {
		return err
	} else if metadata == nil {
		return fmt.Errorf("couldn't find plugin %s", name)
	}

	oldVersion, err := c.parseVersion(metadata.Version)
	if err != nil {
		return errors.Wrap(err, "parse old version")
	}

	newMetadata, err := c.installer.DownloadMetadata(path, version)
	if err != nil {
		return err
	}

	newVersion, err := c.parseVersion(newMetadata.Version)
	if err != nil {
		return errors.Wrap(err, "parse new version")
	}

	if oldVersion.EQ(newVersion) {
		return fmt.Errorf("no update for plugin found")
	} else if oldVersion.GT(newVersion) {
		return fmt.Errorf("new version is older than existing version")
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

func ExecutePluginHook(plugins []Metadata, event, kubeContext, namespace string) error {
	for _, plugin := range plugins {
		pluginFolder := plugin.PluginFolder
		for _, pluginHook := range plugin.Hooks {
			if pluginHook.Event == event {
				err := CallPluginExecutable(filepath.Join(pluginFolder, PluginBinary), pluginHook.BaseArgs, map[string]string{
					"DEVSPACE_PLUGIN_KUBE_CONTEXT_FLAG":   kubeContext,
					"DEVSPACE_PLUGIN_KUBE_NAMESPACE_FLAG": namespace,
				}, os.Stdout)
				if err != nil {
					return err
				}
			}
		}
	}

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
