## Overview

This package enables eBPF-based tracing of TLS data in Go applications.
This is done by attaching uprobes to the crypto/tls read and write functions in Go, allowing to capture and inspect plain-text data during TLS communication.

### Hook points

Getting the required symbols can be done by using the `objdump` command. The following symbols are required to attach uprobes to the read and write functions of the `crypto/tls` package:
- `crypto/tls.(*Conn).Read`
- `crypto/tls.(*Conn).Write`

Useful [tutorial](https://www.grant.pizza/blog/tracing-go-functions-with-ebpf-part-1/) on the subject

### Uprobes vs Uretprobes

Go's runtime makes it impossible using uretprobes due to its stack management.
The stack in Go is small initially and expands as needed, which copies the old
stack into a new memory location. This behavior breaks uretprobes, which rely
on the function stack to capture return values. To properly trace function 
returns in Go, it's necessary to attach uprobes to all return points within 
the function, as uretprobes can lead to program errors during stack expansion.

A nice [explanation](https://blog.0x74696d.com/posts/challenges-of-bpf-tracing-go/) regarding the subject

### Go version

Prior to go version 1.17 go passed arguments to functions using the stack.
This really complicated the task of tracing functions in Go, as we needed to
get the stack pointer and read the arguments from the stack by offsets.

Starting from go version 1.17, We can read the arguments from the registers.

A nice [explanation](https://blog.0x74696d.com/posts/challenges-of-bpf-tracing-go/) regarding the subject

### The g struct
Since we are using uprobes to hook into the return points of the `Read` function,
we are losing reference to the input buffer. To solve this issue we need to get
the buffer pointer on the function start address and pass it to the return point.
This is done by using the `type g struct` defined [here](https://github.com/golang/go/blob/master/src/runtime/runtime2.go).
The struct contains the current goroutine id which in conjuction with the process id
can be used to uniquely identify the goroutine.

The struct is located in a register (for go version 1.18+) and depends on the
architecture:
- ARM64: https://go.googlesource.com/go/+/refs/heads/master/src/cmd/compile/abi-internal.md#arm64-architecture
- AMD64: https://go.googlesource.com/go/+/refs/heads/master/src/cmd/compile/abi-internal.md#amd64-architecture

The goid is located at an offset of 152 bytes from the base address of the g struct.

We grab the buffer pointer in the entry point of the `Read` from the register 
and using pid + goid we save it in a bpf map. The value is then available in the return
point of the `Read` function using the same id calculation.
