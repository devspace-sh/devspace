package cloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"

	"github.com/devspace-cloud/devspace/pkg/util/envutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// PrintSpaces prints the users spaces
func (p *Provider) PrintSpaces(name string) error {
	spaces, err := p.GetSpaces()
	if err != nil {
		return fmt.Errorf("Error retrieving spaces: %v", err)
	}

	activeSpaceID := 0
	if configutil.ConfigExists() {
		generated, err := generated.LoadConfig()
		if err == nil && generated.CloudSpace != nil {
			activeSpaceID = generated.CloudSpace.SpaceID
		}
	}

	headerColumnNames := []string{}
	if activeSpaceID != 0 {
		headerColumnNames = append(headerColumnNames, []string{
			"SpaceID",
			"Name",
			"Active",
			"Domain",
			"Created",
		}...)
	} else {
		headerColumnNames = append(headerColumnNames, []string{
			"SpaceID",
			"Name",
			"Domain",
			"Created",
		}...)
	}

	values := [][]string{}

	for _, space := range spaces {
		if name == "" || name == space.Name {
			created, err := time.Parse(time.RFC3339, strings.Split(space.Created, ".")[0]+"Z")
			if err != nil {
				return err
			}

			domain := ""
			if space.Domain != nil {
				domain = *space.Domain
			}

			if activeSpaceID != 0 {
				values = append(values, []string{
					strconv.Itoa(space.SpaceID),
					space.Name,
					strconv.FormatBool(space.SpaceID == activeSpaceID),
					domain,
					created.String(),
				})
			} else {
				values = append(values, []string{
					strconv.Itoa(space.SpaceID),
					space.Name,
					domain,
					created.String(),
				})
			}
		}
	}

	if len(values) > 0 {
		log.PrintTable(headerColumnNames, values)
	} else {
		log.Info("No spaces found")
	}

	return nil
}

// SetTillerNamespace sets the tiller environment variable
func SetTillerNamespace(serviceAccount *ServiceAccount) error {
	if serviceAccount == nil {
		return envutil.SetEnvVar("TILLER_NAMESPACE", "kube-system")
	}

	return envutil.SetEnvVar("TILLER_NAMESPACE", serviceAccount.Namespace)
}
