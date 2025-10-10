package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	interfaceName = "wg0"
	listenPort    = 51820
)

func main() {
	log.Println("=== Warren WireGuard POC ===")
	log.Printf("Interface: %s", interfaceName)
	log.Printf("Listen port: %d", listenPort)
	log.Println()

	// Check root/sudo
	if os.Geteuid() != 0 {
		log.Fatal("❌ This POC requires root/sudo to create network interfaces\n" +
			"Please run: sudo go run .")
	}

	// Create WireGuard client
	log.Println("1. Creating WireGuard client...")
	client, err := wgctrl.New()
	if err != nil {
		log.Fatalf("Failed to create WireGuard client: %v\n"+
			"Note: On macOS, kernel WireGuard is not available.\n"+
			"This POC demonstrates the control API.\n"+
			"Warren will use wireguard-go (userspace) as fallback.", err)
	}
	defer client.Close()
	log.Println("✓ WireGuard client created")

	// Generate keypair
	log.Println("\n2. Generating WireGuard keypair...")
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey()
	log.Printf("✓ Keypair generated")
	log.Printf("  Private key: %s", privateKey.String())
	log.Printf("  Public key:  %s", publicKey.String())

	log.Println("\n=== WireGuard Configuration ===")
	log.Println("To create a 3-node mesh:")
	log.Println()
	log.Println("Node 1 (10.0.0.1):")
	log.Printf("  sudo ip link add %s type wireguard", interfaceName)
	log.Printf("  sudo ip addr add 10.0.0.1/24 dev %s", interfaceName)
	log.Printf("  sudo wg set %s private-key <(echo %s)", interfaceName, "<private-key-1>")
	log.Printf("  sudo wg set %s listen-port %d", interfaceName, listenPort)
	log.Printf("  sudo ip link set %s up", interfaceName)
	log.Println()
	log.Println("Node 2 (10.0.0.2):")
	log.Printf("  sudo ip link add %s type wireguard", interfaceName)
	log.Printf("  sudo ip addr add 10.0.0.2/24 dev %s", interfaceName)
	log.Printf("  sudo wg set %s private-key <(echo %s)", interfaceName, "<private-key-2>")
	log.Printf("  sudo wg set %s listen-port %d", interfaceName, listenPort)
	log.Printf("  sudo wg set %s peer <public-key-1> endpoint <node1-ip>:%d allowed-ips 10.0.0.1/32", interfaceName, listenPort)
	log.Printf("  sudo wg set %s peer <public-key-3> endpoint <node3-ip>:%d allowed-ips 10.0.0.3/32", interfaceName, listenPort)
	log.Printf("  sudo ip link set %s up", interfaceName)
	log.Println()
	log.Println("Node 3 (10.0.0.3):")
	log.Printf("  # Similar to Node 2, with IP 10.0.0.3")
	log.Println()

	log.Println("=== Testing ===")
	log.Println("Once configured, test connectivity:")
	log.Println("  Node 1: ping 10.0.0.2")
	log.Println("  Node 2: ping 10.0.0.3")
	log.Println()

	log.Println("Throughput test (run on both nodes):")
	log.Println("  Node 1: iperf3 -s")
	log.Println("  Node 2: iperf3 -c 10.0.0.1")
	log.Println()

	log.Println("Expected: > 90% of native network speed")
	log.Println()

	// List existing WireGuard devices
	log.Println("=== Current WireGuard Devices ===")
	devices, err := client.Devices()
	if err != nil {
		log.Printf("Failed to list devices: %v", err)
	} else if len(devices) == 0 {
		log.Println("No WireGuard interfaces found")
	} else {
		for _, device := range devices {
			log.Printf("Device: %s", device.Name)
			log.Printf("  Public key: %s", device.PublicKey.String())
			log.Printf("  Listen port: %d", device.ListenPort)
			log.Printf("  Peers: %d", len(device.Peers))
			for i, peer := range device.Peers {
				log.Printf("    Peer %d:", i+1)
				log.Printf("      Public key: %s", peer.PublicKey.String())
				log.Printf("      Endpoint: %s", peer.Endpoint)
				log.Printf("      Allowed IPs: %v", peer.AllowedIPs)
			}
		}
	}

	log.Println("\n✅ POC complete!")
	log.Println("Note: Actual interface creation requires platform-specific code.")
	log.Println("Warren will use:")
	log.Println("  - Linux: kernel WireGuard (via netlink)")
	log.Println("  - macOS/Windows: wireguard-go (userspace)")
}

// Example: Create WireGuard configuration
func exampleConfig() *wgtypes.Config {
	// Generate keys
	privateKey, _ := wgtypes.GeneratePrivateKey()
	peerKey, _ := wgtypes.GeneratePrivateKey()

	// Example peer configuration
	port := 51820
	endpoint, _ := net.ResolveUDPAddr("udp", "192.168.1.10:51820")
	_, allowedIP, _ := net.ParseCIDR("10.0.0.2/32")

	return &wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &port,
		ReplacePeers: true,
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  peerKey.PublicKey(),
				Endpoint:   endpoint,
				AllowedIPs: []net.IPNet{*allowedIP},
			},
		},
	}
}

// Helper: Print configuration
func printConfig(cfg *wgtypes.Config) {
	fmt.Printf("[Interface]\n")
	fmt.Printf("PrivateKey = %s\n", cfg.PrivateKey.String())
	fmt.Printf("ListenPort = %d\n\n", *cfg.ListenPort)

	for _, peer := range cfg.Peers {
		fmt.Printf("[Peer]\n")
		fmt.Printf("PublicKey = %s\n", peer.PublicKey.String())
		fmt.Printf("Endpoint = %s\n", peer.Endpoint.String())
		fmt.Printf("AllowedIPs = ")
		for i, ip := range peer.AllowedIPs {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", ip.String())
		}
		fmt.Printf("\n\n")
	}
}
