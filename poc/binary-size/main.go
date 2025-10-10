package main

import (
	"fmt"

	// Import all Warren dependencies to measure binary size
	_ "github.com/containerd/containerd"
	_ "github.com/hashicorp/raft"
	_ "github.com/hashicorp/raft-boltdb"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/rs/zerolog"
	_ "github.com/spf13/cobra"
	_ "golang.zx2c4.com/wireguard/wgctrl"
	_ "google.golang.org/grpc"
)

func main() {
	fmt.Println("Warren Binary Size POC")
	fmt.Println("This minimal program imports all major Warren dependencies.")
	fmt.Println("Build and check the binary size with: make build")
}
