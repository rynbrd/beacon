package main

import (
	"os"
	"os/signal"
	"syscall"
)

// notifySignals wires up stop signals for Unix.
func notifyOnStop() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	return ch
}
