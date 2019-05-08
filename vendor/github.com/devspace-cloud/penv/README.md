# penv
`penv` permanently sets environment variables. It supports the following:

* bash - entries are added to `~/.bashrc`
* fish - entries are added to `~/.config/fish/config.fish`
* windows - entries are added to the registry for the current user
* osx - entries are added to a user launchctl script. You will have to restart
        programs to pick up the new environment. (ie restart your terminal)

## Installation
`penv` is both a library and a command. To use the library in your own code see
[the documentation](https://godoc.org/github.com/badgerodon/penv).
To install the command run:

    go get github.com/badgerodon/penv/...

Here's its usage:

    penv <command>

    Commands:
      set <name> <value>
        Permanently NAME to VALUE in the environment

      unset <name>
        Permanently unset NAME in the environment

      append <name> <value>
        Permanently append VALUE to NAME in the environment

## Gotchas
Windows requires at least Go 1.3.

Different operating systems / shells aren't really compatible. I'm able to discern which environment variables I'm responsible for with shells (like bash) by using their config files, but I can't do that with Windows. All appends will get collapsed into sets, and unsets aren't just masking the value, they may actually remove it.

In other words this command works but it's dangerous. If you set your `PATH` don't be surprised when it clears all the previous values and you can't get them back.

## License
MIT
