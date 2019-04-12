package cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/token"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/util/envutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// PrintSpaces prints the users spaces
func (p *Provider) PrintSpaces(cluster, name string, all bool) error {
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

	accountID, err := token.GetAccountID(p.Token)
	if err != nil {
		return errors.Wrap(err, "get account id")
	}

	for _, space := range spaces {
		if name == "" || name == space.Name {
			if cluster != "" && cluster != space.Cluster.Name {
				continue
			}
			if all == false && space.Owner.OwnerID != accountID {
				continue
			}

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

// PadKey formats the key to the correct padding (32 byte)
func PadKey(key []byte) []byte {
	if len(key) == 32 {
		return key
	} else if len(key) > 32 {
		return key[:32]
	}

	// Append to key this wont change the key
	for len(key) < 32 {
		key = append(key, ' ')
	}

	return key
}

// EncryptAES encrypts the given data with the given key
func EncryptAES(key, data []byte) ([]byte, error) {
	// Ensure key is 32 bytes long
	key = PadKey(key)

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// DecryptAES decrypts the given data with the given key
func DecryptAES(key, data []byte) ([]byte, error) {
	// Ensure key is 32 bytes long
	key = PadKey(key)

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("Data size is smaller than nonce size: %d < %d", len(data), nonceSize)
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
