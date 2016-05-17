package debug

import (
	"bytes"
	"fmt"
	"github.com/BlueDragonX/beacon/beacon"
)

// Printer is anything that implements the standard Print function.
type Printer interface {
	Print(v ...interface{})
}

// New creates a backend that writes events to the given Printer.
func New(pr Printer) beacon.Backend {
	return &debug{
		pr: pr,
	}
}

// debug is used to output container events for debugging.
type debug struct {
	pr Printer
}

// ProcessEvent formats the event and writes it to the debugger.
func (d *debug) ProcessEvent(event *beacon.Event) error {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "event: svc=%s id=%s action=%s", event.Container.Service, event.Container.ID, event.Action)

	first := true
	for key, value := range event.Container.Labels {
		if first {
			fmt.Fprint(buf, " labels=")
			first = false
		} else {
			fmt.Fprint(buf, ",")
		}
		fmt.Fprintf(buf, "%s:%s", key, value)
	}

	for n, binding := range event.Container.Bindings {
		if n == 0 {
			fmt.Fprint(buf, " ports=")
		} else {
			fmt.Fprint(buf, ",")
		}
		fmt.Fprintf(buf, "%s:%d->%d/%s", binding.HostIP, binding.HostPort, binding.ContainerPort, binding.Protocol)
	}
	fmt.Fprint(buf, "\n")

	d.pr.Print(buf.String())
	return nil
}

// close is a noop for Debug.
func (d *debug) Close() error {
	return nil
}
