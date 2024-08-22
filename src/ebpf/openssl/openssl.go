package openssl

import (
	"github.com/sirupsen/logrus"
)

type Programs = opensslPrograms
type Maps = opensslMaps
type ProgramSpecs = opensslProgramSpecs
type MapSpecs = opensslMapSpecs

type Specs struct {
	ProgramSpecs
	MapSpecs
}

type Objects struct {
	Programs
	Maps
}

var BpfSpecs Specs
var BpfObjects Objects

func init() {
	err := Assign(&BpfSpecs)

	if err != nil {
		panic(err)
	}

	err = Load(&BpfObjects)

	if err != nil {
		panic(err)
	}

	logrus.Info("Loaded eBPF programs and maps")
}

func Assign(obj interface{}) error {
	spec, err := loadOpenssl()

	if err != nil {
		return err
	}

	return spec.Assign(obj)
}

func Load(obj interface{}) error {
	spec, err := loadOpenssl()

	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, nil)
}
