package ebpf

import (
	"github.com/cilium/ebpf"
	sharedconfig "github.com/otterize/network-mapper/src/shared/config"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var Objs BpfObjects
var Specs BpfSpecs

func LoadEBpfPrograms() {
	logrus.Info("Loading eBPF programs")

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

	logrus.Info("eBPF programs loaded successfully")

	if viper.GetBool(sharedconfig.DebugKey) {
		logrus.Debugf("eBPF programs loaded: %+v",
			lo.MapToSlice(specs.Programs, func(_ string, p *ebpf.ProgramSpec) string {
				return p.Name
			}),
		)
	}
}
