---
title: 1. Install DevSpace
---

DevSpace is a client-only command-line tool that runs as a single binary directly on your computer. DevSpace does not require any other programs or dependencies to be installed.

## Install DevSpace
Install DevSpace either via NPM (recommended for Windows) or using any of the platform-specific scripts below:

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
Invoke-WebRequest -UseBasicParsing ((Invoke-WebRequest -URI "https://github.com/devspace-cloud/devspace/releases/latest" -UseBasicParsing).Content -replace "(?ms).*`"([^`"]*devspace-windows-amd64.exe)`".*","https://github.com/`$1") -o $Env:APPDATA\devspace\devspace.exe;
$env:Path += ";" + $Env:APPDATA + "\devspace";
[Environment]::SetEnvironmentVariable("Path", $env:Path, [System.EnvironmentVariableTarget]::User);
```
<!--END_DOCUSAURUS_CODE_TABS-->

Alternatively, you can simply download the binary for your platform from the [GitHub Releases](https://github.com/devspace-cloud/devspace/releases) page and add this binary to your PATH.

<details>
<summary>
### How to upgrade DevSpace?
</summary>

Upgrading DevSpace is as easy as running:
```bash
devspace upgrade
```

</details>

<details>
<summary>
### How to uninstall DevSpace?
</summary>

Uninstalling DevSpace is as easy as removing the DevSpace binary from your machine. You can use the following commands for removing the binary and optionally also deleting the global DevSpace config folder in your home directory:
<!--DOCUSAURUS_CODE_TABS-->
<!--NPM-->
```bash
npm uninstall -g devspace
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
