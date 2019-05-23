package server

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
)

func TestDownstreamServer(t *testing.T) {
	fromDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	toDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(fromDir)
	defer os.RemoveAll(toDir)

	err = createFiles(fromDir, fileStructure)
	if err != nil {
		t.Fatal(err)
	}

	err = createFiles(toDir, overwriteFileStructure)
	if err != nil {
		t.Fatal(err)
	}

	// Upload tar with client
	clientReader, clientWriter := io.Pipe()
	serverReader, serverWriter := io.Pipe()

	go func() {
		err := StartDownstreamServer(fromDir, serverReader, clientWriter)
		if err != nil {
			t.Fatal(err)
		}
	}()

	conn, err := util.NewClientConnection(clientReader, serverWriter)
	if err != nil {
		t.Fatal(err)
	}

	client := remote.NewDownstreamClient(conn)
	changesClient, err := client.Changes(context.Background(), &remote.Excluded{
		Paths: []string{"emptydir"},
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Created downstream server & client")

	changes, err := getAllChanges(changesClient)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Got all changes")

	// Download all the changed files
	downloadClient, err := client.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range changes {
		if change.ChangeType != remote.ChangeType_CHANGE {
			t.Fatal("Expected only changes with type change")
		}

		err := downloadClient.Send(&remote.Path{
			Path: change.Path,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	err = downloadClient.CloseSend()
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Sent all download paths")

	// Tar pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	for {
		chunk, err := downloadClient.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}

		_, err = w.Write(chunk.Content)
		if err != nil {
			t.Fatal(err)
		}
	}

	w.Close()
	log.Println("Downloaded complete file")

	err = untarAll(r, toDir, "")
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Untared the downloaded file")

	delete(fileStructure.Children, "emptydir")
	err = compareFiles(toDir, fileStructure)
	if err != nil {
		t.Fatal(err)
	}

	// Check for changes again
	changesClient, err = client.Changes(context.Background(), &remote.Excluded{
		Paths: []string{"emptydir"},
	})
	if err != nil {
		t.Fatal(err)
	}

	changes, err = getAllChanges(changesClient)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) > 0 {
		t.Fatal("Expected 0 changes")
	}

	// Change file
	err = ioutil.WriteFile(filepath.Join(fromDir, "test.txt"), []byte("overidden"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Remove(filepath.Join(fromDir, "emptydir2"))
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Get changes again")

	// Check for changes again
	changesClient, err = client.Changes(context.Background(), &remote.Excluded{
		Paths: []string{"emptydir"},
	})
	if err != nil {
		t.Fatal(err)
	}

	changes, err = getAllChanges(changesClient)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("Expected 2 changes, got %d changes", len(changes))
	}
}

func getAllChanges(changesClient remote.Downstream_ChangesClient) ([]*remote.Change, error) {
	changes := make([]*remote.Change, 0, 32)
	for {
		change, err := changesClient.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		changes = append(changes, change)
	}

	return changes, nil
}
