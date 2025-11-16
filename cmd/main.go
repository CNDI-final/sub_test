package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"Action_test/sub_test"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := subtest.StartSystem1(ctx); err != nil {
		log.Fatal(err)
	}
}
