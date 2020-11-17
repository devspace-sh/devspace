package plugin

// PluginFolder is the folder where the plugins are stored
const PluginFolder = "plugins"

type Metadata struct {
	// Name is the name of the plugin
	Name string `json:"name"`

	// Version is a SemVer 2 version of the plugin.
	Version string `json:"version"`

	// Binaries holds the binary to download for the
	// given os & arch
	Binaries []Binary `json:"binaries,omitempty"`

	// Commands are the commands that will be added to devspace
	Commands []Command `json:"commands,omitempty"`

	// Vars are extra variables that can be used in the config
	Vars []Variable `json:"vars,omitempty"`

	// Hooks are commands that will be executed at specific events
	Hooks []Hook `json:"hooks,omitempty"`

	// This will be filled after parsing the metadata
	PluginFolder string `json:"pluginFolder,omitempty"`
}

type Hook struct {
	// Event is the name of the event when to execute this hook
	Event string `json:"event"`

	// Background specifies if the given command should be executed in the background
	Background bool `json:"background"`

	// BaseArgs that will be prepended to all supplied user flags for this plugin command
	BaseArgs []string `json:"baseArgs,omitempty"`
}

type Binary struct {
	// The current OS
	OS string `json:"os"`

	// The current Arch
	Arch string `json:"arch"`

	// The binary url to download from or relative path to use
	Path string `json:"path"`
}

type Command struct {
	// SubCommand is the sub command of devspace this command should be added to
	SubCommand string `json:"subCommand,omitempty"`

	// Name is the name of the command
	Name string `json:"name"`

	// Usage is the single-line usage text shown in help
	Usage string `json:"usage"`

	// Description is a long description shown
	Description string `json:"description"`

	// BaseArgs that will be prepended to all supplied user flags for this plugin command
	BaseArgs []string `json:"baseArgs,omitempty"`
}

type Variable struct {
	// Name is the name of the variable
	Name string `json:"name"`

	// BaseArgs that will be prepended to all supplied user flags for this plugin command
	BaseArgs []string `json:"baseArgs,omitempty"`
}
