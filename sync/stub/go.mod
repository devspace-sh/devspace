module github.com/devspace-cloud/devspace/sync/stub

go 1.13

require github.com/devspace-cloud/devspace v0.0.0-00010101000000-000000000000

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.0+incompatible
	github.com/Nvveen/Gotty => github.com/ijc25/Gotty v0.0.0-20170406111628-a8b993ba6abd

	github.com/agl/ed25519 => github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/devspace-cloud/devspace => ../..
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190812073006-9eafafc0a87e
)
