package ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

var (
	DevSpaceSSHFolder         = "ssh"
	DevSpaceSSHHostKeyFile    = "id_devspace_host_ecdsa"
	DevSpaceSSHPrivateKeyFile = "id_devspace_ecdsa"
	DevSpaceSSHPublicKeyFile  = "id_devspace_ecdsa.pub"
)

func init() {
	homeDir, _ := homedir.Dir()
	DevSpaceSSHFolder = filepath.Join(homeDir, constants.DefaultHomeDevSpaceFolder, DevSpaceSSHFolder)
	DevSpaceSSHHostKeyFile = filepath.Join(DevSpaceSSHFolder, DevSpaceSSHHostKeyFile)
	DevSpaceSSHPrivateKeyFile = filepath.Join(DevSpaceSSHFolder, DevSpaceSSHPrivateKeyFile)
	DevSpaceSSHPublicKeyFile = filepath.Join(DevSpaceSSHFolder, DevSpaceSSHPublicKeyFile)
}

var keyLock sync.Mutex

func generatePrivateKey() (*ecdsa.PrivateKey, string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}

	// generate and write private key as PEM
	var privateKeyBuf strings.Builder
	b, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, "", err
	}
	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	}
	if err := pem.Encode(&privateKeyBuf, privateKeyPEM); err != nil {
		return nil, "", err
	}

	return privateKey, privateKeyBuf.String(), nil
}

func MakeHostKey() (string, error) {
	_, privKeyStr, err := generatePrivateKey()
	if err != nil {
		return "", err
	}
	return privKeyStr, nil
}

func MakeSSHKeyPair() (string, string, error) {
	privateKey, privKeyStr, err := generatePrivateKey()
	if err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))
	return pubKeyBuf.String(), privKeyStr, nil
}

func getHostKey() (string, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	_, err := os.Stat(DevSpaceSSHFolder)
	if err != nil {
		err = os.MkdirAll(DevSpaceSSHFolder, 0755)
		if err != nil {
			return "", err
		}
	}

	// check if key pair exists
	_, err = os.Stat(DevSpaceSSHHostKeyFile)
	if err != nil {
		privateKey, err := MakeHostKey()
		if err != nil {
			return "", errors.Wrap(err, "generate host key")
		}

		err = os.WriteFile(DevSpaceSSHHostKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write host key")
		}
	}

	// read public key
	out, err := os.ReadFile(DevSpaceSSHHostKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read host ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}

func getPublicKey() (string, error) {
	keyLock.Lock()
	defer keyLock.Unlock()

	_, err := os.Stat(DevSpaceSSHFolder)
	if err != nil {
		err = os.MkdirAll(DevSpaceSSHFolder, 0755)
		if err != nil {
			return "", err
		}
	}

	// check if key pair exists
	_, err = os.Stat(DevSpaceSSHPrivateKeyFile)
	if err != nil {
		pubKey, privateKey, err := MakeSSHKeyPair()
		if err != nil {
			return "", errors.Wrap(err, "generate key pair")
		}

		err = os.WriteFile(DevSpaceSSHPublicKeyFile, []byte(pubKey), 0644)
		if err != nil {
			return "", errors.Wrap(err, "write public ssh key")
		}

		err = os.WriteFile(DevSpaceSSHPrivateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write private ssh key")
		}
	}

	// read public key
	out, err := os.ReadFile(DevSpaceSSHPublicKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read public ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}
