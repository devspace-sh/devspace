package flags

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/loft-sh/devspace/pkg/util/terminal"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"os"

	flag "github.com/spf13/pflag"
)

var _, tty = terminal.SetupTTY(os.Stdin, os.Stdout)

// GlobalFlags is the flags that contains the global flags
type GlobalFlags struct {
	Silent bool
	NoWarn bool
	Debug  bool

	Namespace                string
	KubeContext              string
	Profiles                 []string
	ProfileRefresh           bool
	ProfileParents           []string
	DisableProfileActivation bool
	ConfigPath               string
	Vars                     []string

	RestoreVars    bool
	SaveVars       bool
	VarsSecretName string
	SwitchContext  bool

	InactivityTimeout int

	Flags *flag.FlagSet
}

// UseLastContext uses the last context
func (gf *GlobalFlags) UseLastContext(f factory.Factory, generatedConfig *generated.Config, log log.Logger) error {
	if gf.SwitchContext {
		if generatedConfig == nil || generatedConfig.GetActive().LastContext == nil {
			log.Warn("There is no last context to use. Only use the '--switch-context / -s' flag if you already have deployed the project before")
		} else {
			if tty.IsTerminalIn() {
				client, err := f.NewKubeDefaultClient()
				if err != nil {
					return errors.Wrap(err, "create kube client")
				}

				question := ""
				if generatedConfig.GetActive().LastContext.Context != "" && generatedConfig.GetActive().LastContext.Context != client.CurrentContext() {
					log.WriteString("\n")
					log.Warnf("Current kube context: '%s'", ansi.Color(client.CurrentContext(), "white+b"))
					log.Warnf("Last    kube context: '%s'", ansi.Color(generatedConfig.GetActive().LastContext.Context, "white+b"))
					question = "Do you want to use the previous kube context '" + generatedConfig.GetActive().LastContext.Context + "'?"
				} else if generatedConfig.GetActive().LastContext.Namespace != "" && generatedConfig.GetActive().LastContext.Namespace != client.Namespace() {
					log.WriteString("\n")
					log.Warnf("Current namespace: '%s'", ansi.Color(client.Namespace(), "white+b"))
					log.Warnf("Last    namespace: '%s'", ansi.Color(generatedConfig.GetActive().LastContext.Namespace, "white+b"))
					question = "Do you want to use the previous namespace '" + generatedConfig.GetActive().LastContext.Namespace + "'?"
				} else {
					return nil
				}

				answer, err := log.Question(&survey.QuestionOptions{
					Question:     question,
					DefaultValue: "yes",
					Options:      []string{"yes", "no"},
				})
				if err != nil {
					return err
				} else if answer == "no" {
					generatedConfig.GetActive().LastContext.Context = client.CurrentContext()
					generatedConfig.GetActive().LastContext.Namespace = client.Namespace()
					return nil
				}
			}

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
func (gf *GlobalFlags) ToConfigOptions(log log.Logger) *loader.ConfigOptions {
	if len(gf.ProfileParents) > 0 {
		log.Infof("--profile-parent is deprecated, please use --profile instead")
	}

	profiles := []string{}
	profiles = append(profiles, gf.ProfileParents...)
	profiles = append(profiles, gf.Profiles...)
	return &loader.ConfigOptions{
		Profiles:                 profiles,
		ProfileRefresh:           gf.ProfileRefresh,
		DisableProfileActivation: gf.DisableProfileActivation,
		KubeContext:              gf.KubeContext,
		Namespace:                gf.Namespace,
		Vars:                     gf.Vars,
		RestoreVars:              gf.RestoreVars,
		SaveVars:                 gf.SaveVars,
		VarsSecretName:           gf.VarsSecretName,
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
	flags.StringSliceVarP(&globalFlags.Profiles, "profile", "p", []string{}, "The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified")
	flags.StringSliceVar(&globalFlags.ProfileParents, "profile-parent", []string{}, "One or more profiles that should be applied before the specified profile (e.g. devspace dev --profile-parent=base1 --profile-parent=base2 --profile=my-profile)")
	flags.BoolVar(&globalFlags.ProfileRefresh, "profile-refresh", false, "If true will pull and re-download profile parent sources")
	flags.BoolVar(&globalFlags.DisableProfileActivation, "disable-profile-activation", false, "If true will ignore all profile activations")
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
