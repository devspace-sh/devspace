package cloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/util/log"
)

// PrintSpaces prints the users spaces
func (p *Provider) PrintSpaces(name string) error {
	devspaces, err := p.GetSpaces()
	if err != nil {
		return fmt.Errorf("Error retrieving devspaces: %v", err)
	}

	headerColumnNames := []string{
		"SpaceID",
		"Name",
		"Created",
	}
	values := [][]string{}

	for _, devspace := range devspaces {
		if name == "" || name == devspace.Name {
			created, err := time.Parse(time.RFC3339, strings.Split(devspace.Created, ".")[0]+"Z")
			if err != nil {
				return err
			}

			values = append(values, []string{
				strconv.Itoa(devspace.SpaceID),
				devspace.Name,
				created.String(),
			})
		}
	}

	if len(values) > 0 {
		log.PrintTable(headerColumnNames, values)
	} else {
		log.Info("No spaces found")
	}

	return nil
}
