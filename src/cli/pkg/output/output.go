package output

import (
	"fmt"
	"os"
)

func PrintStderr(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", a...)
}

func PrintStdout(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format+"\n", a...)
}
