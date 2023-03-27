package restarthelper

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/e2e/kube"
	"github.com/loft-sh/devspace/pkg/util/factory"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	scanner2 "github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = DevSpaceDescribe("restarthelper", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f           factory.Factory
		kubeClient  *kube.KubeHelper
		specTimeout = ginkgo.SpecTimeout(30 * time.Second)
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()

		kubeClient, err = kube.NewKubeHelper()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should inject legacy restart helper after v1beta11 schema upgrade and start container", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/restarthelper/testdata/v5")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("restarthelper")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		reader, writer, err := os.Pipe()
		framework.ExpectNoError(err)

		teeReader := io.TeeReader(reader, os.Stdout)
		scanner := bufio.NewScanner(teeReader)
		scanner.Split(scanner2.ScanLines)

		//output := &bytes.Buffer{}
		log := logpkg.NewStreamLogger(writer, writer, logrus.DebugLevel)

		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
					Debug:     true,
				},
				Pipeline: "dev",
				SkipPush: true,
				Ctx:      cancelCtx,
				Log:      log,
			}
			err = devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		waitSeen := false
		waitCount := 0
		waitMax := 3
		waitingMessage := "(Still waiting...)"
		startedMessage := "Started with legacy restart helper"
		startedSeen := false

		for scanner.Scan() {
			text := scanner.Text()
			hasStartedMessage := strings.HasSuffix(text, startedMessage)
			if hasStartedMessage {
				startedSeen = true
				break
			}

			hasWaitingMessage := strings.HasSuffix(text, waitingMessage)
			if hasWaitingMessage {
				if !waitSeen {
					waitSeen = true
				}

				if waitSeen {
					waitCount++
				}
			}

			if waitCount > waitMax {
				break
			}
		}

		cancel()
		<-done

		gomega.Expect(waitCount).Should(gomega.BeNumerically("<", waitMax), "restart helper is waiting longer than expected")
		gomega.Expect(startedSeen).Should(gomega.BeTrue(), "container should have started")
	}, specTimeout)

	ginkgo.It("should automatically inject restart helper and start container", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/restarthelper/testdata/v6")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("restarthelper")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		reader, writer, err := os.Pipe()
		framework.ExpectNoError(err)

		teeReader := io.TeeReader(reader, os.Stdout)
		scanner := bufio.NewScanner(teeReader)
		scanner.Split(scanner2.ScanLines)

		//output := &bytes.Buffer{}
		log := logpkg.NewStreamLogger(writer, writer, logrus.DebugLevel)

		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
					Debug:     true,
				},
				Pipeline: "dev",
				SkipPush: true,
				Ctx:      cancelCtx,
				Log:      log,
			}
			err = devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		waitSeen := false
		waitCount := 0
		waitMax := 3
		waitingMessage := "(Still waiting...)"
		startedMessage := "Started with dev command entrypoint"
		startedSeen := false

		for scanner.Scan() {
			text := scanner.Text()
			hasStartedMessage := strings.HasSuffix(text, startedMessage)
			if hasStartedMessage {
				startedSeen = true
				break
			}

			hasWaitingMessage := strings.HasSuffix(text, waitingMessage)
			if hasWaitingMessage {
				if !waitSeen {
					waitSeen = true
				}

				if waitSeen {
					waitCount++
				}
			}

			if waitCount > waitMax {
				break
			}
		}

		cancel()
		<-done

		gomega.Expect(waitCount).Should(gomega.BeNumerically("<", waitMax), "restart helper is waiting longer than expected")
		gomega.Expect(startedSeen).Should(gomega.BeTrue(), "container should have started")
	}, specTimeout)

	ginkgo.It("should manually inject restart helper and automatically override with current restart helper", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/restarthelper/testdata/v6-inject-restart-helper")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("restarthelper")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		reader, writer, err := os.Pipe()
		framework.ExpectNoError(err)

		teeReader := io.TeeReader(reader, os.Stdout)
		scanner := bufio.NewScanner(teeReader)
		scanner.Split(scanner2.ScanLines)

		//output := &bytes.Buffer{}
		log := logpkg.NewStreamLogger(writer, writer, logrus.DebugLevel)

		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
					Debug:     true,
				},
				Pipeline: "dev",
				SkipPush: true,
				Ctx:      cancelCtx,
				Log:      log,
			}
			err = devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		waitSeen := false
		waitCount := 0
		waitMax := 3
		waitingMessage := "(Still waiting...)"
		startedMessage := "Started with dev command entrypoint"
		startedSeen := false
		legacyStartedMessage := "Started with legacy restart helper"
		legacyStartedSeen := false

		for scanner.Scan() {
			text := scanner.Text()
			hasStartedMessage := strings.HasSuffix(text, startedMessage)
			if hasStartedMessage {
				startedSeen = true
				break
			}

			hasLegacyStartedMessage := strings.HasSuffix(text, legacyStartedMessage)
			if hasLegacyStartedMessage {
				legacyStartedSeen = true
				break
			}

			hasWaitingMessage := strings.HasSuffix(text, waitingMessage)
			if hasWaitingMessage {
				if !waitSeen {
					waitSeen = true
				}

				if waitSeen {
					waitCount++
				}
			}

			if waitCount > waitMax {
				break
			}
		}

		cancel()
		<-done

		gomega.Expect(waitCount).Should(gomega.BeNumerically("<", waitMax), "restart helper is waiting longer than expected")
		gomega.Expect(startedSeen).Should(gomega.BeTrue(), "container should have started")
		gomega.Expect(legacyStartedSeen).Should(gomega.BeFalse(), "container should not have started with legacy helper")
	}, specTimeout)

	ginkgo.It("should manually inject restart helper and require manual start", func(ctx context.Context) {
		tempDir, err := framework.CopyToTempDir("tests/restarthelper/testdata/v6-manual-start")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		ns, err := kubeClient.CreateNamespace("restarthelper")
		framework.ExpectNoError(err)
		defer framework.ExpectDeleteNamespace(kubeClient, ns)

		// create a new dev command and start it
		done := make(chan error)
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		reader, writer, err := os.Pipe()
		framework.ExpectNoError(err)

		teeReader := io.TeeReader(reader, os.Stdout)
		scanner := bufio.NewScanner(teeReader)
		scanner.Split(scanner2.ScanLines)

		//output := &bytes.Buffer{}
		log := logpkg.NewStreamLogger(writer, writer, logrus.DebugLevel)

		go func() {
			devCmd := &cmd.RunPipelineCmd{
				GlobalFlags: &flags.GlobalFlags{
					NoWarn:    true,
					Namespace: ns,
					Debug:     true,
				},
				Pipeline: "dev",
				SkipPush: true,
				Ctx:      cancelCtx,
				Log:      log,
			}
			err = devCmd.RunDefault(f)
			if err != nil {
				f.GetLog().Errorf("error: %v", err)
			}
			done <- err
		}()

		waitSeen := false
		waitCount := 0
		waitMax := 3
		waitingMessage := "(Still waiting...)"
		startedMessage := "Started manually"
		startedSeen := false

		for scanner.Scan() {
			text := scanner.Text()
			hasStartedMessage := strings.HasSuffix(text, startedMessage)
			if hasStartedMessage {
				startedSeen = true
				break
			}

			hasWaitingMessage := strings.HasSuffix(text, waitingMessage)
			if hasWaitingMessage {
				if !waitSeen {
					waitSeen = true
				}

				if waitSeen {
					waitCount++
				}
			}

			if waitCount > waitMax {
				break
			}
		}

		cancel()
		<-done

		gomega.Expect(waitCount).Should(gomega.BeNumerically("<", waitMax), "restart helper is waiting longer than expected")
		gomega.Expect(startedSeen).Should(gomega.BeTrue(), "container should have started")
	}, specTimeout)
})
