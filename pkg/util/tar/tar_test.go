package tar

import (
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	
	"gotest.tools/assert"
)

func TestExtractSingleFileTarGz(t *testing.T) {
	//This is a single file "hello.txt" with the content "hello world" zipped and converted to .tar.gz
	err := ExtractSingleFileTarGz("test.tar.gz", "hello.txt", "hello.txt")
	if err != nil {
		t.Fatalf("Error extracting single file: %v", err)
	}
	defer os.RemoveAll("hello.txt")

	helloFile, err := fsutil.ReadFile("hello.txt", -1)
	if err != nil {
		t.Fatalf("Error getting outputFile of ExtractSingleFileTarGz. It might not be created: %v", err)
	}
	assert.Equal(t, "hello world", string(helloFile), "extracted file has wrong content")
}

func TestExtractSingleFileToStringTarGz(t *testing.T) {
	//This is a single file "hello.txt" with the content "hello world" zipped and converted to .tar.gz
	content, err := ExtractSingleFileToStringTarGz("test.tar.gz", "hello.txt")
	if err != nil {
		t.Fatalf("Error extracting single file: %v", err)
	}
	
	assert.Equal(t, "hello world", content, "Extracted file has wrong content")
}
