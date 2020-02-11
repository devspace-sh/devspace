package encryption

import (
	"testing"

	"gotest.tools/assert"
)

func TestPadKey(t *testing.T) {
	assert.Equal(t, "12345678901234567890123456789012", string(PadKey([]byte("12345678901234567890123456789012"))), "PadKey of length 32 isn't equal to original key")
	assert.Equal(t, "12345678901234567890123456789012", string(PadKey([]byte("123456789012345678901234567890123"))), "PadKey of length 33 isn't a shortened version of original key")
	assert.Equal(t, "1234567890123456789012345678901 ", string(PadKey([]byte("1234567890123456789012345678901"))), "PadKey of length 31 isn't an extended version of original key")
}

func TestEncryptDecryptAES(t *testing.T) {
	encrypted, err := EncryptAES([]byte("12345678901234567890123456789012"), []byte("Hello World"))
	assert.NilError(t, err, "Error encrypting hello world")
	decrypted, err := DecryptAES([]byte("12345678901234567890123456789012"), encrypted)
	assert.NilError(t, err, "Error decrypting hello world")
	assert.Equal(t, "Hello World", string(decrypted), "\"Hello World\" encrypted and decrypted is not \"Hello World\" anymore")
}
