//go:build linux

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/labstack/gommon/log"
	"os"
	"runtime"
)

func main() {
	var err error

	pid := flag.Int("pid", 0, "pid of the process to trace")
	progId := flag.Int("prog-id", 0, "id of the program to attach")
	binPath := flag.String("bin-path", "", "symbol to trace")
	fnName := flag.String("fn-name", "", "name of the function to trace")
	retprobe := flag.Bool("retprobe", false, "attach a uretprobe instead of a uprobe")

	flag.Parse()

	log.Info("pid: ", *pid)
	log.Info("prog-id: ", *progId)
	log.Info("bin-path: ", *binPath)
	log.Info("fn-name: ", *fnName)

	runtime.LockOSThread()

	//originNsPath := "/proc/1/ns/mnt"
	//originNsFd, err := os.Open(originNsPath)

	//if err != nil {
	//	log.Panic(err)
	//}

	//err = unix.Unshare(unix.CLONE_NEWNS)

	//if err != nil {
	//	log.Panic(err)
	//}

	fullBinPath := fmt.Sprintf("/host/proc/%d/root%s", *pid, *binPath)

	//mntNsFd, err := os.Open(mntNsPath)
	//
	//if err != nil {
	//	log.Panic(err)
	//}
	//
	//defer mntNsFd.Close()

	//err = unix.Setns(int(mntNsFd.Fd()), syscall.CLONE_NEWNS)
	//
	//if err != nil {
	//	log.Panic(err)
	//}

	ex, err := link.OpenExecutable(fullBinPath)

	if err != nil {
		log.Fatalf("opening executable: %s", err)
	}

	program, err := ebpf.NewProgramFromID(ebpf.ProgramID(*progId))

	if err != nil {
		log.Fatalf("loading program: %s", err)
	}

	programInfo, err := program.Info()

	if err != nil {
		log.Fatalf("loading program info: %s", err)
	}

	programId, programIdAvailable := programInfo.ID()

	log.Infof("Program loaded %d (%v): %s", programId, programIdAvailable, programInfo.Name)

	var probe link.Link

	if *retprobe {
		probe, err = ex.Uretprobe(*fnName, program, nil)

		if err != nil {
			log.Fatalf("creating uretprobe: %s", err)
		}

		defer probe.Close()
		log.Info("Uretprobe created")
	} else {
		probe, err = ex.Uprobe(*fnName, program, &link.UprobeOptions{
			PID: *pid,
		})

		if err != nil {
			log.Fatalf("creating uprobe: %s", err)
		}

		defer probe.Close()
		log.Info("Uprobe created")
	}

	log.Infof("Probe loaded: %T", probe)

	//err = unix.Setns(int(originNsFd.Fd()), syscall.CLONE_NEWNS)
	//
	//if err != nil {
	//	log.Panic(err)
	//}

	//log.Info("Switched back to original namespace")
	//
	//err = probe.Pin("/host/sys/fs/bpf/nsutil_link")
	//
	//if err != nil {
	//	log.Fatalf("pinning probe: %s", err)
	//}
	//
	//log.Info("Probe pinned")

	log.Info("press any key to exit")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}
