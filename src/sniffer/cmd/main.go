package main

import (
	"github.com/otterize/otternose/sniffer/pkg/sniffer"
)

func main() {
	snifferInstance := sniffer.NewSniffer()
	err := snifferInstance.RunForever()
	if err != nil {
		panic(err)
	}
}
