---
title: 1. Installation
---

You can either use one of our install scripts or simply download the released binary for your platform manually.

## Install Scripts
The following install scripts make it easy to install the DevSpace CLI. Simply copy the the commands into the recommended command line tool. The script will install the devspace CLI and add it to the PATH environment variable.

### For Windows
1. Open CMD with **admin rights**.
2. Run this install script:
```cmd
curl -s "https://raw.githubusercontent.com/covexo/devspace/master/scripts/installer-win.bat" >"%Temp%\install-devspace.bat"
"%Temp%\install-devspace.bat" "%PROGRAMFILES%\devspace"
del "%Temp%\install-devspace.bat"
```

**Note:** After running the install script, you should reopen the terminal window to refresh the environment variables.

### For Linux
1. Run the command:
```bash
curl --silent "https://api.github.com/repos/covexo/devspace/releases/latest" | sed -nE 's!.*"(https://github.com[^"]*devspace-linux-amd64)".*!\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace && sudo mv devspace /usr/local/bin
```

### For Mac
1. Run the command:
```bash
curl --silent "https://api.github.com/repos/covexo/devspace/releases/latest" | sed -nE 's!.*"(https://github.com[^"]*devspace-darwin-amd64)".*!\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace && sudo mv devspace /usr/local/bin
```

## Binary Download
An alternative to the install scripts is to:
1. download the latest release from the [GitHub releases page](https://github.com/covexo/devspace/releases)
2. add the binary folder path to your PATH environment variable **OR** run `PATH_TO_YOUR_RELEASE_FILE install` with admin/root priveleges
