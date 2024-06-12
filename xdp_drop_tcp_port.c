#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <bpf/bpf_helpers.h>
#include <linux/in.h>
#include <linux/types.h> // Include for u32 and u16

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32); // Use __u32 instead of u32
    __type(value, __u16); // Use __u16 instead of u16
} drop_port SEC(".maps");

SEC("xdp")
int xdp_drop_tcp_port(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    if (eth->h_proto != __constant_htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *ip = data + sizeof(struct ethhdr);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    if (ip->protocol != IPPROTO_TCP)
        return XDP_PASS;

    struct tcphdr *tcp = (void *)(ip + 1);
    if ((void *)(tcp + 1) > data_end)
        return XDP_PASS;

    __u32 key = 0; // Use __u32 instead of u32
    __u16 *port = bpf_map_lookup_elem(&drop_port, &key); // Use __u16 instead of u16
    if (port && tcp->dest == __constant_htons(*port)) {
        return XDP_DROP;
    }

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
