package serve

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Interface interface {
	routable
	loggable
}

type routable interface {
	Routes() http.Handler
}

type loggable interface {
	LogStartUp()
	LogShutdown()
	PrintInfo(msg string, properties map[string]string)
	Write(p []byte) (n int, err error)
}

type Timeouts struct {
	IdleTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

var defaultTimeouts = Timeouts{
	IdleTimeout:     time.Minute,
	ReadTimeout:     10 * time.Second,
	WriteTimeout:    30 * time.Second,
	ShutdownTimeout: 5 * time.Second,
}

func ListenAndServe(servable Interface, port int) error {
	return ListenAndServeWithTimeouts(servable, port, Timeouts{ShutdownTimeout: 5 * time.Second})
}

func ListenAndServeDefaultTimeouts(serve Interface, port int) error {
	return ListenAndServeWithTimeouts(serve, port, defaultTimeouts)
}

func newS(serve Interface, port int, tOut Timeouts) *S {
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      serve.Routes(),
		IdleTimeout:  tOut.IdleTimeout,
		ReadTimeout:  tOut.ReadTimeout,
		WriteTimeout: tOut.WriteTimeout,
		ErrorLog:     log.New(serve, "", 0),
	}

	return &S{
		httpServer: httpServer,
		shutdown:   make(chan error),
		logger:     serve,
		done:       make(chan struct{}),
	}
}

type S struct {
	httpServer      *http.Server
	shutdown        chan error
	logger          loggable
	done            chan struct{}
	shutdownTimeout time.Duration
}

func ListenAndServeWithTimeouts(servable Interface, port int, tOut Timeouts) error {
	s := newS(servable, port, tOut)

	go s.listenForSignals(syscall.SIGINT, syscall.SIGTERM)

	servable.LogStartUp()

	err := s.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		close(s.done)
		return err
	}

	err = <-s.shutdown
	if err != nil {
		return err
	}

	servable.LogShutdown()

	return nil
}

func (s *S) listenForSignals(sigs ...os.Signal) {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, sigs...)
	defer signal.Stop(quit)

	select {
	case sig := <-quit:
		s.logger.PrintInfo("shutting down server", map[string]string{
			"signal": sig.String(),
		})
	case <-s.done:
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	s.shutdown <- s.httpServer.Shutdown(ctx)
}
