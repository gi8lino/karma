package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/gi8lino/karma/internal/app"
)

var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := app.Run(ctx, Version, os.Args[1:], os.Stdout); err != nil {
		if !errors.Is(err, app.ErrCheckFailed) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
