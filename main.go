package main

import (
	"os"

	"github.com/mrlyc/heracles/cmd"
)

func main() {
	os.Setenv("OTEL_IGNORE_ERROR", "1")
	cmd.Execute()
}
