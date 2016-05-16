package beacon

import (
	"log"
	"os"
)

// Logger is used by the package to log events. It may be set to the
// application logger to change the destination.
var Logger = log.New(os.Stdout, "", 0)
