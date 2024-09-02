package ebpf

import (
	"github.com/pkg/errors"
	"github.com/prometheus/procfs"
	"os"
	"strconv"
	"syscall"
)

func getFileInode(path string) (uint64, error) {
	stat, err := os.Stat(path)

	if err != nil {
		return 0, errors.Wrapf(err, "failed to stat %s", path)
	}

	ino, ok := stat.Sys().(*syscall.Stat_t)

	if !ok {
		return 0, errors.New("failed to get inode information for " + path)
	}

	return ino.Ino, nil
}

func getPIDNamespaceInode(pid int) (uint32, error) {
	fs, err := procfs.NewFS("/host/proc")

	if err != nil {
		return 0, errors.Wrapf(err, "failed to open procfs")
	}

	proc, err := fs.Proc(pid)

	if err != nil {
		return 0, errors.Wrapf(err, "failed to open proc %d", pid)
	}

	namespaces, err := proc.Namespaces()

	if err != nil {
		return 0, errors.Wrapf(err, "failed to get namespaces for PID %d", pid)
	}

	pidNamespace, ok := namespaces["pid"]

	if !ok {
		return 0, errors.New("failed to find PID namespace for PID " + strconv.Itoa(pid))
	}

	return pidNamespace.Inode, nil
}

func Data2Bytes(bs []uint8) []byte {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return b
}
