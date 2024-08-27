package gobin

import (
	"debug/buildinfo"
	"fmt"
	delve "github.com/go-delve/delve/pkg/goversion"
	"log"
	"os"
)

// FindGoVersion attempts to determine the Go version
func FindGoVersion(f *os.File) (gover delve.GoVersion, raw string, err error) {
	info, err := buildinfo.Read(f)
	if err != nil {
		log.Fatalf("error reading build info: %v", err)
	}

	versionString := info.GoVersion
	gover, ok := delve.Parse(versionString)
	if !ok {
		return gover, "", fmt.Errorf("error parsing go version %s", versionString)
	}

	return gover, versionString, nil
}
