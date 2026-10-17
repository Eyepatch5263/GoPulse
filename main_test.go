package main

import (
	"context"
	"testing"
	"time"
)

func TestRunMonitorCancellation(t *testing.T) {
	// created a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	res := make(chan MonitorResult)

	done := make(chan struct{})

	// when the goroutine finishes, it closes done
	// the done channel isn't closes after 50 ms, it closes only when runMonitor returns
	go func() {
		defer close(done)
		runMonitor(ctx, res, 50*time.Millisecond)
	}()

	// we want to give the monitoring goroutine time to reach the channel-send operation and potentially block there before we cancel it
	time.Sleep(200 * time.Millisecond)

	// the ctx is marked cancelled -> ctx.done() becomes ready
	cancel()

	select {
	// A receive from close channel can proceed immediately
	case <-done:

	case <-time.After(2 * time.Millisecond):
		t.Fatal("Monitor didn't stop after cancellation")
	}

	if _, ok := <-res; ok {
		t.Fatal("Unexpected result after cancellation")
	}
}
