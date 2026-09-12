package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"overnode/common/pkg/config"
	"overnode/common/pkg/logging"
	"overnode/gate/internal/app"
	"syscall"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	var c app.Config
	if err := config.Load(args, stdout, &c, config.Options{Command: "overgate", Description: "OverNode HTTP forward proxy", Fields: app.Schema()}); err != nil {
		if errors.Is(err, config.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "overgate: %v\n", err)
		return 1
	}
	logger, closeLogger, err := logging.New(c.Logging, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "overgate: %v\n", err)
		return 1
	}
	defer func() {
		if err := closeLogger(); err != nil {
			fmt.Fprintf(stderr, "overgate: close log: %v\n", err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, c.ProxyF, logger); err != nil {
		logger.Error("proxy failed", "error", err)
		return 1
	}
	return 0
}
