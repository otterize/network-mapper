package version

import (
	"os"
	"sync"
)

var (
	version string
	once    sync.Once
)

// Version returns the current version.
// It is implemented by reading a file instead of go:embed to avoid cache busting the Dockerfile before the build.
func Version() string {
	once.Do(func() {
		data, err := os.ReadFile("./shared/version/version")
		if err == nil {
			// only in development mode
			version = string(data)
			return
		}

		// only in production
		data, err = os.ReadFile("./version")
		if err != nil {
			panic(err)
		}
		version = string(data)
	})

	return version
}
