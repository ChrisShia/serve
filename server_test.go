package serve

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestListenAndServe_ShutsDownGracefully(t *testing.T) {
	t.Run("", func(t *testing.T) {
		mock := &mockServer{logs: make([]string, 0)}

		go func() {
			err := ListenAndServe(mock, 0)
			if err != nil {
				t.Errorf("Wanted graceful shutdown, got error: %s", err.Error())
			}
		}()

		time.Sleep(200 * time.Millisecond)

		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		if err != nil {
			t.Fatalf("failed to send SIGINT: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
		expectedLog := "shutting down server: signal:interrupt"

		actualLog := mock.log()

		if len(actualLog) != 1 {
			t.Errorf("Wanted 1 log entry got %d", len(actualLog))
		}

		if mock.logs[0] != expectedLog {
			t.Errorf("Wanted \"%s\", got \"%s\"", expectedLog, mock.logs[0])
		}
	})
}

type mockServer struct {
	started bool
	stopped bool
	logs    []string
	mu      sync.Mutex
}

func (ms *mockServer) ListenAndServe() error {
	return nil
}

func (ms *mockServer) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (ms *mockServer) Routes() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ms.PrintInfo("Handling request to /", nil)
	})
	return router
}

func (ms *mockServer) LogStartUp() {
}

func (ms *mockServer) LogShutdown() {
}

func (ms *mockServer) PrintInfo(msg string, properties map[string]string) {
	sprintf := fmt.Sprintf("%s: %s", msg, stringify(properties))
	ms.append(sprintf)
}

func (ms *mockServer) append(s string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.logs = append(ms.logs, s)
}

func (ms *mockServer) log() []string {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.logs
}

func stringify(p map[string]string) string {
	builder := strings.Builder{}
	for k, v := range p {
		builder.WriteString(fmt.Sprintf("%s:%s,", k, v))
	}
	s := builder.String()
	s = strings.TrimRight(s, ",")
	return s
}

func (ms *mockServer) Write(p []byte) (n int, err error) {
	ms.append(string(p))
	return len(p), nil
}
