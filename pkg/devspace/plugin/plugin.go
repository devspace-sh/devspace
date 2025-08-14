package plugin

import (
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/blang/semver"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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

var devspaceVars = map[string]string{}
var devspaceVarsOnce sync.Once

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
	// resolve path if it's a local one
	var err error
	if isLocalReference(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
	}

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
	if !found {
		return nil, fmt.Errorf("plugin %s does not support %s/%s", metadata.Name, runtime.GOOS, runtime.GOARCH)
	}

	// download binary to temp folder and test it
	tempBinaryName := "./plugin-binary"
	if runtime.GOOS == "windows" {
		tempBinaryName += ".exe"
	}
	err = c.installer.DownloadBinary(path, version, binaryPath, tempBinaryName)
	if err != nil {
		return nil, errors.Wrap(err, "download plugin binary")
	}
	_ = os.Chmod(tempBinaryName, 0755)

	// test the binary
	absolutePath, err := filepath.Abs(tempBinaryName)
	if err != nil {
		return nil, err
	}
	o, err := exec.Command(absolutePath).Output()
	if err != nil {
		_ = os.Remove(absolutePath)
		return nil, fmt.Errorf("error executing plugin binary, make sure the plugin binary is executable and returns a zero exit code when run without arguments: %s => %v", string(o), err)
	}

	// create the plugin folder
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

	err = os.WriteFile(filepath.Join(pluginFolder, pluginYaml), out, 0666)
	if err != nil {
		return nil, err
	}

	outBinaryPath := filepath.Join(pluginFolder, PluginBinary)
	err = moveFile(tempBinaryName, outBinaryPath)
	if err != nil {
		return nil, err
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

	plugins, err := os.ReadDir(pluginFolder)
	if err != nil {
		return nil, err
	}

	retMetadatas := []Metadata{}
	for _, dirEntry := range plugins {
		plugin, err := dirEntry.Info()
		if err != nil {
			continue
		}

		pFolder := filepath.Join(pluginFolder, plugin.Name())
		metadataFileContents, err := os.ReadFile(filepath.Join(pFolder, pluginYaml))
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

	plugins, err := os.ReadDir(pluginFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}

		return "", nil, err
	}

	for _, dirEntry := range plugins {
		plugin, err := dirEntry.Info()
		if err != nil {
			continue
		}

		metadataFileContents, err := os.ReadFile(filepath.Join(pluginFolder, plugin.Name(), pluginYaml))
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

	out, err := os.ReadFile(filepath.Join(pluginFolder, Encode(path), pluginYaml))
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

func AddDevspaceVarsToPluginEnv(vars interface{}) {
	devspaceVarsOnce.Do(func() {
		if vars != nil {
			devspaceVar, isMapStringInterface := vars.(map[string]interface{})
			if isMapStringInterface {
				for key, value := range devspaceVar {
					// only map[string]string will be processed, map[string]Variable will be skipped
					vString, isString := value.(string)
					if isString {
						devspaceVars[key] = vString
					}
				}
			}
		}
	})
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
						return CallPluginExecutable(filepath.Join(pluginFolder, PluginBinary), newArgs, devspaceVars, os.Stdout)
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

func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("failed removing original file: %s", err)
	}
	return nil
}
