package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"overnode/common/pkg/config"
	"overnode/common/pkg/logging"
	"overnode/site/internal/app"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var configuration app.Config
	if err := config.Load(args, stdout, &configuration, config.Options{
		Command:     "oversite",
		Description: "OverNode static site server",
		Fields:      app.Schema(),
	}); err != nil {
		if errors.Is(err, config.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "oversite: %v\n", err)
		return 1
	}

	logger, closeLogger, err := logging.New(configuration.Logging, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "oversite: %v\n", err)
		return 1
	}
	defer func() {
		if err := closeLogger(); err != nil {
			fmt.Fprintf(stderr, "oversite: close log: %v\n", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, configuration.HTTPD, logger); err != nil {
		logger.Error("site server failed", "error", err)
		return 1
	}
	return 0
}
