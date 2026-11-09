package sync

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/server"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/log"
)

func TestInitialSyncRemotePreference(t *testing.T) {
	for _, testCase := range []struct {
		strategy latest.InitialSyncStrategy
		want     string
	}{
		{latest.InitialSyncStrategyMirrorRemote, "remote"},
		{latest.InitialSyncStrategyPreferRemote, "remote"},
		{latest.InitialSyncStrategyPreferNewest, "local"},
		{latest.InitialSyncStrategyMirrorLocal, "local"},
		{latest.InitialSyncStrategyPreferLocal, "local"},
		{latest.InitialSyncStrategyKeepAll, "local"},
		{latest.InitialSyncStrategyDisabled, "local"},
	} {
		t.Run(string(testCase.strategy), func(t *testing.T) {
			local, remotePath := t.TempDir(), t.TempDir()
			localFile := filepath.Join(local, "conflict.txt")
			remoteFile := filepath.Join(remotePath, "conflict.txt")
			writeFile := func(filename, content string, modified time.Time) {
				t.Helper()
				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(filename, modified, modified); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(localFile, "local", time.Unix(2000000000, 0))
			writeFile(remoteFile, "remote", time.Unix(1700000000, 0))

			client, err := NewSync(context.Background(), local, Options{
				InitialSync:      testCase.strategy,
				UpstreamDisabled: true,
				Log:              log.Discard,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { client.Stop(nil) })
			// Download-only sync still marks the upstream initial-sync state complete.
			client.upstream = &upstream{}

			clientConn, serverConn := net.Pipe()
			t.Cleanup(func() {
				_ = clientConn.Close()
				_ = serverConn.Close()
			})
			go func() {
				if err := server.StartDownstreamServer(serverConn, serverConn, &server.DownstreamOptions{
					RemotePath: remotePath,
					Polling:    true,
				}); err != nil {
					panic(err)
				}
			}()
			if err := client.InitDownstream(clientConn, clientConn); err != nil {
				t.Fatal(err)
			}
			uploadDone := make(chan struct{})
			if err := client.initialSync(uploadDone, nil); err != nil {
				t.Fatal(err)
			}
			<-uploadDone
			content, err := os.ReadFile(localFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != testCase.want {
				t.Errorf("initial sync: got %q, want %q", content, testCase.want)
			}

			// Later downloads must continue to protect a newer, pending local edit.
			writeFile(localFile, "pending local edit", time.Unix(2000000000, 0))
			writeFile(remoteFile, "later remote edit", time.Unix(1700000001, 0))
			if err := client.downstream.applyChanges([]*remote.Change{{
				ChangeType: remote.ChangeType_CHANGE,
				Path:       "/conflict.txt",
			}}, false); err != nil {
				t.Fatal(err)
			}
			content, err = os.ReadFile(localFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "pending local edit" {
				t.Errorf("normal sync overwrote a newer local edit: %q", content)
			}
		})
	}
}
