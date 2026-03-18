package main

import (
	"github.com/potacast/potacast/internal/cli"
)

var version = "dev"

func main() {
	cli.Execute(version)
}
