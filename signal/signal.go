package signal

import (
	"os"
	"os/signal"
	"syscall"
)

var capturedSignals = []os.Signal{syscall.SIGTERM, syscall.SIGINT}

func RegisterSignalHandlers() <-chan struct{} {
	notifyCh := make(chan os.Signal, 2)
	stopCh := make(chan struct{})

	go func() {
		<-notifyCh
		close(stopCh)
		<-notifyCh
		os.Exit(1)
	}()

	signal.Notify(notifyCh, capturedSignals...)

	return stopCh
}
