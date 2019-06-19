---
title: 1. Install DevSpace CLI
---

To build and deploy applications with DevSpace, you need to install DevSpace CLI.

## Install DevSpace CLI
Install DevSpace CLI with NPM (recommended for Windows users) or any of the platform-specific installation scripts shown below.

<!--DOCUSAURUS_CODE_TABS-->
<!--NPM-->
```bash
npm install -g devspace
```

<!--Mac Terminal-->
```bash
curl -s -L "https://github.com/devspace-cloud/devspace/releases/latest" | sed -nE 's!.*"([^"]*devspace-darwin-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace;
sudo mv devspace /usr/local/bin;
```

<!--Linux Bash-->
```bash
curl -s -L "https://github.com/devspace-cloud/devspace/releases/latest" | sed -nE 's!.*"([^"]*devspace-linux-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace;
sudo mv devspace /usr/local/bin;
```

<!--Windows Powershell-->
```powershell
md -Force "$Env:APPDATA\devspace"; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]'Tls,Tls11,Tls12';
wget -UseBasicParsing ((Invoke-WebRequest -URI "https://github.com/devspace-cloud/devspace/releases/latest" -UseBasicParsing).Content -replace "(?ms).*`"([^`"]*devspace-windows-amd64.exe)`".*","https://github.com/`$1") -o $Env:APPDATA\devspace\devspace.exe; & "$Env:APPDATA\devspace\devspace.exe" "install"; $env:Path = (Get-ItemProperty -Path HKCU:\Environment -Name Path).Path
```
<!--END_DOCUSAURUS_CODE_TABS-->

Alternatively, you can simply download the binary for your platform from the [GitHub Releases](https://github.com/devspace-cloud/devspace/releases) page and add the binary to your PATH.

<details>
<summary>
### How to uninstall DevSpace CLI?
</summary>

Uninstalling DevSpace CLI is as easy as removing the devspace binary from your machine. You can use the following commands for removing the binary and optionally also deleting the DevSpace folder in your home directory:
<!--DOCUSAURUS_CODE_TABS-->
<!--NPM-->
```bash
npm uninstall -g devspace

# If you also want to delete the DevSpace configuration folder:
rm "~/.devspace";           # for Mac and Linux
Remove-Item "~\.devspace";  # for Windows
```

<!--Mac Terminal-->
```bash
sudo rm  /usr/local/bin/devspace;

# If you also want to delete the DevSpace configuration folder:
rm "~/.devspace";
```

<!--Linux Bash-->
```bash
sudo rm /usr/local/bin/devspace;

# If you also want to delete the DevSpace configuration folder:
rm "~/.devspace";
```

<!--Windows Powershell-->
```powershell
Remove-Item "$Env:APPDATA\devspace";

# If you also want to delete the DevSpace configuration folder:
Remove-Item "~\.devspace";
```
<!--END_DOCUSAURUS_CODE_TABS-->


</details>

## Install Docker (optional)
The preferred image building method is Docker, however DevSpace CLI is also able to build images directly inside Kubernetes pods (using [kaniko](https://github.com/GoogleContainerTools/kaniko)) if you don't have Docker installed. If you want to install Docker, you can download the latest stable releases here:
- **Mac**: [Docker Community Edition](https://download.docker.com/mac/stable/Docker.dmg)
- **Windows Pro**: [Docker Community Edition](https://download.docker.com/win/stable/Docker%20for%20Windows%20Installer.exe)
- **Windows 10 Home**: [Docker Toolbox](https://download.docker.com/win/stable/DockerToolbox.exe) (legacy)
