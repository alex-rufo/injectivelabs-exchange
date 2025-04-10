package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "exchange"}

func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	runOnSignal(cancel, syscall.SIGTERM, syscall.SIGINT)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runOnSignal(f func(), sig ...os.Signal) {
	stop := make(chan os.Signal, 5)
	signal.Notify(stop, sig...)
	go func() {
		<-stop
		f()
	}()
}
