package localregistry

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	kubectltesting "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"gotest.tools/assert"
)

type useLocalRegistryTestCase struct {
	name        string
	client      kubectl.Client
	config      *latest.Config
	imageConfig *latest.Image
	skipPush    bool
	expected    bool
}

func TestUseLocalRegistry(t *testing.T) {
	testCases := []useLocalRegistryTestCase{
		{
			name: "KinD Cluster",
			client: &kubectltesting.Client{
				Context: "kind-kind",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "KinD Cluster Local Registry Enabled",
			client: &kubectltesting.Client{
				Context: "kind-kind",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(true),
				},
			},
			expected: true,
		},
		{
			name: "KinD Cluster Local Registry Enabled skip push",
			client: &kubectltesting.Client{
				Context: "kind-kind",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(true),
				},
			},
			skipPush: true,
			expected: false,
		},
		{
			name: "KinD Cluster Local Registry Fallback",
			client: &kubectltesting.Client{
				Context: "kind-kind",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: nil,
				},
			},
			expected: false,
		},
		{
			name: "VCluster with KinD Cluster",
			client: &kubectltesting.Client{
				Context: "vcluster_devspace-kind_vcluster-devspace-kind_kind-kind",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Loft VCluster with KinD Cluster",
			client: &kubectltesting.Client{
				Context: "loft-vcluster_devspace-kind_vcluster-devspace-kind_kind-kind",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Docker Desktop Cluster",
			client: &kubectltesting.Client{
				Context: "docker-desktop",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Docker Desktop Cluster Local Registry Enabled",
			client: &kubectltesting.Client{
				Context: "docker-desktop",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(true),
				},
			},
			expected: true,
		},
		{
			name: "Docker Desktop Cluster Local Registry Enabled skip push",
			client: &kubectltesting.Client{
				Context: "docker-desktop",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(true),
				},
			},
			skipPush: true,
			expected: false,
		},
		{
			name: "Docker Desktop Cluster Local Registry Fallback",
			client: &kubectltesting.Client{
				Context: "docker-desktop",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: nil,
				},
			},
			expected: false,
		},
		{
			name: "VCluster with Docker Desktop Cluster",
			client: &kubectltesting.Client{
				Context: "vcluster_devspacehelper_deploy-example_docker-desktop",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Loft VCluster with Docker Desktop Cluster",
			client: &kubectltesting.Client{
				Context: "loft-vcluster_devspacehelper_deploy-example_docker-desktop",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Minikube Cluster",
			client: &kubectltesting.Client{
				Context: "minikube",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Minikube Cluster Local Registry Enabled",
			client: &kubectltesting.Client{
				Context: "minikube",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(true),
				},
			},
			expected: true,
		},
		{
			name: "Minikube Cluster Local Registry Enabled skip push",
			client: &kubectltesting.Client{
				Context: "minikube",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(true),
				},
			},
			skipPush: true,
			expected: false,
		},
		{
			name: "Minikube Cluster Local Registry Fallback",
			client: &kubectltesting.Client{
				Context: "minikube",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: nil,
				},
			},
			expected: false,
		},
		{
			name: "VCluster with Minikube Cluster",
			client: &kubectltesting.Client{
				Context: "vcluster_devspace-minikube_vcluster-devspace-minikube_minikube",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Loft VCluster with Minikube Cluster",
			client: &kubectltesting.Client{
				Context: "loft-vcluster_devspace-minikube_vcluster-devspace-minikube_minikube",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Remote Cluster",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: true,
		},
		{
			name: "Remote Cluster skip push",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			skipPush: true,
			expected: false,
		},
		{
			name: "Remote Cluster Local Registry Disabled",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: &latest.LocalRegistryConfig{
					Enabled: ptr.Bool(false),
				},
			},
			expected: false,
		},
		{
			name: "VCluster with Remote Cluster",
			client: &kubectltesting.Client{
				Context: "vcluster_vcluster-eks_vcluster-vcluster-eks_arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: true,
		},
		{
			name: "Loft VCluster with Remote Cluster",
			client: &kubectltesting.Client{
				Context: "loft-vcluster_vcluster-eks_vcluster-vcluster-eks_arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: true,
		},
		{
			name: "VCluster with Remote Cluster skip push",
			client: &kubectltesting.Client{
				Context: "vcluster_vcluster-eks_vcluster-vcluster-eks_arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			skipPush: true,
			expected: false,
		},
		{
			name: "Loft VCluster with Remote Cluster skip push",
			client: &kubectltesting.Client{
				Context: "loft-vcluster_vcluster-eks_vcluster-vcluster-eks_arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			skipPush: true,
			expected: false,
		},
		{
			name:   "Nil KubeClient",
			client: nil,
			config: &latest.Config{
				LocalRegistry: nil,
			},
			expected: false,
		},
		{
			name: "Remote Cluster BuildKit In Cluster Build Config",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			imageConfig: &latest.Image{
				BuildKit: &latest.BuildKitConfig{
					InCluster: &latest.BuildKitInClusterConfig{},
				},
			},
			expected: false,
		},
		{
			name: "Remote Cluster BuildKit Build Config",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			imageConfig: &latest.Image{
				BuildKit: &latest.BuildKitConfig{},
			},
			expected: true,
		},
		{
			name: "Remote Cluster Kaniko Build Config",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			imageConfig: &latest.Image{
				Kaniko: &latest.KanikoConfig{},
			},
			expected: false,
		},
		{
			name: "Remote Cluster Custom Build Config",
			client: &kubectltesting.Client{
				Context: "arn:aws:eks:us-west-2:1234567890:cluster/remote-eks",
			},
			config: &latest.Config{
				LocalRegistry: nil,
			},
			imageConfig: &latest.Image{
				Custom: &latest.CustomConfig{},
			},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		actual := UseLocalRegistry(testCase.client, testCase.config, testCase.imageConfig, testCase.skipPush)
		assert.Equal(t, actual, testCase.expected, "Unexpected result in test case %s", testCase.name)
	}
}
