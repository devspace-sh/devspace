package services

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"

	"gotest.tools/assert"
)

func TestGetCommand(t *testing.T) {
	client := &client{
		config: &latest.Config{
			Dev: latest.DevConfig{
				Terminal: &latest.Terminal{
					Command: []string{"echo"},
				},
			},
		},
	}
	command := client.getCommand([]string{"args"}, "")
	assert.Equal(t, 1, len(command), "Returned command has wrong length")
	assert.Equal(t, "args", command[0], "Wrong command returned")

	client.config = &latest.Config{}
	command = client.getCommand([]string{}, "")
	assert.Equal(t, 3, len(command), "Returned command has wrong length")
	assert.Equal(t, "sh", command[0], "Wrong command returned")
	assert.Equal(t, "-c", command[1], "Wrong command returned")
	assert.Equal(t, "command -v bash >/dev/null 2>&1 && exec bash || exec sh", command[2], "Wrong command returned")
}
