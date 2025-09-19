package serve

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestListenForSignalsFromChan(t *testing.T) {
	mock := &mockServer{logs: make([]string, 0)}
	osSig := make(chan os.Signal)
	server := S{
		las:             mock,
		logger:          mock,
		done:            make(chan struct{}),
		shutdown:        make(chan error),
		shutdownTimeout: time.Second,
	}

	go server.listenFromSignalsFromChan(osSig)

	sigint := syscall.SIGINT
	osSig <- sigint

	err := <-server.shutdown
	if err != nil {
		t.Errorf("Wanted graceful shutdown, got error: %s", err.Error())
	}

	if len(mock.log()) != 1 {
		t.Errorf("Wanted 1 log entry got %d", len(mock.logs))
	} else {
		expectedLog := fmt.Sprintf("shutting down server: signal:%s", sigint.String())
		actualLog := mock.log()
		if expectedLog != actualLog[0] {
			t.Errorf("Wanted \"%s\", got \"%s\"", expectedLog, actualLog[0])
		}
	}
}
