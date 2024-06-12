package main

import (
	"fmt"
	"log"
	"net"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

const ifaceName = "eth0" // Change this to your network interface name

func main() {
	// Allow the current process to lock memory for eBPF maps
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatalf("Failed to remove memory lock limit: %v", err)
	}

	// Load the eBPF program
	spec, err := ebpf.LoadCollectionSpec("xdp_drop_tcp_port.o")
	if err != nil {
		log.Fatalf("Failed to load eBPF program: %v", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		log.Fatalf("Failed to create eBPF collection: %v", err)
	}
	defer coll.Close()

	// Load the drop port map
	dropPortMap := coll.Maps["drop_port"]
	if dropPortMap == nil {
		log.Fatalf("Failed to find drop_port map in eBPF program")
	}

	// Set the drop port from userspace
	port := uint16(4040) // Change this to your desired port
	key := uint32(0)
	if err := dropPortMap.Put(key, port); err != nil {
		log.Fatalf("Failed to set drop port: %v", err)
	}

	// Convert the interface name to an interface index
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatalf("Failed to get interface by name: %v", err)
	}

	// Attach the eBPF program to the network interface
	xdpProg := coll.Programs["xdp_drop_tcp_port"]
	if xdpProg == nil {
		log.Fatalf("Failed to find xdp_drop_tcp_port program in eBPF collection")
	}

	lnk, err := link.AttachXDP(link.XDPOptions{
		Program:   xdpProg,
		Interface: iface.Index, // Use the interface index
	})
	if err != nil {
		log.Fatalf("Failed to attach XDP program: %v", err)
	}
	defer lnk.Close()

	fmt.Printf("eBPF program successfully loaded and attached to interface %s\n", ifaceName)
	fmt.Printf("TCP packets to port %d will be dropped\n", port)

	// Keep the program running
	select {}
}
