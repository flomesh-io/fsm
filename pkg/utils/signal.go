package utils

import (
	"os"
	"os/signal"
	"syscall"
)

func RegisterExitHandlers(shutdownFuncs ...func()) (stop chan struct{}) {
	stop = make(chan struct{})

	go func() {
		// Block until any signal is received.
		<-stop

		// execute our shutdown functions
		for _, f := range shutdownFuncs {
			f()
		}
	}()

	return stop
}

func RegisterOSExitHandlers(shutdownFuncs ...func()) (stop chan struct{}) {
	var exitSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL}

	stop = make(chan struct{})
	s := make(chan os.Signal, len(exitSignals))
	signal.Notify(s, exitSignals...)

	go func() {
		// Wait for a signal from the OS before dispatching
		// a stop signal to all other goroutines observing this channel.
		<-s
		close(stop)

		// execute our shutdown functions
		for _, f := range shutdownFuncs {
			f()
		}
	}()

	return stop
}
