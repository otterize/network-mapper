package gotls

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"otterize.com/ebpf/gotls/bpf"
	"regexp"
	"strconv"
	"syscall"
)

const (
	binPath = "/Users/davidrobert/ebpf/bin/main"
	//symbol  = "crypto/tls.(*Conn).Read"
	symbol = "crypto/tls.(*Conn).Write"
	//symbol = "main.(*Conn).tlsCaller"
	//symbol = "net/http.(*Client).Post"
)

const (
	mapKey uint32 = 0
)

func B2S(bs []int8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}

func disassembleGoFunction(binPath string, symbol string) (bpf.BpfGoFnInfo, error) {
	// In order to manually inspect the Go binary, we can run "objdump -dS main | less"
	// this will show us the disassembly and the go code together for easier inspection.
	// We can also use the "--disassemble" flag to objdump to disassemble a specific symbol - only works for newer versions of binutils (2.32+)

	// Run objdump command
	cmd := exec.Command("objdump", "-d", binPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run objdump: %v", err)
	}

	funcInfo := bpf.BpfGoFnInfo{}

	// Regular expressions to match the symbol and the address of copying registers to the stack
	symbolRegex := regexp.MustCompile(`^(\S+) <` + regexp.QuoteMeta(symbol) + `>:`)

	// Register names to search for
	registers := []string{"x0", "x1", "x2", "x3"}
	registersPattern := ""
	for i, reg := range registers {
		if i > 0 {
			registersPattern += "|"
		}
		registersPattern += reg
	}

	// Construct dynamic regex pattern for any of the specified registers
	pattern := fmt.Sprintf(`^\s*(\S+):\s*(?:.*\bstr\s+(%s),\s*\[sp,\s*#(\d+)\])`, registersPattern)
	stackAddressRegex := regexp.MustCompile(pattern)

	// Read output line by line
	scanner := bufio.NewScanner(&out)
	insideFunction := false
	limit := 100
	for scanner.Scan() {
		line := scanner.Text()
		if symbolRegex.MatchString(line) {
			fmt.Println("Found line:", line)
			// Parse the address from the line
			matches := symbolRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				address, err := strconv.ParseUint(matches[1], 16, 64)
				if err != nil {
					return bpf.BpfGoFnInfo{}, err
				}
				funcInfo.BaseAddr = address
			}
			insideFunction = true
		}

		if insideFunction {
			limit -= 1
			if stackAddressRegex.MatchString(line) {
				matches := stackAddressRegex.FindStringSubmatch(line)
				if len(matches) > 2 {
					register := matches[2]

					memAddr, err := strconv.ParseUint(matches[1], 16, 64)
					if err != nil {
						return bpf.BpfGoFnInfo{}, err
					}

					stackAddr, err := strconv.ParseUint(matches[3], 10, 64)
					if err != nil {
						return bpf.BpfGoFnInfo{}, err
					}

					if register == "x0" {
						funcInfo.R0StackAddr = stackAddr
					} else if register == "x1" {
						funcInfo.R1StackAddr = stackAddr
					} else if register == "x2" {
						funcInfo.R2StackAddr = stackAddr
					} else if register == "x3" {
						funcInfo.R3StackAddr = stackAddr
					}

					if funcInfo.StackAddr < memAddr {
						funcInfo.StackAddr = memAddr
					}
				}
			}

			if funcInfo.R0StackAddr > 0 && funcInfo.R1StackAddr > 0 && funcInfo.R2StackAddr > 0 && funcInfo.R3StackAddr > 0 {
				break
			}

			if limit == 0 {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading output: %v", err)
	}

	return funcInfo, nil
}

func main() {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	funcInfo, err := disassembleGoFunction(binPath, symbol)
	if err != nil {
		log.Fatalf("inspecting binary: %s", err)
	}
	uprobeOffset := funcInfo.StackAddr - funcInfo.BaseAddr

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpf.BpfObjects{}
	if err := bpf.LoadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %s", err)
	}
	defer objs.Close()

	// Open an ELF binary and read its symbols.
	ex, err := link.OpenExecutable(binPath)
	if err != nil {
		log.Fatalf("opening executable: %s", err)
	}

	// Open a Uprobe at the exit point of the symbol and attach the pre-compiled eBPF program to it.
	up, err := ex.Uprobe(symbol, objs.GotlsWriteHook, &link.UprobeOptions{
		Offset: uprobeOffset,
	})
	if err != nil {
		log.Fatalf("creating uretprobe: %s", err)
	}
	defer up.Close()

	log.Printf("Updating uprobe function info map")
	if err := objs.GoFnInfoMap.Update(mapKey, &funcInfo, ebpf.UpdateAny); err != nil {
		log.Fatalf("Failed to update map: %v", err)
	}

	reader, err := perf.NewReader(objs.Events, os.Getpagesize()*64)
	if err != nil {
		log.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	go func() {
		// Wait for a signal and close the perf reader,
		// which will interrupt rd.Read() and make the program exit.
		<-stopper
		log.Println("Received signal, exiting program..")

		if err := reader.Close(); err != nil {
			log.Fatalf("closing perf event reader: %s", err)
		}
	}()

	log.Println("Waiting for events..")

	var event bpf.BpfEvent
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				return
			}
			log.Printf("Error reading from perf event reader: %s", err)
			continue
		}

		if record.LostSamples != 0 {
			log.Printf("Perf event ring buffer full, dropped %d samples", record.LostSamples)
			continue
		}

		// Parse the perf event entry into a bpfEvent structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing perf event: %s", err)
			continue
		}

		msgString := B2S(event.Msg[:])
		log.Printf("  Pid: %d\n", event.Meta.Pid)
		log.Printf("  Msg size: %d\n", event.Meta.MsgSize)
		log.Printf("  Msg: %s\n", msgString)
	}
}
