package connect

// import (
// 	"strings"
// 	"testing"

// 	"github.com/devspace-cloud/devspace/pkg/util/factory"
// 	"gotest.tools/assert"
// )

// func TestNewConnectCmd(t *testing.T) {
// 	connectCmd := NewConnectCmd()
// 	subcommands := connectCmd.Commands()

// 	expectedSubcommandNames := []string{"cluster"}
// 	for _, subcommand := range subcommands {
// 		subCommandName := subcommand.Name()
// 		index := pos(expectedSubcommandNames, subCommandName)
// 		assert.Equal(t, true, index > -1, "Wrong subcommand "+subCommandName)
// 		expectedSubcommandNames = append(expectedSubcommandNames[:index], expectedSubcommandNames[index+1:]...)
// 	}
// 	assert.Equal(t, 0, len(expectedSubcommandNames), "Some subcommands of add are missing: "+strings.Join(expectedSubcommandNames, ", "))
// }

// func pos(slice []string, value string) int {
// 	for p, v := range slice {
// 		if v == value {
// 			return p
// 		}
// 	}
// 	return -1
// }
