package serman

import (
	"io"
	"os/exec"
	"strings"
	"sync"
)

// LogBackend provides a live log stream for service-manager backends that can
// expose one. The stream is closed when the viewer is dismissed.
type LogBackend interface {
	OpenLogs(Service) (io.ReadCloser, error)
}

type serviceLogStream struct {
	reader *io.PipeReader
	cmd    *exec.Cmd
	done   chan struct{}
	once   sync.Once
}

func (s *serviceLogStream) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *serviceLogStream) Close() error {
	s.once.Do(func() {
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		_ = s.reader.Close()
		<-s.done
	})
	return nil
}

// OpenLogs starts svlogtail for a service and streams its existing and future
// log output. The caller must close the returned stream to stop the process.
func (b *RunitBackend) OpenLogs(svc Service) (io.ReadCloser, error) {
	cmd := exec.Command("svlogtail", svc.Name) //nolint:gosec // service name is passed as an argument, never through a shell
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return nil, err
	}
	stream := &serviceLogStream{reader: reader, cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		_ = writer.Close()
		close(stream.done)
	}()
	return stream, nil
}

const maxServiceLogBytes = 256 * 1024

func appendServiceLog(current, chunk string) string {
	current += chunk
	if len(current) <= maxServiceLogBytes {
		return current
	}
	cut := len(current) - maxServiceLogBytes
	if nextLine := strings.IndexByte(current[cut:], '\n'); nextLine >= 0 {
		cut += nextLine + 1
	}
	return current[cut:]
}
