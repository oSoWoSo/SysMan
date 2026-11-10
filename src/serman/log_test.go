package serman

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestRunitBackendOpenLogsStreamsServiceOutputAndStops(t *testing.T) {
	binDir := t.TempDir()
	tool := filepath.Join(binDir, "svlogtail")
	script := "#!/bin/sh\nprintf 'service=%s\\n' \"$1\"\nexec /bin/sleep 30\n"
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	stream, err := (&RunitBackend{}).OpenLogs(Service{Name: "svc;not-a-command"})
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stream).ReadString('\n')
	if err != nil {
		_ = stream.Close()
		t.Fatalf("read log stream: %v", err)
	}
	if line != "service=svc;not-a-command\n" {
		_ = stream.Close()
		t.Fatalf("unexpected service argument output %q", line)
	}

	closed := make(chan struct{})
	go func() {
		_ = stream.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("closing the log stream did not stop svlogtail")
	}
}

type testLogBackend struct {
	*RunitBackend
	output string
}

func (b *testLogBackend) OpenLogs(Service) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(b.output)), nil
}

func TestTUILogShortcutDisplaysStreamAndCloses(t *testing.T) {
	m := newTestModel([]Service{{Name: "demo", Enabled: true}})
	m.backend = &testLogBackend{RunitBackend: NewRunitBackend("", ""), output: "first line\nsecond line\n"}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	m = updated.(tuiModel)
	if !m.showLogs || cmd == nil {
		t.Fatal("v should open the selected service log view")
	}

	started := cmd().(tuiLogsStartedMsg)
	updated, cmd = m.Update(started)
	m = updated.(tuiModel)
	for cmd != nil {
		updated, cmd = m.Update(cmd())
		m = updated.(tuiModel)
	}
	if !strings.Contains(m.render(), "first line\nsecond line") {
		t.Fatalf("log view omitted streamed output:\n%s", m.render())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if updated.(tuiModel).showLogs {
		t.Fatal("Esc should close the log view")
	}
}
