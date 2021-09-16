package legacy

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"testing"
)

type testCase struct {
	config           string
	forceInteractive bool
	forceTerminal    bool

	expectedConfig         string
	expectedWasInteractive bool
}

func TestLegacyInteractiveMode(t *testing.T) {
	testCases := []testCase{
		{
			config: `version: v1beta10
images:
  default:
    image: test
dev:
  deprecatedInteractiveEnabled: true`,
			forceInteractive: true,

			expectedWasInteractive: true,
			expectedConfig: `version: v1beta10
images:
  default:
    image: test
    entrypoint:
    - sleep
    cmd:
    - "999999999"
dev:
  terminal:
    imageSelector: image(default):tag(default)
`,
		},
		{
			config: `version: v1beta10
images:
  default:
    image: test
dev:
  terminal: {}`,
			forceInteractive: true,

			expectedWasInteractive: true,
			expectedConfig: `version: v1beta10
images:
  default:
    image: test
dev:
  terminal: {}
`,
		},
		{
			config: `version: v1beta10
images:
  default:
    image: test
dev:
  terminal: {}`,
			expectedWasInteractive: false,
			expectedConfig: `version: v1beta10
images:
  default:
    image: test
dev:
  terminal: {}
`,
		},
	}

	for _, tc := range testCases {
		origConfig := &latest.Config{}
		err := yaml.Unmarshal([]byte(tc.config), origConfig)
		if err != nil {
			t.Fatal(err)
		}

		wasInteractive, err := LegacyInteractiveMode(origConfig, tc.forceInteractive, tc.forceTerminal, logpkg.Discard)
		assert.NilError(t, err)
		assert.Equal(t, wasInteractive, tc.expectedWasInteractive, "was interactive")

		out, err := yaml.Marshal(origConfig)
		assert.NilError(t, err)
		assert.Equal(t, string(out), tc.expectedConfig, "expected config")
	}
}
