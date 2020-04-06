module github.com/devspace-cloud/devspace

go 1.13

require (
	github.com/agl/ed25519 v0.0.0-00010101000000-000000000000 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.1.1
	github.com/docker/cli v0.0.0-20200130152716-5d0cf8839492
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/docker/go v1.5.1-1 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/spdystream v0.0.0-20160310174837-449fdfce4d96
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.3.2
	github.com/google/go-github v17.0.0+incompatible // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/go-version v1.0.0 // indirect
	github.com/haya14busa/goplay v1.0.0 // indirect
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/josharian/impl v0.0.0-20190715203526-f0d59e96e372 // indirect
	github.com/juju/errors v0.0.0-20180806074554-22422dad46e1
	github.com/juju/ratelimit v1.0.1
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
	github.com/krishicks/yaml-patch v0.0.10
	github.com/machinebox/graphql v0.2.2
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/otiai10/copy v0.0.0-20180813030456-0046ee23fdbd
	github.com/pkg/errors v0.9.1
	github.com/rhysd/go-github-selfupdate v0.0.0-20180520142321-41c1bbb0804a
	github.com/rjeczalik/notify v0.0.0-20181126183243-629144ba06a1
	github.com/sabhiram/go-gitignore v0.0.0-20180611051255-d3107576ba94
	github.com/shirou/gopsutil v0.0.0-20190627142359-4c8b404ee5c5
	github.com/sirupsen/logrus v1.4.2
	github.com/skratchdot/open-golang v0.0.0-20160302144031-75fb7ed4208c
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/tcnksm/go-gitconfig v0.1.2 // indirect
	github.com/theupdateframework/notary v0.6.1 // indirect
	github.com/toqueteos/trie v1.0.0 // indirect
	github.com/ulikunitz/xz v0.5.7 // indirect
	google.golang.org/grpc v1.27.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.7
	gopkg.in/src-d/enry.v1 v1.6.4
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/toqueteos/substring.v1 v1.0.2 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.1.2
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/cli-runtime v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.17.3
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
	mvdan.cc/sh/v3 v3.0.0-alpha2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.0+incompatible
	github.com/Nvveen/Gotty => github.com/ijc25/Gotty v0.0.0-20170406111628-a8b993ba6abd

	github.com/agl/ed25519 => github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v1.4.2-0.20200203170920-46ec8731fbce

	golang.org/x/sys => golang.org/x/sys v0.0.0-20191128015809-6d18c012aee9
)
