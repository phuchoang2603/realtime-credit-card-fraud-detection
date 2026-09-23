package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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
