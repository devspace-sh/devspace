package variable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/context/values"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/utils/pkg/downloader"
	"github.com/loft-sh/utils/pkg/downloader/commands"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/mitchellh/go-homedir"
)

// PredefinedVariableOptions holds the options for a predefined variable to load
type PredefinedVariableOptions struct {
	ConfigPath string
	KubeClient kubectl.Client
	Profile    []string
}

// PredefinedVariableFunction is the definition of a predefined variable
type PredefinedVariableFunction func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error)

// predefinedVars holds all predefined variables that can be used in the config
var predefinedVars = map[string]PredefinedVariableFunction{
	"DEVSPACE_NAME": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		name, ok := values.NameFrom(ctx)
		if !ok {
			return "", nil
		}
		return name, nil
	},
	"DEVSPACE_EXECUTABLE": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		ex, err := os.Executable()
		if err != nil {
			return nil, err
		}
		return ex, nil
	},
	"DEVSPACE_KUBECTL_EXECUTABLE": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		debugLog := log.WithLevel(logrus.DebugLevel)
		path, err := downloader.NewDownloader(commands.NewKubectlCommand(), debugLog, constants.DefaultHomeDevSpaceFolder).EnsureCommand(ctx)
		if err != nil {
			debugLog.Debugf("Error downloading kubectl: %v", err)
			return "", nil
		}
		return path, nil
	},
	"DEVSPACE_TMPDIR": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		tempFolder, ok := values.TempFolderFrom(ctx)
		if !ok {
			return os.TempDir(), nil
		}
		return tempFolder, nil
	},
	"DEVSPACE_VERSION": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		return upgrade.GetVersion(), nil
	},
	"DEVSPACE_RANDOM": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		return randutil.GenerateRandomString(6), nil
	},
	"DEVSPACE_PROFILE": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		return GetLastProfile(options.Profile), nil
	},
	"DEVSPACE_PROFILES": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		return strings.Join(options.Profile, " "), nil
	},
	"DEVSPACE_USER_HOME": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		homeDir, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		return homeDir, nil
	},
	"DEVSPACE_TIMESTAMP": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	},
	"DEVSPACE_GIT_BRANCH": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		configPath := options.ConfigPath
		branch, err := git.GetBranch(filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("error retrieving git branch: %v, but predefined var devspace.git.branch is used", err)
		}

		return branch, nil
	},
	"DEVSPACE_GIT_COMMIT": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		configPath := options.ConfigPath
		hash, err := git.GetHash(ctx, filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("no git repository found (%v), but predefined var devspace.git.commit is used", err)
		}

		return hash[:8], nil
	},
	"DEVSPACE_CONTEXT": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		if options.KubeClient == nil {
			return "", nil
		}

		return options.KubeClient.CurrentContext(), nil
	},
	"DEVSPACE_NAMESPACE": func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
		if options.KubeClient == nil {
			return "", nil
		}

		return options.KubeClient.Namespace(), nil
	},
}

func init() {
	// migrate old names
	predefinedVars["devspace.name"] = predefinedVars["DEVSPACE_NAME"]
	predefinedVars["devspace.version"] = predefinedVars["DEVSPACE_VERSION"]
	predefinedVars["devspace.random"] = predefinedVars["DEVSPACE_RANDOM"]
	predefinedVars["devspace.profile"] = predefinedVars["DEVSPACE_PROFILE"]
	predefinedVars["devspace.userHome"] = predefinedVars["DEVSPACE_USER_HOME"]
	predefinedVars["devspace.timestamp"] = predefinedVars["DEVSPACE_TIMESTAMP"]
	predefinedVars["devspace.git.branch"] = predefinedVars["DEVSPACE_GIT_BRANCH"]
	predefinedVars["devspace.git.commit"] = predefinedVars["DEVSPACE_GIT_COMMIT"]
	predefinedVars["devspace.context"] = predefinedVars["DEVSPACE_CONTEXT"]
	predefinedVars["devspace.namespace"] = predefinedVars["DEVSPACE_NAMESPACE"]

	// support migrating DevSpace Cloud users
	predefinedVars["DEVSPACE_SPACE"] = predefinedVars["DEVSPACE_NAMESPACE"]
}

func IsPredefinedVariable(name string) bool {
	_, ok := predefinedVars[name]
	return ok
}

func AddPredefinedVars(plugins []plugin.Metadata) {
	for _, p := range plugins {
		pluginName := p.Name
		pluginFolder := p.PluginFolder
		for _, variable := range p.Vars {
			v := variable
			predefinedVars[variable.Name] = func(ctx context.Context, options *PredefinedVariableOptions, log log.Logger) (interface{}, error) {
				args, err := json.Marshal(os.Args)
				if err != nil {
					return "", err
				}

				buffer := &bytes.Buffer{}
				err = plugin.CallPluginExecutable(filepath.Join(pluginFolder, plugin.PluginBinary), v.BaseArgs, map[string]string{
					plugin.OsArgsEnv: string(args),
				}, buffer)
				if err != nil {
					return "", fmt.Errorf("executing plugin %s: %s - %v", pluginName, buffer.String(), err)
				}

				return strings.TrimSpace(buffer.String()), nil
			}
		}
	}
}

// NewPredefinedVariable creates a new predefined variable for the given name or fails if there
// is none with the given name
func NewPredefinedVariable(name string, options *PredefinedVariableOptions, log log.Logger) (Variable, error) {
	if _, ok := predefinedVars[name]; !ok {
		return nil, errors.New("predefined variable " + name + " not found")
	}

	return &predefinedVariable{
		name:    name,
		options: options,
		log:     log,
	}, nil
}

type predefinedVariable struct {
	name    string
	options *PredefinedVariableOptions
	log     log.Logger
}

func (p *predefinedVariable) Load(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	getVar, ok := predefinedVars[p.name]
	if !ok {
		return nil, errors.New("predefined variable " + p.name + " not found")
	}

	return getVar(ctx, p.options, p.log)
}

func GetLastProfile(profiles []string) string {
	if len(profiles) == 0 {
		return ""
	}
	return profiles[len(profiles)-1]
}
