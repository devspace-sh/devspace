package config

import (
	"bytes"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"gopkg.in/yaml.v3"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"

	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/onsi/gomega"
)

var _ = DevSpaceDescribe("config", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f          *framework.DefaultFactory
		kubeClient *kube.KubeHelper
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()

		kubeClient, err = kube.NewKubeHelper()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should convert correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/convert")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// reload it
		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace.yaml", &loader.ConfigOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Dev), 1)
		framework.ExpectEqual(config.Config().Dev["sync-0"].Container, "test")
		framework.ExpectEqual(config.Config().Dev["sync-0"].Terminal.Command, "test")
	})

	ginkgo.It("should patch correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// reload it
		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), "var-patch.yaml", &loader.ConfigOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(config.Config().Images["test"].Image, "sample")
		framework.ExpectEqual(config.Config().Images["test"].Kaniko.Labels, map[string]string{"app": "sample"})

		// with profile
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "var-patch.yaml", &loader.ConfigOptions{Profiles: []string{"alt"}})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(config.Config().Images["test"].Image, "sample-alt")
		framework.ExpectEqual(config.Config().Images["test"].Kaniko.Labels, map[string]string{"app": "sample-alt"})
	})

	ginkgo.It("should resolve runtime environment variables correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/runtime-variables")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("config")
		framework.ExpectNoError(err)
		defer func() {
			err := kubeClient.DeleteNamespace(ns)
			framework.ExpectNoError(err)
		}()

		printCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: ns,
			},
			Pipeline: "deploy",
		}

		err = printCmd.RunDefault(f)
		framework.ExpectNoError(err)
		framework.ExpectLocalFileContentsImmediately(filepath.Join(tempDir, "out.txt"), "test-testimage-latest-dep1")
		framework.ExpectLocalFileContentsImmediately(filepath.Join(tempDir, "out2.txt"), "Done")
		framework.ExpectLocalFileContentsImmediately(filepath.Join(tempDir, "out3.txt"), ns+"-resolved-${NOT_RESOLVED}")

		// read the generated.yaml
		config, err := localcache.NewCacheLoader().Load(constants.DefaultConfigPath)
		framework.ExpectNoError(err)

		ic, _ := config.GetImageCache("app-test")
		out, err := os.ReadFile(filepath.Join(tempDir, "out0.txt"))
		framework.ExpectNoError(err)
		gomega.Expect(string(out)).To(gomega.MatchRegexp("my-docker-username/helloworld2:" + ic.Tag))
	})

	ginkgo.It("should load multiple profiles in order via --profile", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/multiple-profiles")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				Profiles: []string{"one", "two", "three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")

		// run without profile
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{},
			Out:         configBuffer,
			SkipInfo:    true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 0)
	})

	ginkgo.It("should filter duplicate profiles via --profile", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/multiple-profiles")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				Profiles: []string{"one", "three", "three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 3)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
	})

	ginkgo.It("should filter duplicate profiles via --profile and --profile-parent", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/multiple-profiles")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "profiles.yaml",
				Profiles:   []string{"two", "one", "one", "three", "one", "two"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 4)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
		framework.ExpectEqual(latestConfig.Deployments["test1"].Name, "test1")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should order profiles correctly via --profile and --profile-parent", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/multiple-profiles")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "profiles.yaml",
				Profiles:   []string{"one", "two", "three", "four"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 5)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
		framework.ExpectEqual(latestConfig.Deployments["test4"].Name, "test4")
		framework.ExpectEqual(latestConfig.Deployments["test1"].Name, "test1")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should not be able to add in patch if key already exists", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/patch-add-dont-overwrite-existing-key")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "devspace.yaml",
				Profiles:   []string{"deploy"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectError(err)
	})

	ginkgo.It("should be able to add in patch if key does not already exists", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/patch-add-dont-overwrite-existing-key")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "devspace.yaml",
				Profiles:   []string{"patch-ok"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(string(latestConfig.Images["importme"].RebuildStrategy), "ignoreContextChanges")
	})

	ginkgo.It("should load profile cached and uncached", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "test", nil
		})

		// load it without profile
		config, _, err := framework.LoadConfig(f, kubeClient.Client(), "devspace.yaml")
		framework.ExpectNoError(err)

		// check no profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 1)

		// now set the profile via command
		profileCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				Profiles: []string{"does-not-exist"},
			},
		}

		// try to set non existing profile
		err = profileCmd.Run(f)
		framework.ExpectError(err)

		// reload it
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace.yaml", &loader.ConfigOptions{Profiles: []string{"remove-image"}})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 0)
		framework.ExpectEqual(len(config.Config().Deployments), 1)

		// reload it and set it through config options
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace.yaml", &loader.ConfigOptions{Profiles: []string{"add-deployment"}})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 2)

		// reload it with different profile
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace.yaml", &loader.ConfigOptions{Profiles: []string{"merge-deployments"}})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 1)
		framework.ExpectEqual(config.Config().Deployments["test456"].Name, "test456")
		framework.ExpectEqual(len(config.Config().Deployments["test456"].Kubectl.Manifests), 2)
		framework.ExpectEqual(config.Config().Deployments["test456"].Kubectl.Manifests[0], "test")
		framework.ExpectEqual(config.Config().Deployments["test456"].Kubectl.Manifests[1], "test.yaml")

		// load new config
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace_new.yaml", &loader.ConfigOptions{Profiles: []string{"merge-deployments"}})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 2)

		framework.ExpectEqual(config.Config().Deployments["test"].Name, "test")
		framework.ExpectEqual(len(config.Config().Deployments["test"].Kubectl.Manifests), 1)
		framework.ExpectEqual(config.Config().Deployments["test"].Kubectl.Manifests[0], "test")

		framework.ExpectEqual(config.Config().Deployments["test456"].Name, "test456")
		framework.ExpectEqual(len(config.Config().Deployments["test456"].Kubectl.Manifests), 2)
		framework.ExpectEqual(config.Config().Deployments["test456"].Kubectl.Manifests[0], "test")
		framework.ExpectEqual(config.Config().Deployments["test456"].Kubectl.Manifests[1], "test.yaml")

		// load new config
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace_new.yaml", &loader.ConfigOptions{Profiles: []string{"add-deployment"}})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 2)

		framework.ExpectEqual(config.Config().Deployments["test"].Name, "test")
		framework.ExpectEqual(len(config.Config().Deployments["test"].Kubectl.Manifests), 1)
		framework.ExpectEqual(config.Config().Deployments["test"].Kubectl.Manifests[0], "test")

		framework.ExpectEqual(config.Config().Deployments["test678"].Name, "test678")
		framework.ExpectEqual(len(config.Config().Deployments["test678"].Kubectl.Manifests), 1)
		framework.ExpectEqual(config.Config().Deployments["test678"].Kubectl.Manifests[0], "test")

		// load new config
		config, _, err = framework.LoadConfigWithOptions(f, kubeClient.Client(), "devspace_new.yaml", &loader.ConfigOptions{Profiles: []string{"replace-manifests"}})
		framework.ExpectNoError(err)

		// check profile was loaded
		framework.ExpectEqual(len(config.Config().Images), 1)
		framework.ExpectEqual(len(config.Config().Deployments), 1)

		framework.ExpectEqual(config.Config().Deployments["test123"].Name, "test123")
		framework.ExpectEqual(len(config.Config().Deployments["test123"].Kubectl.Manifests), 2)
		framework.ExpectEqual(config.Config().Deployments["test123"].Kubectl.Manifests[0], "test123.yaml")
		framework.ExpectEqual(config.Config().Deployments["test123"].Kubectl.Manifests[1], "test")
	})

	ginkgo.It("should auto activate profile using single environment variable", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/default")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with non-matching environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "false")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with matching environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using single var", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/default")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching environment variable set.
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with matching environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using exact string matching environment variable", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/string-exact")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "test123")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "test")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using exact string matching vars", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/string-exact")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=test123"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=test"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using exact regular expression matching environment variable", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/regexp-exact")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "some test here")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "test")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using exact regular expression matching vars", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/regexp-exact")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=some test here"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=test"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using regular expression matching environment variable", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/regexp")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "false")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "^the string begins with ^t and ends with $")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using regular expression matching vars", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/regexp")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=^the string begins with ^t and ends with $"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using regular expression matching environment variable substring", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/regexp-substring")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "the best string")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "a test string")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using regular expression matching vars substring", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/regexp-substring")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run with non-matching vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=the best string"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=a test string"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should not auto activate profile using single environment variable with --disable-profile-activation", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/default")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath:               "env.yaml",
				DisableProfileActivation: true,
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath:               "env.yaml",
				DisableProfileActivation: true,
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)
	})

	ginkgo.It("should not auto activate profile using vars with --disable-profile-activation", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/default")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath:               "vars.yaml",
				Vars:                     []string{"FOO=false"},
				DisableProfileActivation: true,
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath:               "vars.yaml",
				Vars:                     []string{"FOO=true"},
				DisableProfileActivation: true,
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)
	})

	ginkgo.It("should auto activate profile using multiple environment variables", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/and")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with single environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with both environment variables set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("BAR", "true")
		defer os.Unsetenv("BAR")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using multiple vars", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/and")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=false", "BAR=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with single var matching.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=true", "BAR=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with both vars matching.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=true", "BAR=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using vars and environment variables", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/and")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without environment variable
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env-and-vars.yaml",
				Vars:       []string{"BAR=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with vars matching and environment variable not matching.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env-and-vars.yaml",
				Vars:       []string{"BAR=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		os.Setenv("FOO", "false")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with vars not matching and environment variable matching.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env-and-vars.yaml",
				Vars:       []string{"BAR=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with both vars and environment variable matching.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env-and-vars.yaml",
				Vars:       []string{"BAR=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using multiple environment variable activations", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/or")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with FOO environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("FOO")

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")

		// run with BAR environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("BAR", "true")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)
		os.Unsetenv("BAR")

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate profile using multiple vars activations", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/or")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=false", "BAR=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with FOO var set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=true", "BAR=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")

		// run with BAR var set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=false", "BAR=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should auto activate multiple profiles using single environment variable", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
	})

	ginkgo.It("should auto activate multiple profiles using single var", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "vars.yaml",
				Vars:       []string{"FOO=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
	})

	ginkgo.It("should auto activate multiple profiles using both vars and env activation", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env-and-vars.yaml",
				Vars:       []string{"FOO=false"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 0)

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env-and-vars.yaml",
				Vars:       []string{"FOO=true"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
	})

	ginkgo.It("should auto activate multiple profiles using single environment variable and --profile flag", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate no profile was activated
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
	})

	ginkgo.It("should auto activate profile once using single environment variable and multiple --profile flags", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile three was activated once
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("BAR", "true")
		defer os.Unsetenv("BAR")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile three was activated once
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
	})

	ginkgo.It("should auto activate profile once using single environment variable and multiple --profile and --profile-parent flags", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"three", "three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile three was activated once
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"three", "three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("BAR", "true")
		defer os.Unsetenv("BAR")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate profile three was activated once
		framework.ExpectEqual(len(latestConfig.Deployments), 1)
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
	})

	ginkgo.It("should auto activate multiple profiles using single environment variable and --profile flags in order", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"four", "three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 2)
		framework.ExpectEqual(latestConfig.Deployments["test4"].Name, "test4")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")

		// run with environment variable set.
		configBuffer = &bytes.Buffer{}
		printCmd = &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "env.yaml",
				Profiles:   []string{"four", "three"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig = &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 3)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test4"].Name, "test4")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
	})

	ginkgo.It("should auto activate multiple profiles using single environment variable and --profile and --profile-parent flags in order", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-activation")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// run without vars
		configBuffer := &bytes.Buffer{}
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				ConfigPath: "profiles.yaml",
				Profiles:   []string{"two", "one", "one", "three", "one", "two"},
			},
			Out:      configBuffer,
			SkipInfo: true,
		}

		// run with environment variable set.
		os.Setenv("FOO", "true")
		defer os.Unsetenv("FOO")
		err = printCmd.Run(f)
		framework.ExpectNoError(err)

		latestConfig := &latest.Config{}
		err = yaml.Unmarshal(configBuffer.Bytes(), latestConfig)
		framework.ExpectNoError(err)

		// validate config
		framework.ExpectEqual(len(latestConfig.Deployments), 5)
		framework.ExpectEqual(latestConfig.Deployments["test"].Name, "test")
		framework.ExpectEqual(latestConfig.Deployments["test5"].Name, "test5")
		framework.ExpectEqual(latestConfig.Deployments["test3"].Name, "test3")
		framework.ExpectEqual(latestConfig.Deployments["test1"].Name, "test1")
		framework.ExpectEqual(latestConfig.Deployments["test2"].Name, "test2")
	})

	ginkgo.It("should resolve variables correctly", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/vars")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		_ = os.Setenv("FROM_ENV", "FROM_ENV")

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "test", nil
		})

		// load it from the regular path first
		config, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if variables were loaded correctly
		framework.ExpectEqual(len(config.Variables()), 4+len(variable.AlwaysResolvePredefinedVars))
		framework.ExpectEqual(config.Variables()["FROM_ENV"], "FROM_ENV")
		framework.ExpectEqual(len(config.LocalCache().ListVars()), 1)
		test1, _ := config.LocalCache().GetVar("TEST_1")
		framework.ExpectEqual(test1, "test")
		framework.ExpectEqual(len(dependencies), 1)
		framework.ExpectEqual(len(dependencies[0].Config().LocalCache().ListVars()), 1)
		notUsed, _ := dependencies[0].Config().LocalCache().GetVar("NOT_USED2")
		framework.ExpectEqual(notUsed, "test")
		framework.ExpectEqual(dependencies[0].Config().Variables()["TEST_OVERRIDE"], "devspace.yaml")

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "", fmt.Errorf("shouldn't get asked again")
		})

		// rerun now with cached
		_, _, err = framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "dep1", nil
		})

		// rerun now with cached
		config, dependencies, err = framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "dep1", "dev.yaml"))
		framework.ExpectNoError(err)

		// config
		framework.ExpectEqual(len(config.Variables()), 3+len(variable.AlwaysResolvePredefinedVars))
		framework.ExpectEqual(len(config.LocalCache().ListVars()), 2)
		notUsed, _ = config.LocalCache().GetVar("NOT_USED2")
		framework.ExpectEqual(notUsed, "test")
		test2, _ := config.LocalCache().GetVar("TEST_2")
		framework.ExpectEqual(test2, "dep1")
		framework.ExpectEqual(config.Variables()["TEST_OVERRIDE"], "dev.yaml")
		framework.ExpectEqual(len(dependencies), 0)
	})

	ginkgo.It("should cache multiple configs independently", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/multiple")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "default", nil
		})

		// load it from the default path
		config, dependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// check if default config variables were loaded correctly
		framework.ExpectEqual(len(config.Variables()), 2+len(variable.AlwaysResolvePredefinedVars))
		framework.ExpectEqual(len(config.LocalCache().ListVars()), 1)
		value, _ := config.LocalCache().GetVar("NAME")
		framework.ExpectEqual(value, "default")
		framework.ExpectEqual(len(dependencies), 0)

		// set the question answer func here
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "custom", nil
		})

		// load it from a custom path
		customConfig, customDependencies, err := framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "custom.yaml"))
		framework.ExpectNoError(err)

		// check if custom config variables were loaded correctly
		framework.ExpectEqual(len(customConfig.Variables()), 2+len(variable.AlwaysResolvePredefinedVars))
		framework.ExpectEqual(len(customConfig.LocalCache().ListVars()), 1)
		value, _ = customConfig.LocalCache().GetVar("NAME")
		framework.ExpectEqual(value, "custom")
		framework.ExpectEqual(len(customDependencies), 0)

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "", fmt.Errorf("shouldn't get asked again")
		})

		// reload default config with cache
		_, _, err = framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "devspace.yaml"))
		framework.ExpectNoError(err)

		// make sure we don't get asked again
		f.SetAnswerFunc(func(params *survey.QuestionOptions) (string, error) {
			return "", fmt.Errorf("shouldn't get asked again")
		})

		// reload custom config with cache
		_, _, err = framework.LoadConfig(f, kubeClient.Client(), filepath.Join(tempDir, "custom.yaml"))
		framework.ExpectNoError(err)
	})

	ginkgo.It("should replace and add deployments using profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "replace-and-add-deployments.yaml"), &loader.ConfigOptions{
			Profiles: []string{"test"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 2)

		deployment1 := config.Config().Deployments["test"]
		framework.ExpectEqual(deployment1.Name, "test")
		framework.ExpectEqual(deployment1.Kubectl.Manifests[0], "test.yaml")

		deployment2 := config.Config().Deployments["test2"]
		framework.ExpectEqual(deployment2.Name, "test2")
		framework.ExpectEqual(deployment2.Kubectl.Manifests[0], "test2.yaml")
	})

	ginkgo.It("should apply patch to all deployments using wildcard profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "wildcard-match.yaml"), &loader.ConfigOptions{
			Profiles: []string{"test"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 2)

		deployment1 := config.Config().Deployments["test"]
		framework.ExpectEqual(deployment1.Name, "test")
		framework.ExpectEqual(deployment1.Kubectl.Manifests[0], "network-policy.yaml")

		deployment2 := config.Config().Deployments["test2"]
		framework.ExpectEqual(deployment2.Name, "test2")
		framework.ExpectEqual(deployment2.Kubectl.Manifests[0], "network-policy.yaml")
	})

	ginkgo.It("should apply patch to all deployments using regexp profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "wildcard-match-regexp.yaml"), &loader.ConfigOptions{
			Profiles: []string{"test"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 3)

		deployment1 := config.Config().Deployments["development1"]
		framework.ExpectEqual(deployment1.Name, "development1")
		gomega.Expect(deployment1.Kubectl).To(gomega.BeNil())

		deployment2 := config.Config().Deployments["staging1"]
		framework.ExpectEqual(deployment2.Name, "staging1")
		gomega.Expect(deployment2.Kubectl).To(gomega.BeNil())

		deployment3 := config.Config().Deployments["production1"]
		framework.ExpectEqual(deployment3.Name, "production1")
		framework.ExpectEqual(deployment3.Kubectl.Manifests[0], "network-policy.yaml")
	})

	ginkgo.It("should apply patch to deployments using legacy property match profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "legacy-match.yaml"), &loader.ConfigOptions{
			Profiles: []string{"test"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 2)

		deployment1 := config.Config().Deployments["test"]
		framework.ExpectEqual(deployment1.Name, "test")
		gomega.Expect(deployment1.Kubectl).To(gomega.BeNil())

		deployment2 := config.Config().Deployments["test2"]
		framework.ExpectEqual(deployment2.Name, "test2")
		framework.ExpectEqual(deployment2.Kubectl.Manifests[0], "network-policy.yaml")
	})

	ginkgo.It("should apply patch to all deployments using comparison profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "wildcard-match-comparison.yaml"), &loader.ConfigOptions{
			Profiles: []string{"test"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 1)

		deployment1 := config.Config().Deployments["development1"]
		framework.ExpectEqual(deployment1.Name, "development1")
		gomega.Expect(deployment1.Helm).ToNot(gomega.BeNil())
	})

	ginkgo.It("should apply patch to some deployments using wildcard profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "wildcard-match-some.yaml"), &loader.ConfigOptions{
			Profiles: []string{"test"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 2)

		deployment1 := config.Config().Deployments["test"]
		framework.ExpectEqual(deployment1.Name, "test")
		gomega.Expect(deployment1.Kubectl).To(gomega.BeNil())
		gomega.Expect(deployment1.Helm).ToNot(gomega.BeNil())

		deployment2 := config.Config().Deployments["test2"]
		framework.ExpectEqual(deployment2.Name, "test2")
		framework.ExpectEqual(deployment2.Kubectl.Manifests[0], "test2.yaml")
	})

	ginkgo.It("should apply patch to some deployments using recursive descent profile patches", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "recursive-descent.yaml"), &loader.ConfigOptions{
			Profiles: []string{"staging"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(config.Config().Images["backend"].Image, "john/stagingbackend")

		framework.ExpectEqual(len(config.Config().Deployments), 1)

		deployment1 := config.Config().Deployments["backend"]
		framework.ExpectEqual(deployment1.Name, "backend")
		gomega.Expect(deployment1.Kubectl).To(gomega.BeNil())
		gomega.Expect(deployment1.Helm).ToNot(gomega.BeNil())
	})

	ginkgo.It("should apply patch even value is empty", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "empty-patch-value.yaml"), &loader.ConfigOptions{
			Profiles: []string{"empty-value"},
		})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 1)

		deployment1 := config.Config().Deployments["test-sigsegv"]
		framework.ExpectEqual(deployment1.Name, "test-sigsegv")
		gomega.Expect(deployment1.Kubectl).To(gomega.BeNil())
		gomega.Expect(deployment1.Helm).ToNot(gomega.BeNil())
	})

	// regression test for issue: https://github.com/loft-sh/devspace/issues/1835
	ginkgo.It("should load config with var in patch", func() {
		tempDir, err := framework.CopyToTempDir("tests/config/testdata/profile-patches")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		config, _, err := framework.LoadConfigWithOptions(f, kubeClient.Client(), filepath.Join(tempDir, "path-variable.yaml"),
			&loader.ConfigOptions{Profiles: []string{"demo"}})
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(config.Config().Deployments), 1)

		deployment := config.Config().Deployments["test-me-server"]
		framework.ExpectEqual(deployment.Name, "test-me-server")

		values, ok := deployment.Helm.Values["containers"].([]interface{})
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(values).NotTo(gomega.BeEmpty())

		v, ok := values[0].(map[string]interface{})
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(v).NotTo(gomega.BeNil())

		framework.ExpectEqual(v["name"], "replace-0")

		gomega.Expect(deployment.Kubectl).To(gomega.BeNil())
		gomega.Expect(deployment.Helm).ToNot(gomega.BeNil())
	})
})
