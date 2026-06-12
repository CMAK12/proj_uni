package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRun_BadDSN(t *testing.T) {
	err := run(context.Background(), "127.0.0.1:0", "/nonexistent-dir-xyz/db.sqlite", testLogger())
	if err == nil {
		t.Fatal("expected error opening database in nonexistent directory")
	}
}

func TestRun_GracefulShutdown(t *testing.T) {
	// Grab a free port, then release it for run to bind.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, addr, ":memory:", testLogger()) }()

	// Wait until the server accepts connections.
	deadline := time.Now().Add(3 * time.Second)
	for {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("server did not start in time")
		}
		resp, err := http.Get("http://" + addr + "/")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel() // trigger graceful shutdown
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error on graceful shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return after context cancel")
	}
}
