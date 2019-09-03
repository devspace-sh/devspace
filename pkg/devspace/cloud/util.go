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
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// PrintSpaces prints the users spaces
func (p *Provider) PrintSpaces(cluster, name string, all bool) error {
	spaces, err := p.GetSpaces()
	if err != nil {
		return fmt.Errorf("Error retrieving spaces: %v", err)
	}

	activeSpaceID := 0
	currentContext, _, err := kubeconfig.GetCurrentContext()
	if err == nil {
		activeSpaceID, _, _ = kubeconfig.GetSpaceID(currentContext)
	}

	headerColumnNames := []string{}
	if activeSpaceID != 0 {
		headerColumnNames = append(headerColumnNames, []string{
			"SpaceID",
			"Name",
			"Cluster",
			"Active",
			"Domains",
			"Created",
		}...)
	} else {
		headerColumnNames = append(headerColumnNames, []string{
			"SpaceID",
			"Name",
			"Cluster",
			"Domains",
			"Created",
		}...)
	}

	values := [][]string{}

	bearerToken, err := p.GetToken()
	if err != nil {
		return errors.Wrap(err, "get token")
	}

	accountID, err := token.GetAccountID(bearerToken)
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

			domains := make([]string, 0, len(space.Domains))
			for _, domain := range space.Domains {
				domains = append(domains, domain.URL)
			}

			if activeSpaceID != 0 {
				values = append(values, []string{
					strconv.Itoa(space.SpaceID),
					space.Name,
					space.Cluster.Name,
					strconv.FormatBool(space.SpaceID == activeSpaceID),
					strings.Join(domains, ", "),
					created.String(),
				})
			} else {
				values = append(values, []string{
					strconv.Itoa(space.SpaceID),
					space.Name,
					space.Cluster.Name,
					strings.Join(domains, ", "),
					created.String(),
				})
			}
		}
	}

	if len(values) > 0 {
		log.PrintTable(log.GetInstance(), headerColumnNames, values)
	} else {
		log.Info("No spaces found")
	}

	return nil
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
