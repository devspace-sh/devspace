package flags

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	flag "github.com/spf13/pflag"
)

// GlobalFlags is the flags that contains the global flags
type GlobalFlags struct {
	Silent                   bool
	NoWarn                   bool
	NoColors                 bool
	Debug                    bool
	DisableProfileActivation bool
	SwitchContext            bool
	InactivityTimeout        int
	KubeConfig               string
	OverrideName             string
	Namespace                string
	KubeContext              string
	ConfigPath               string
	Profiles                 []string
	Vars                     []string

	Flags *flag.FlagSet
}

// ToConfigOptions converts the globalFlags into config options
func (gf *GlobalFlags) ToConfigOptions() *loader.ConfigOptions {
	profiles := []string{}
	profiles = append(profiles, gf.Profiles...)
	return &loader.ConfigOptions{
		OverrideName:             gf.OverrideName,
		Profiles:                 profiles,
		DisableProfileActivation: gf.DisableProfileActivation,
		Vars:                     gf.Vars,
	}
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{
		Vars:  []string{},
		Flags: flags,
	}

	flags.StringVar(&globalFlags.OverrideName, "override-name", "", "If specified will override the DevSpace project name provided in the devspace.yaml")
	flags.BoolVar(&globalFlags.NoWarn, "no-warn", false, "If true does not show any warning when deploying into a different namespace or kube-context than before")
	flags.BoolVar(&globalFlags.NoColors, "no-colors", false, "Do not show color highlighting in log output. This avoids invisible output with different terminal background colors")
	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	flags.BoolVar(&globalFlags.Silent, "silent", false, "Run in silent mode and prevents any devspace log output except panics & fatals")

	flags.StringSliceVarP(&globalFlags.Profiles, "profile", "p", []string{}, "The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified")
	flags.BoolVar(&globalFlags.DisableProfileActivation, "disable-profile-activation", false, "If true will ignore all profile activations")
	flags.BoolVarP(&globalFlags.SwitchContext, "switch-context", "s", false, "Switches and uses the last kube context and namespace that was used to deploy the DevSpace project")
	flags.StringVarP(&globalFlags.Namespace, "namespace", "n", "", "The kubernetes namespace to use")
	flags.StringVar(&globalFlags.KubeContext, "kube-context", "", "The kubernetes context to use")
	flags.StringSliceVar(&globalFlags.Vars, "var", []string{}, "Variables to override during execution (e.g. --var=MYVAR=MYVALUE)")
	flags.StringVar(&globalFlags.KubeConfig, "kubeconfig", "", "The kubeconfig path to use")

	flags.IntVar(&globalFlags.InactivityTimeout, "inactivity-timeout", 0, "Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems")
	flags.AddFlag(&flag.Flag{
		Name:   "config",
		Usage:  "DEPRECATED: please use the DEVSPACE_CONFIG environment variable instead",
		Hidden: true,
		Value:  NewStringValue("", &globalFlags.ConfigPath),
	})
	return globalFlags
}

type StringValue string

func NewStringValue(val string, p *string) *StringValue {
	*p = val
	return (*StringValue)(p)
}

func (s *StringValue) Set(val string) error {
	*s = StringValue(val)
	return nil
}
func (s *StringValue) Type() string {
	return "string"
}

func (s *StringValue) String() string { return string(*s) }
