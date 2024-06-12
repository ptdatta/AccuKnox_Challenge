package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf"
)

const (
	TASK_COMM_LEN = 16
	ETH_P_IP      = 0x0800
	IPPROTO_TCP   = 6
)

var (
	PROCESS_NAME = "myprocess"
	TARGET_PORT  = uint16(4040)
)

func main() {
	code := `
#include <uapi/linux/ptrace.h>
#include <net/sock.h>

#define TASK_COMM_LEN 16
#define ETH_P_IP 0x0800
#define IPPROTO_TCP 6

BPF_HASH(tcp_ports, u32, u16);

int filter_tcp_port(struct __sk_buff *skb) {
    u32 pid = bpf_get_current_pid_tgid();
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    char comm[TASK_COMM_LEN];
    bpf_get_current_comm(&comm, sizeof(comm));

    u16 *port_ptr = tcp_ports.lookup(&pid);
    if (port_ptr != 0 && bpf_skb_load_bytes(skb, offsetof(struct ethhdr, h_proto), &pid, sizeof(pid)) == sizeof(pid) && pid == htons(ETH_P_IP) && bpf_skb_load_bytes(skb, offsetof(struct iphdr, protocol), &pid, sizeof(pid)) == sizeof(pid) && pid == IPPROTO_TCP && bpf_get_socket_cookie(skb, 0) == pid) {
        struct tcphdr *tcph = (struct tcphdr *)(skb->data + sizeof(struct ethhdr) + sizeof(struct iphdr));
        u16 dest_port = ntohs(tcph->dest);

        if (dest_port != *port_ptr) {
            bpf_trace_printk("Dropping traffic from %s (PID: %d) to port %d\n", comm, pid, dest_port);
            return TC_ACT_SHOT;
        }
    }

    return TC_ACT_OK;
}
`
	spec := ebpf.ProgramSpec{
		Type:         ebpf.SocketFilter,
		Instructions: ebpf.MustAssemble(code),
	}

	module := ebpf.NewModule(spec.String())
	defer module.Close()

	program := module.SocketFilter("filter_tcp_port")
	if program == nil {
		fmt.Fprintln(os.Stderr, "Failed to load program")
		os.Exit(1)
	}
	defer program.Close()

	// Attach program to the egress path of all sockets
	err := program.AttachSocketFilter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach program: %v\n", err)
		os.Exit(1)
	}

	// Set up a signal handler to detach the program on termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		program.DetachSocketFilter()
		os.Exit(0)
	}()

	fmt.Println("eBPF program is now attached. Press Ctrl+C to detach and exit.")
	select {}
}
