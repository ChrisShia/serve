package serve

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"
)

type routeLogger interface {
	router
	logger
}

type router interface {
	Routes() http.Handler
}

type logger interface {
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

func ListenAndServe(servable routeLogger, port int) error {
	return ListenAndServeWithTimeouts(servable, port, Timeouts{ShutdownTimeout: 5 * time.Second})
}

func ListenAndServeDefaultTimeouts(serve routeLogger, port int) error {
	return ListenAndServeWithTimeouts(serve, port, defaultTimeouts)
}

func newS(serve routeLogger, port int, tOut Timeouts) *S {
	return &S{
		las:             newHTTPServer(serve, port, tOut),
		shutdown:        make(chan error),
		logger:          serve,
		done:            make(chan struct{}),
		shutdownTimeout: tOut.ShutdownTimeout,
	}
}

func newHTTPServer(serve routeLogger, port int, tOut Timeouts) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      serve.Routes(),
		IdleTimeout:  tOut.IdleTimeout,
		ReadTimeout:  tOut.ReadTimeout,
		WriteTimeout: tOut.WriteTimeout,
		ErrorLog:     log.New(serve, "", 0),
	}
}

type S struct {
	las             listenAndServer
	logger          logger
	shutdown        chan error
	done            chan struct{}
	shutdownTimeout time.Duration
}

type listenAndServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

func ListenAndServeWithTimeouts(servable routeLogger, port int, tOut Timeouts) error {
	s := newS(servable, port, tOut)

	go s.listenForSignals(syscall.SIGINT, syscall.SIGTERM)

	servable.LogStartUp()

	err := s.las.ListenAndServe()
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
