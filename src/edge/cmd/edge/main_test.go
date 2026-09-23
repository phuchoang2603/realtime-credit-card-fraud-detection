package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestEdgeProcessHelper(t *testing.T) {
	if os.Getenv("EDGE_PROCESS_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	main()
	os.Exit(0)
}

func TestProcessReadinessAndSIGTERM(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestEdgeProcessHelper$")
	cmd.Env = append(os.Environ(), "EDGE_PROCESS_HELPER=1", "EDGE_HTTP_ADDR="+address, "EDGE_SHUTDOWN_TIMEOUT=200ms")
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	ready := false
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + address + "/ready")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		cancel()
		<-done
		t.Fatalf("process never ready: %s", logs.String())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SIGTERM exit: %v; %s", err, logs.String())
		}
	case <-time.After(3 * time.Second):
		cancel()
		<-done
		t.Fatalf("SIGTERM did not exit within budget: %s", logs.String())
	}
	if response, err := client.Get("http://" + address + "/ready"); err == nil {
		response.Body.Close()
		t.Fatal("process still accepting requests after shutdown")
	}
}

func TestInvalidConfigExitsBeforeListening(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestEdgeProcessHelper$")
	cmd.Env = append(os.Environ(), "EDGE_PROCESS_HELPER=1", "EDGE_HTTP_ADDR=invalid-address")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("invalid configuration exited successfully: %s", output)
	}
	if !bytes.Contains(output, []byte("invalid edge configuration")) {
		t.Fatalf("missing startup diagnostic: %s", output)
	}
}
