package main

import (
	"os"

	"kii.com/cmd/cli"
)

func main() {
	err := cli.Run()
	if err != nil {
		os.Exit(1)
	}
}
