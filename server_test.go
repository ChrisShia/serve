package serve

import (
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
		mock := &mockS{logs: make([]string, 0)}

		go func() {
			err := ListenAndServe(mock, 8080)
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

func TestListenAndServe_ShutdownTimeout(t *testing.T) {

}

type mockS struct {
	started bool
	stopped bool
	logs    []string
	mu      sync.Mutex
}

func (ms *mockS) Routes() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ms.PrintInfo("Handling request to /", nil)
	})
	return router
}

func (ms *mockS) LogStartUp() {
}

func (ms *mockS) LogShutdown() {
}

func (ms *mockS) PrintInfo(msg string, properties map[string]string) {
	sprintf := fmt.Sprintf("%s: %s", msg, stringify(properties))
	ms.append(sprintf)
}

func (ms *mockS) append(s string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.logs = append(ms.logs, s)
}

func (ms *mockS) log() []string {
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

func (ms *mockS) Write(p []byte) (n int, err error) {
	ms.append(string(p))
	return len(p), nil
}
