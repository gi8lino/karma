package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gi8lino/karma/internal/app"
)

var (
	Version = "dev"
	Commit  = "none"
)

func main() {
	if err := app.Run(context.Background(), Version, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
