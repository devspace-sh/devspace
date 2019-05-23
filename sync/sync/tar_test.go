package sync

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

var pool = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789%(&)°=?!§ _:$%&/()"

// Generate a random string of A-Z chars with len = l
func random(l int) []byte {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = pool[rand.Intn(len(pool))]
	}
	return bytes
}

type testFile struct {
	Data     []byte
	Children map[string]testFile
}

var fileStructure = testFile{
	Children: map[string]testFile{
		"test.txt": testFile{
			Data: random(10),
		},
		"emptydir": testFile{
			Children: map[string]testFile{},
		},
		"dir1": testFile{
			Children: map[string]testFile{
				"dir1-child": testFile{
					Children: map[string]testFile{
						"test": testFile{
							Data: random(100),
						},
						"test-123": testFile{
							Data: []byte{},
						},
					},
				},
			},
		},
	},
}

func createFiles(dir string, file testFile) error {
	for name, child := range file.Children {
		if child.Children == nil {
			err := createFiles(filepath.Join(dir, name), child)
			if err != nil {
				return err
			}
		} else {
			err := ioutil.WriteFile(filepath.Join(dir, name), child.Data, 0666)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func TestTar(t *testing.T) {
	fromDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	toDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tempDir)
	defer os.RemoveAll(fromDir)
	defer os.RemoveAll(toDir)

	err = createFiles(fromDir, fileStructure)
	if err != nil {
		t.Fatal(err)
	}

	// writeTar(fromDir)
}

func TestPipe(t *testing.T) {
	read, write, err := os.Pipe()

	for i := 0; i < 5; i++ {
		_, err := write.Write([]byte("0123456789"))
		if err != nil {
			t.Fatalf("Write %v", err)
		}
	}

	write.Close()

	buffer, err := ioutil.ReadAll(read)
	if err != nil {
		t.Fatalf("Read %v", err)
	}

	t.Fatal(string(buffer))
}
