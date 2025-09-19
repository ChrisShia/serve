package serve

import (
	"context"
	"os"
	"os/signal"
)

func (s *S) listenForSignals(sigs ...os.Signal) {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, sigs...)
	defer signal.Stop(quit)

	s.listenFromSignalsFromChan(quit)
}

func (s *S) listenFromSignalsFromChan(quit chan os.Signal) {
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

	s.shutdown <- s.las.Shutdown(ctx)
}
