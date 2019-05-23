package sync

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestSymlink(t *testing.T) {
	stat, err := os.Stat("/Users/fabiankramm/Programmieren/go-workspace/src/github.com/devspace-cloud/devspace/sync/def")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(stat.IsDir())

	files, err := ioutil.ReadDir("/Users/fabiankramm/Programmieren/go-workspace/src/github.com/devspace-cloud/devspace/sync/def")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
	}

	t.Fatal("Test")
}
