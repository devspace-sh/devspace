# Contribution Guidelines
Please read this guide if you plan to contribute to the DevSpace CLI. We welcome any kind of contribution. No matter if you are an experienced programmer or just starting, we are looking forward to your contribution.

## Reporting Issues
If you find a bug while working with the DevSpace CLI, please [open an issue on GitHub](https://github.com/loft-sh/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

## Feature Requests
You are more than welcome to open issues in this project to [suggest new features](https://github.com/loft-sh/devspace/issues/new?labels=kind%2Ffeature&template=feature-request.md&title=Feature%20Request:).

## Contributing Code
This project is mainly written in Golang. To contribute code,
1. Ensure you are running golang version 1.11.4 or greater for go module support
2. Set the following environment variables:
    ```
    GO111MODULE=on
    GOFLAGS=-mod=vendor
    ```
3. Fork the project
4. Clone the project: `git clone https://github.com/[YOUR_USERNAME]/devspace && cd devspace`
5. Run `go clean -modcache`
6. Run `go mod vendor` to update the dependencies
7. Build the project, e.g. via `go build -o devspace.exe`
8. Build devspacehelper using below command
   ```
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-extldflags=-static" -o ~/.devspace/devspacehelper/latest/devspacehelper helper/main.go
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-extldflags=-static" -o ~/.devspace/devspacehelper/latest/devspacehelper-arm64 helper/main.go
   chmod +x ~/.devspace/devspacehelper/latest/devspacehelper
   chmod +x ~/.devspace/devspacehelper/latest/devspacehelper-arm64
   ```
   
   The above command is required to be executed as sometimes you may observe below error,
   ```
   start_dev: error setting up proxy commands in container:   Internal error occurred: error executing command in container: failed to exec in container: failed to start exec "38d5fc79b8a7c63d38ba5f99237d80df186871fa4b43987a83a926628d1c47e1": OCI runtime exec failed: exec failed: unable to start container process: exec /tmp/devspacehelper: text file busy: unknown
   ```


9. Make changes
10. Run unit tests: `./hack/coverage.bash`
11. Run E2E tests: `cd e2e/ && go test -v -ginkgo.v`
12. Format your code: `go fmt ./...`
13. Commit changes *([Please refer the commit message conventions](https://www.conventionalcommits.org/en/v1.0.0/))*
14. Push commits
15. Open pull request

## Improving the Documentation
The documentation is contained within `./docs` and made with Docusaurus. See the [Docs README](./docs) for infos about developing the docs.
