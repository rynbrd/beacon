package docker

import (
	"log"
	"os"
)

var logger *log.Logger = log.New(os.Stdout, "", 0)
