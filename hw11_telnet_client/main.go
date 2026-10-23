package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

const defaultTimeout = 10 * time.Second

var errUsage = errors.New("usage: go-telnet [--timeout=10s] host port")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "..."+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var timeout time.Duration
	flag.DurationVar(&timeout, "timeout", defaultTimeout, "connection timeout")
	flag.Parse()

	if flag.NArg() != 2 {
		return errUsage
	}

	address := flag.Arg(0) + ":" + flag.Arg(1)

	client := NewTelnetClient(address, timeout, os.Stdin, os.Stdout)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	fmt.Fprintf(os.Stderr, "...Connected to %s\n", address)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		defer stop()

		if err := client.Send(); err != nil {
			fmt.Fprintln(os.Stderr, "...Connection was closed by peer:", err)
			return
		}
		fmt.Fprintln(os.Stderr, "...EOF")
	}()

	go func() {
		defer stop()

		err := client.Receive()
		switch {
		case errors.Is(err, net.ErrClosed):
			return
		case err != nil:
			fmt.Fprintln(os.Stderr, "...Receive error:", err)
			return
		}
		fmt.Fprintln(os.Stderr, "...Connection was closed by peer")
	}()

	<-ctx.Done()

	return nil
}
