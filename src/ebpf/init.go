package ebpf

import "github.com/sirupsen/logrus"

var Objs BpfObjects
var Specs BpfSpecs

func InitializeBpf() {
	// Load and assign specs
	specs, err := LoadBpf()
	if err != nil {
		logrus.Fatalf("error loading specs: %s", err)

	}
	err = specs.Assign(&Specs)
	if err != nil {
		logrus.Fatalf("error assigning specs: %s", err)
	}

	if err := LoadBpfObjects(&Objs, nil); err != nil {
		logrus.Fatalf("error loading objects: %s", err)
	}

	logrus.Info("Loaded gotls objects")
}
