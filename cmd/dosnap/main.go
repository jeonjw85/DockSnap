package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := Execute(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitStatus(err))
	}
}
