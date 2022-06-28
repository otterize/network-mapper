package main

import (
	"context"
	"github.com/otterize/otternose/sniffer/pkg/sniffer"
)

func main() {
	snifferInstance := sniffer.NewSniffer()
	err := snifferInstance.RunForever(context.Background())
	if err != nil {
		panic(err)
	}
}
