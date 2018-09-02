![DevSpace Workflow](docs/website/static/img/header-readme.svg)

# DevSpace CLI - Cloud-Native Development with Kubernetes
[![Build Status](https://travis-ci.org/covexo/devspace.svg?branch=master)](https://travis-ci.org/covexo/devspace)
[![Go Report Card](https://goreportcard.com/badge/github.com/covexo/devspace)](https://goreportcard.com/report/github.com/covexo/devspace)

With the DevSpace CLI, developers can build cloud-native applications directly inside a Kubernetes cluster. It works with any self-hosted Kubernetes cluster (e.g. minikube or baremetal) as well as with managed Kubernetes cluster provided by cloud platforms, e.g. Google Kubernetes Engine.

## Why using a DevSpace?
Your DevSpace lets you build, test and run code directly inside a Kubernetes cluster and:
- Allows you to access cluster-internal services and data with ease
- Works perfectly with your favorite **hot reloading** tools (e.g. nodemon)
- Lets you iterate quickly: no more re-building and pushing images on every change
- Integrates into your existing workflow: **code with your favorite IDE** and desktop tools
- Allows you to build, test and run code directly inside Kubernetes (via **real-time code synchronization**)
- Supports efficient debugging through port forwarding and terminal proxying
- Provides **automatic image building** without the need to install Docker
- Lets you migrate to Docker & Kubernetes within minutes
- Works with any Kubernetes cluster (e.g. minikube, self-hosted or cloud platform)

## Quickstart
The DevSpace CLI allows you to create a DevSpace for any existing project with just a single command:
```
devspace up
```
Take a look at the [Getting Started Guide](https://devspace.covexo.com/docs/getting-started/quickstart.html) on our documentation page to start coding with a DevSpace.

**Note:** Don't worry, with the cleanup command `devspace reset`, you can easily reset your project and go back to local development.

## Demo
coming soon

## Documentation
Here you can find some links to the most important pages of our documentation:
- [Getting Started Guide](https://devspace.covexo.com/docs/getting-started/quickstart.html)
- [CLI Documentation](https://devspace.covexo.com/docs/cli/init.html)
- [Configuration Specification](https://devspace.covexo.com/docs/configuration/dockerfile.html)
- [Architecture Documentation](https://devspace.covexo.com/docs/advanced/architecture.html)

## Contributing
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

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fcovexo%2Fdevspace.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fcovexo%2Fdevspace?ref=badge_large)
