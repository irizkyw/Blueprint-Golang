package shutdown

import (
	"os"
	"os/signal"
	"syscall"
)

func WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	defer close(quit)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
