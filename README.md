![DevSpace Workflow](docs/website/static/img/header-readme.svg)

# DevSpace - Cloud-Native Development with Kubernetes
[![Build Status](https://travis-ci.org/covexo/devspace.svg?branch=master)](https://travis-ci.org/covexo/devspace)
[![Go Report Card](https://goreportcard.com/badge/github.com/covexo/devspace)](https://goreportcard.com/report/github.com/covexo/devspace)
[![Join the community on Spectrum Chart](https://withspectrum.github.io/badge/badge.svg)](https://spectrum.chat/devspace)

With a DevSpace, you can build, test and run **code directly inside any Kubernetes cluster**. You can run `devspace up` in any of your projects and the client-only DevSpace CLI will start a DevSpace within your Kubernetes cluster. Keep coding as usual and the DevSpace CLI will sync any code change directly into the containers of your DevSpace. 

**No more waiting** for re-building images, re-deploying containers and restarting applications on every source code change. Simply edit your code with any IDE and run your code instantly inside your DevSpace.

## Why use a DevSpace?
Program inside any Kubernetes cluster (e.g. minikube, self-hosted or cloud platform) and:
- iterate quickly: no more building and pushing images on every change, use **hot reloading** instead (e.g. with nodemon)
- keep your existing workflow and tools: **the DevSpace CLI works with every IDE** (no plugins required)
- access cluster-internal services and data during development
- debug efficiently with port forwarding and terminal proxying
- migrate to Docker & Kubernetes within minutes

## Demo
This demo shows how to run `devspace up` directly from the terminal inside Visual Studio Code. However, the DevSpace CLI is not a plugin and will work with any terminal. In this example, we are starting a DevSpace for a React application.

![DevSpace CLI Demo](docs/website/static/img/devspace-cli-demo-readme.gif)

## [Installation](https://devspace.covexo.com/docs/getting-started/installation.html)
These commands will install the devspace CLI and add it to the PATH environment variable. For more details, see: [Install Guide](https://devspace.covexo.com/docs/getting-started/installation.html).

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
```bash
curl --silent "https://api.github.com/repos/covexo/devspace/releases/latest" | sed -nr 's!.*"(https://github.com[^"]*devspace-linux-amd64)".*!\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace && sudo mv devspace /usr/local/bin
```

### For Mac
```bash
curl --silent "https://api.github.com/repos/covexo/devspace/releases/latest" | sed -nr 's!.*"(https://github.com[^"]*devspace-darwin-amd64)".*!\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace && sudo mv devspace /usr/local/bin
```

## [Quickstart](https://devspace.covexo.com/docs/getting-started/quickstart.html)
The DevSpace CLI allows you to create a DevSpace for any existing project with just a single command:
```
devspace up
```
Take a look at the [Getting Started Guide](https://devspace.covexo.com/docs/getting-started/quickstart.html) on our documentation page to start coding with a DevSpace.

**Note:** Don't worry, with you can use `devspace reset` to reset your project and go back to local development.

## [Documentation](https://devspace.covexo.com/docs/getting-started/quickstart.html)
Here you can find some links to the most important pages of our documentation:
- [Getting Started Guide](https://devspace.covexo.com/docs/getting-started/quickstart.html)
- [Frequently Asked Questions (FAQ)](https://devspace.covexo.com/docs/getting-started/faq.html)
- [CLI Documentation](https://devspace.covexo.com/docs/cli/init.html)
- [Configuration Options](https://devspace.covexo.com/docs/configuration/dockerfile.html)
- [Architecture Documentation](https://devspace.covexo.com/docs/advanced/architecture.html)

## [DevSpace Cloud](https://devspace-cloud.com/)
The DevSpace Cloud provides hosted DevSpaces. The service is currently in private beta. If you would like to join the beta program, you can [**request access to the DevSpace Cloud](https://devspace-cloud.com/). As a thank you for testing the DevSpace cloud, members of the beta program will receive a special **forever free subcription** to the DevSpace Cloud.

## [Architecture](https://devspace.covexo.com/docs/advanced/architecture.html).
Architecturally, the DevSpace CLI is a client-side software that interacts with services within your Kubernetes cluster. While the DevSpace CLI can deploy required services (e.g. image registry, Tiller server, Kaniko build pods) automatically, you can also configure it to use already deployed or externally hosted services.

![DevSpace CLI Architecture](docs/website/static/img/devspace-architecture.svg)

For a more detailed description of the internals of the DevSpace CLI, take a look at the [Architecture Documentation](https://devspace.covexo.com/docs/advanced/architecture.html).

**Note:** Any interaction between your local computer and your DevSpace is passed through your Kubernetes API server, so you should ensure that your API server is protected with a suitable configuration for using TLS.

## [Contributing](CONTRIBUTING.md)
As any open source projects, we are looking forward to your contributions.

### Reporting Issues
If you find a bug while working with the DevSpace CLI, please [open an issue on GitHub](https://github.com/covexo/devspace/issues/new?labels=kind%2Fbug&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

### Feedback & Feature Requests
You are more than welcome to open issues in this project to:
- [give feedback](https://github.com/covexo/devspace/issues/new?labels=kind%2Ffeedback&title=Feedback:)
- [suggest new features](https://github.com/covexo/devspace/issues/new?labels=kind%2Ffeature&title=Feature%20Request:)
- [ask a question](https://github.com/covexo/devspace/issues/new?labels=kind%2Fquestion&title=Question:)

### Contributing Code
This project is mainly written in Golang. To contribute code,
1. Check-out the project: `git clone https://github.com/covexo/devspace && cd devspace`
2. Install the dependencies: `dep ensure -v` (requires [Installing Dep](https://golang.github.io/dep/docs/installation.html))
3. Make changes to the code (add new dependencies to the Gopkg.toml)
4. Build the project, e.g. via `go build -o devspace.exe`

## License
You can use the DevSpace CLI for any private or commercial projects because it is licensed unter the Apache 2.0 open source license.
