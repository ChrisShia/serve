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
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

var defaultTimeouts = Timeouts{
	IdleTimeout:  time.Minute,
	ReadTimeout:  10 * time.Second,
	WriteTimeout: 30 * time.Second,
}

func ListenAndServe(serve Interface, port int) error {
	return ListenAndServeWithTimeouts(serve, port, defaultTimeouts)
}

func newServer(serve Interface, port int, tOut Timeouts) *S {
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
	}
}

type S struct {
	httpServer *http.Server
	shutdown   chan error
}

func ListenAndServeWithTimeouts(servable Interface, port int, tOut Timeouts) error {
	shutdownError := make(chan error)

	s := newServer(servable, port, tOut)

	go s.listenForSignals(shutdownError, syscall.SIGINT, syscall.SIGTERM)

	servable.LogStartUp()

	err := s.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	servable.LogShutdown()

	return nil
}

var InfoLog interface {
	PrintInfo(msg string, properties map[string]string)
}

func (s *S) listenForSignals(shutdown chan<- error, sigs ...os.Signal) {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, sigs...)

	sig := <-quit

	InfoLog.PrintInfo("shutting down server", map[string]string{
		"signal": sig.String(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdown <- s.httpServer.Shutdown(ctx)
}
