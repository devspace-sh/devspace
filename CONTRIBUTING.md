# Contribution Guidelines
Please read this guide if you plan to contribute to the DevSpace CLI. We welcome any kind of contribution. No matter if you are an experienced programmer or just starting, we are looking forward to your contribution.

## Reporting Issues
If you find a bug while working with the DevSpace CLI, please [open an issue on GitHub](https://github.com/covexo/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

## Feature Requests
You are more than welcome to open issues in this project to [suggest new features](https://github.com/covexo/devspace/issues/new?labels=kind%2Ffeature&template=feature-request.md&title=Feature%20Request:).

## Contributing Code
This project is mainly written in Golang. To contribute code,
1. Fork the project
2. Clone the project: `git clone https://github.com/[YOUR_USERNAME]/devspace && cd devspace`
3. Install the dependencies: `dep ensure -v` (requires [Installing Dep](https://golang.github.io/dep/docs/installation.html))
4. Make changes to the code (add new dependencies to the Gopkg.toml)
5. Build the project, e.g. via `go build -o devspace.exe`
6. Make changes
7. Run tests: `go test ./...`
8. Format your code: `go fmt ./...`
9. Commit changes
10. Push commits
11. Open pull request

## Improving the Documentation
The documentation is contained within `./docs` and made with Docusaurus. See the [Docs README](./docs) for infos about developing the docs.
