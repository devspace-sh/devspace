# DevSpace - Kubernetes for developers
[![Build Status](https://travis-ci.org/covexo/devspace.svg?branch=master)](https://travis-ci.org/devspace-cloud/devspace)
[![Go Report Card](https://goreportcard.com/badge/github.com/covexo/devspace)](https://goreportcard.com/report/github.com/devspace-cloud/devspace)
[![Slack](http://devspace.cloud/slack/badge.svg)](http://devspace.cloud/slack)
[![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/home?status=Just%20found%20out%20about%20%23DevSpace.cli%3A%20https%3A//github.com/devspace-cloud/devspace%0A%0AIt%20lets%20you%20build%20cloud%20native%20software%20directly%20on%20top%20of%20%23Kubernetes%20and%20%23Docker%0A%23CloudNative%20%23k8s)

DevSpace allows any developer to build and deploy truly scalable, cloud-native software on top of Kubernetes.

## Why use DevSpace?
DevSpace is pure Kubernetes but optimized for developer experience
- iterate quickly: no more building and pushing images on every change, use **hot reloading** instead (e.g. with nodemon)
- keep your existing workflow and tools: **the DevSpace works with every IDE** (no plugins required)
- access cluster-internal services and data during development
- debug efficiently with port forwarding and terminal proxying
- migrate to Docker & Kubernetes within minutes

## Demo

![DevSpace CLI Demo](docs/website/static/img/devspace-cli-demo-readme.gif)

## [Install & Quickstart Guide](https://devspace.cloud/docs/getting-started/installation)
Follow this link for the [Install & Quickstart Guide](https://devspace.cloud/docs/getting-started/installation).

After installing the DevSpace CLI, you can any project to a Space with these commands:
```
devspace init
devspace create space production
devspace deploy production
```
Take a look at the [Install & Getting Started Guide](https://devspace.cloud/docs/getting-started/installation) to see how to get started with a DevSpace.

**Note:** Don't worry, you can use `devspace reset` to reset your project.

## [Documentation](https://devspace.cloud/docs/getting-started/installation)
Here you can find some links to the most important pages of our documentation:
- [Getting Started Guide](https://devspace.cloud/docs/getting-started/installation)
- [Deployment with DevSpace](https://devspace.cloud/docs/deployment/workflow)
- [Development with DevSpace](https://devspace.cloud/docs/development/workflow)
- [Configuration Reference](https://devspace.cloud/docs/configuration/reference)
- [CLI Documentation](https://devspace.cloud/docs/cli/overview)

## [Contributing](CONTRIBUTING.md)
As any open source projects, we are looking forward to your contributions.

### Reporting Issues
If you find a bug while working with the DevSpace CLI, please [open an issue on GitHub](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

### Feedback & Feature Requests
You are more than welcome to open issues in this project to:
- [give feedback](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Ffeedback&title=Feedback:)
- [suggest new features](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Ffeature&template=feature-request.md&title=Feature%20Request:)
- [ask a question](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Fquestion&title=Question:)

### Contributing Code
This project is mainly written in Golang. To contribute code,
1. Ensure you are running golang version 1.11.4 or greater for go module support.
2. Check-out the project: `git clone https://github.com/devspace-cloud/devspace && cd devspace`
3. Make changes to the code (dependencies are downloaded when you run any go command such as `go build`)
4. Build the project, e.g. via `go build -o devspace.exe`

## License
You can use the DevSpace.cli for any private or commercial projects because it is licensed under the Apache 2.0 open source license.
