package sync

import (
	"bytes"
	"strings"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

func TestInitialSyncCompletedMessage(t *testing.T) {
	testCases := []struct {
		name               string
		localPath          string
		remotePath         string
		prefix             string
		expectedTarget     string
		expectedLogMessage string
	}{
		{
			name:               "both local and remote path set",
			localPath:          "/workspace/project",
			remotePath:         "/app",
			prefix:             "Upstream",
			expectedTarget:     "/workspace/project <-> /app",
			expectedLogMessage: "Upstream - Initial sync completed for /workspace/project <-> /app",
		},
		{
			name:               "downstream with both paths set",
			localPath:          "/workspace/project",
			remotePath:         "/app",
			prefix:             "Downstream",
			expectedTarget:     "/workspace/project <-> /app",
			expectedLogMessage: "Downstream - Initial sync completed for /workspace/project <-> /app",
		},
		{
			name:               "only local path set",
			localPath:          "/workspace/project",
			remotePath:         "",
			prefix:             "Upstream",
			expectedTarget:     "/workspace/project",
			expectedLogMessage: "Upstream - Initial sync completed for /workspace/project",
		},
		{
			name:               "only remote path set",
			localPath:          "",
			remotePath:         "/app",
			prefix:             "Upstream",
			expectedTarget:     "/app",
			expectedLogMessage: "Upstream - Initial sync completed for /app",
		},
		{
			name:               "neither path set",
			localPath:          "",
			remotePath:         "",
			prefix:             "Upstream",
			expectedTarget:     "",
			expectedLogMessage: "Upstream - Initial sync completed",
		},
	}

	for _, tc := range testCases {
		s := &Sync{
			LocalPath: tc.localPath,
			Options: Options{
				RemotePath: tc.remotePath,
			},
		}

		assert.Equal(t, s.TargetPath(), tc.expectedTarget, tc.name)
		assert.Equal(t, s.InitialSyncCompletedMessage(tc.prefix), tc.expectedLogMessage, tc.name)
	}
}

func TestInitialSyncLoggingWithTargetContext(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.NewStreamLogger(&buf, &buf, logrus.InfoLevel)

	s := &Sync{
		LocalPath: "/workspace/frontend",
		Options: Options{
			RemotePath: "/src/frontend",
			Log:        testLogger,
		},
		log: testLogger,
	}

	upstreamMsg := s.InitialSyncCompletedMessage("Upstream")
	s.log.Info(upstreamMsg)

	downstreamMsg := s.InitialSyncCompletedMessage("Downstream")
	s.log.Info(downstreamMsg)

	logOutput := buf.String()
	assert.Assert(t, strings.Contains(logOutput, "Upstream - Initial sync completed for /workspace/frontend <-> /src/frontend"), "expected upstream completion log with target context")
	assert.Assert(t, strings.Contains(logOutput, "Downstream - Initial sync completed for /workspace/frontend <-> /src/frontend"), "expected downstream completion log with target context")
}
