package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
)

func TestMediaTransferOutlivesServerWriteTimeout(t *testing.T) {
	const size = 64 << 20 // larger than the loopback socket buffers

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.mp3"), make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewUnstartedServer(NewServer(&Handler{}, &cfg.Cfg{MediaDir: dir}))
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/a.mp3")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	time.Sleep(300 * time.Millisecond)

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		t.Fatalf("transfer failed after %d bytes: %v", n, err)
	}
	if n != size {
		t.Fatalf("read %d bytes, want %d", n, size)
	}
}
