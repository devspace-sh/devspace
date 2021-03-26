package flags

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"

	flag "github.com/spf13/pflag"
)

// GlobalFlags is the flags that contains the global flags
type GlobalFlags struct {
	Silent bool
	NoWarn bool
	Debug  bool

	Namespace      string
	KubeContext    string
	Profile        string
	ProfileRefresh bool
	ProfileParents []string
	ConfigPath     string
	Vars           []string

	RestoreVars    bool
	SaveVars       bool
	VarsSecretName string
	SwitchContext  bool

	InactivityTimeout int

	Flags *flag.FlagSet
}

// UseLastContext uses the last context
func (gf *GlobalFlags) UseLastContext(generatedConfig *generated.Config, log log.Logger) error {
	if gf.KubeContext == "" && gf.Namespace == "" && gf.SwitchContext == true {
		if generatedConfig == nil || generatedConfig.GetActive().LastContext == nil {
			log.Warn("There is no last context to use. Only use the '--switch-context / -s' flag if you already have deployed the project before")
		} else {
			gf.KubeContext = generatedConfig.GetActive().LastContext.Context
			gf.Namespace = generatedConfig.GetActive().LastContext.Namespace

			log.Infof("Switching to context '%s' and namespace '%s'", ansi.Color(gf.KubeContext, "white+b"), ansi.Color(gf.Namespace, "white+b"))
			return nil
		}
	}

	gf.SwitchContext = false
	return nil
}

// ToConfigOptions converts the globalFlags into config options
func (gf *GlobalFlags) ToConfigOptions() *loader.ConfigOptions {
	return &loader.ConfigOptions{
		Profile:        gf.Profile,
		ProfileRefresh: gf.ProfileRefresh,
		ProfileParents: gf.ProfileParents,
		KubeContext:    gf.KubeContext,
		Namespace:      gf.Namespace,
		Vars:           gf.Vars,
		RestoreVars:    gf.RestoreVars,
		SaveVars:       gf.SaveVars,
		VarsSecretName: gf.VarsSecretName,
	}
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{
		Vars:  []string{},
		Flags: flags,
	}

	flags.BoolVar(&globalFlags.NoWarn, "no-warn", false, "If true does not show any warning when deploying into a different namespace or kube-context than before")
	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	flags.BoolVar(&globalFlags.Silent, "silent", false, "Run in silent mode and prevents any devspace log output except panics & fatals")

	flags.StringVar(&globalFlags.ConfigPath, "config", "", "The devspace config file to use")
	flags.StringVarP(&globalFlags.Profile, "profile", "p", "", "The devspace profile to use (if there is any)")
	flags.StringSliceVar(&globalFlags.ProfileParents, "profile-parent", []string{}, "One or more profiles that should be applied before the specified profile (e.g. devspace dev --profile-parent=base1 --profile-parent=base2 --profile=my-profile)")
	flags.BoolVar(&globalFlags.ProfileRefresh, "profile-refresh", false, "If true will pull and re-download profile parent sources")
	flags.StringVarP(&globalFlags.Namespace, "namespace", "n", "", "The kubernetes namespace to use")
	flags.StringVar(&globalFlags.KubeContext, "kube-context", "", "The kubernetes context to use")
	flags.BoolVarP(&globalFlags.SwitchContext, "switch-context", "s", false, "Switches and uses the last kube context and namespace that was used to deploy the DevSpace project")
	flags.StringSliceVar(&globalFlags.Vars, "var", []string{}, "Variables to override during execution (e.g. --var=MYVAR=MYVALUE)")

	flags.BoolVar(&globalFlags.RestoreVars, "restore-vars", false, "If true will restore the variables from kubernetes before loading the config")
	flags.BoolVar(&globalFlags.SaveVars, "save-vars", false, "If true will save the variables to kubernetes after loading the config")
	flags.StringVar(&globalFlags.VarsSecretName, "vars-secret", "devspace-vars", "The secret to restore/save the variables from/to, if --restore-vars or --save-vars is enabled")
	flags.IntVar(&globalFlags.InactivityTimeout, "inactivity-timeout", 180, "Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems")

	return globalFlags
}
