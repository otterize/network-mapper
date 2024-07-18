//go:build ignore
#include <linux/types.h>
#include <linux/bpf.h>
#include <bpf/bpf_tracing.h>
#include <asm/ptrace.h>

struct datarec {
  __u64 counter;
} datarec;

struct {
  __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
  __type(key, __u32);
  __type(value, datarec);
  __uint(max_entries, 1);
} uprobe_stats_map SEC(".maps");

SEC("uprobe/uprobe_counter")
static __u32 uprobe_counter(struct pt_regs *ctx) {

  __u32 index = 0;
  struct datarec *rec = bpf_map_lookup_elem(&uprobe_stats_map, &index);
  if (!rec)
    return 1;

  rec->counter++;
  char msg[80];
  bpf_probe_read(&msg, sizeof(msg), (void *)PT_REGS_PARM2(ctx));
  bpf_printk("uprobe called %s", msg);

  return 0;
}

char _license[] SEC("license") = "Dual BSD/GPL";
