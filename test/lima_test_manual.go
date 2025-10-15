//go:build darwin
// +build darwin

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cuemby/warren/pkg/embedded"
)

func main() {
	fmt.Println("=== Warren Lima Integration Test ===")
	fmt.Println()

	// Create temp data directory
	tmpDir := filepath.Join(os.TempDir(), "warren-lima-test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		fmt.Printf("❌ Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("📁 Using data directory: %s\n", tmpDir)
	fmt.Println()

	// Test 1: Start Lima VM
	fmt.Println("Test 1: Starting Lima VM with containerd...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	mgr, err := embedded.EnsureContainerdMacOS(ctx, tmpDir)
	if err != nil {
		fmt.Printf("❌ Failed to start Lima VM: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Lima VM started successfully")
	fmt.Println()

	// Test 2: Check socket path
	fmt.Println("Test 2: Verifying containerd socket...")
	socketPath := mgr.GetSocketPath()
	if socketPath == "" {
		fmt.Println("❌ Socket path is empty")
		os.Exit(1)
	}
	fmt.Printf("✅ Socket path: %s\n", socketPath)
	fmt.Println()

	// Test 3: Wait a bit for VM to fully boot
	fmt.Println("Test 3: Waiting for VM to fully boot (10s)...")
	time.Sleep(10 * time.Second)
	fmt.Println("✅ Wait complete")
	fmt.Println()

	// Test 4: Stop Lima VM
	fmt.Println("Test 4: Stopping Lima VM gracefully...")
	if err := mgr.Stop(); err != nil {
		fmt.Printf("❌ Failed to stop Lima VM: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Lima VM stopped successfully")
	fmt.Println()

	fmt.Println("=== All Tests Passed! ===")
}
