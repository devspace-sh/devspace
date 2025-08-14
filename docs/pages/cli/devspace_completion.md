---
title: "devspace completion --help"
sidebar_label: devspace completion
---


Outputs shell completion for the given shell (bash or zsh)

## Synopsis


	```
devspace completion SHELL [flags]
```

```
Outputs shell completion for the given shell (bash or zsh)

	This depends on the bash-completion binary.  Example installation instructions:
	OS X:
		$ brew install bash-completion
		$ source $(brew --prefix)/etc/bash_completion
		$ devspace completion bash > ~/.devspace-completion  # for bash users
		$ devspace completion fish > ~/.devspace-completion  # for fish users
		$ devspace completion zsh > ~/.devspace-completion   # for zsh users
		$ source ~/.devspace-completion
	Ubuntu:
		$ apt-get install bash-completion
		$ source /etc/bash-completion
		$ source <(devspace completion bash) # for bash users
		$ devspace completion fish | source # for fish users
		$ source <(devspace completion zsh)  # for zsh users

	Additionally, you may want to output the completion to a file and source in your .bashrc
```


## Flags

```
  -h, --help   help for completion
```


## Global & Inherited Flags

```
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true will ignore all profile activations
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems
      --kube-context string          The kubernetes context to use
      --kubeconfig string            The kubeconfig path to use
  -n, --namespace string             The kubernetes namespace to use
      --no-colors                    Do not show color highlighting in log output. This avoids invisible output with different terminal background colors
      --no-warn                      If true does not show any warning when deploying into a different namespace or kube-context than before
      --override-name string         If specified will override the DevSpace project name provided in the devspace.yaml
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

