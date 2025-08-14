//go:build !windows
// +build !windows

package server

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/util"
)

func TestDownstreamServer(t *testing.T) {
	fromDir := t.TempDir()
	toDir := t.TempDir()

	err := createFiles(fromDir, fileStructure)
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
		err := StartDownstreamServer(serverReader, clientWriter, &DownstreamOptions{
			RemotePath:   fromDir,
			ExcludePaths: []string{"emptydir"},
			ExitOnClose:  false,
			Polling:      true,
		})
		if err != nil {
			panic(err)
		}
	}()

	conn, err := util.NewClientConnection(clientReader, serverWriter)
	if err != nil {
		t.Fatal(err)
	}

	// Count changes
	client := remote.NewDownstreamClient(conn)
	amount, err := client.ChangesCount(context.Background(), &remote.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if amount.Amount == 0 {
		t.Fatalf("Unexpected change amount, expected >0, got %d", amount.Amount)
	}

	changesClient, err := client.Changes(context.Background(), &remote.Empty{})
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

		err := downloadClient.Send(&remote.Paths{
			Paths: []string{
				change.Path,
			},
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

	err = untarAll(r, &UpstreamOptions{UploadPath: toDir})
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
	changesClient, err = client.Changes(context.Background(), &remote.Empty{})
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
	err = os.WriteFile(filepath.Join(fromDir, "test.txt"), []byte("overidden"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Remove(filepath.Join(fromDir, "emptydir2"))
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Get changes again")

	// Check for changes again
	changesClient, err = client.Changes(context.Background(), &remote.Empty{})
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
		changeChunk, err := changesClient.Recv()
		if changeChunk != nil {
			changes = append(changes, changeChunk.Changes...)
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return changes, nil
}
