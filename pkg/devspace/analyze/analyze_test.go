package analyze

import(
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/log"
)

type testLogger struct {
	log.DiscardLogger
}

var writtenText string

func (l *testLogger) WriteString(message string){
	writtenText += message
}

func TestAnalyze(t *testing.T) {
	writtenText = ""

	//@MoreTest
	//Right now, CreateReport and Analyze are untestable because they create a new kubernetes-client that can't be faked
}
