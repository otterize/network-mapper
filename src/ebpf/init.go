package ebpf

import "github.com/sirupsen/logrus"

var Objs BpfObjects
var Specs BpfSpecs

func init() {
	// Load pre-compiled programs and maps into the kernel.
	if err := LoadBpfObjects(&Objs, nil); err != nil {
		logrus.Fatalf("loading objects: %s", err)
	}

	logrus.Info("Loaded gotls objects")
}
