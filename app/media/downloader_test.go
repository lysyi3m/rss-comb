package media

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "yt-dlp.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return "sh " + path
}

func TestGetVideoInfoPassesConfiguredArgs(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	cmd := writeScript(t, `echo "$@" > `+argsFile+`
echo '{"id":"v","live_status":"is_live","duration":12.5}'`)

	info, err := GetVideoInfo(context.Background(), cmd, "--cookies /tmp/c.txt", "https://example.com/v")
	if err != nil {
		t.Fatal(err)
	}
	if info.LiveStatus != "is_live" || info.Duration != 12 {
		t.Errorf("info = %+v", info)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.TrimSpace(string(data))
	if !strings.HasPrefix(args, "--cookies /tmp/c.txt --simulate") {
		t.Errorf("args %q do not put configured args before --simulate", args)
	}
	if !strings.HasSuffix(args, "https://example.com/v") {
		t.Errorf("args %q do not end with the URL", args)
	}
}

func TestGetVideoInfoIgnoresOtherOutputLines(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{"print before JSON", `echo "A title"
echo '{"id":"v","live_status":"is_live"}'`},
		{"print after JSON", `echo '{"id":"v","live_status":"is_live"}'
echo "A title"`},
		{"empty object after JSON", `echo '{"id":"v","live_status":"is_live"}'
echo '{}'`},
		{"null after JSON", `echo '{"id":"v","live_status":"is_live"}'
echo 'null'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := GetVideoInfo(context.Background(), writeScript(t, tt.script), "", "https://example.com/v")
			if err != nil {
				t.Fatal(err)
			}
			if info.LiveStatus != "is_live" {
				t.Errorf("live status = %q", info.LiveStatus)
			}
		})
	}
}

func TestGetVideoInfoRejectsOutputWithoutVideoRecord(t *testing.T) {
	cmd := writeScript(t, `echo "A title"
echo '{}'`)

	if _, err := GetVideoInfo(context.Background(), cmd, "", "https://example.com/v"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCancelKillsDescendantsHoldingOutputPipes(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	release := filepath.Join(dir, "release")
	survived := filepath.Join(dir, "survived")
	cmd := writeScript(t, `(echo x > `+ready+`
i=0
while [ ! -e `+release+` ] && [ $i -lt 600 ]; do sleep 0.05; i=$((i+1)); done
echo x > `+survived+`) &
wait`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := GetVideoInfo(ctx, cmd, "", "https://example.com/v")
		done <- err
	}()

	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not start")
		}
	}

	cancel()
	start := time.Now()
	if err := <-done; err == nil {
		t.Fatal("expected an error after cancellation")
	}
	if elapsed := time.Since(start); elapsed >= subprocessWaitDelay {
		t.Errorf("returned after %s, want the kill to close the pipes", elapsed)
	}

	if err := os.WriteFile(release, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(survived); err == nil {
		t.Error("descendant survived cancellation")
	}
}
