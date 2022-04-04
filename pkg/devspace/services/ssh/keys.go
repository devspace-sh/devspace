package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	DevSpaceSSHFolder         = "ssh"
	DevSpaceSSHHostKeyFile    = "id_devspace_host_rsa"
	DevSpaceSSHPrivateKeyFile = "id_devspace_rsa"
	DevSpaceSSHPublicKeyFile  = "id_devspace_rsa.pub"
)

func init() {
	homeDir, _ := homedir.Dir()
	DevSpaceSSHFolder = filepath.Join(homeDir, constants.DefaultHomeDevSpaceFolder, DevSpaceSSHFolder)
	DevSpaceSSHHostKeyFile = filepath.Join(DevSpaceSSHFolder, DevSpaceSSHHostKeyFile)
	DevSpaceSSHPrivateKeyFile = filepath.Join(DevSpaceSSHFolder, DevSpaceSSHPrivateKeyFile)
	DevSpaceSSHPublicKeyFile = filepath.Join(DevSpaceSSHFolder, DevSpaceSSHPublicKeyFile)
}

var keyLock sync.Mutex

func MakeHostKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	// generate and write private key as PEM
	var privKeyBuf strings.Builder
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", err
	}

	return privKeyBuf.String(), nil
}

func MakeSSHKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// generate and write private key as PEM
	var privKeyBuf strings.Builder
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	var pubKeyBuf strings.Builder
	pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))
	return pubKeyBuf.String(), privKeyBuf.String(), nil
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

		err = ioutil.WriteFile(DevSpaceSSHHostKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write host key")
		}
	}

	// read public key
	out, err := ioutil.ReadFile(DevSpaceSSHHostKeyFile)
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

		err = ioutil.WriteFile(DevSpaceSSHPublicKeyFile, []byte(pubKey), 0644)
		if err != nil {
			return "", errors.Wrap(err, "write public ssh key")
		}

		err = ioutil.WriteFile(DevSpaceSSHPrivateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			return "", errors.Wrap(err, "write private ssh key")
		}
	}

	// read public key
	out, err := ioutil.ReadFile(DevSpaceSSHPublicKeyFile)
	if err != nil {
		return "", errors.Wrap(err, "read public ssh key")
	}

	return base64.StdEncoding.EncodeToString(out), nil
}
