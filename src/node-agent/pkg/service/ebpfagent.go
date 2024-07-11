package service

import (
	"context"
	bpfmanclient "github.com/bpfman/bpfman/clients/gobpfman/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const socketPath = "unix:///run/bpfman-sock/bpfman.sock"

func ConnectToBpfmanOrDie(ctx context.Context) *bpfmanclient.BpfmanClient {
	conn, err := grpc.NewClient(socketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		logrus.WithError(err).Panic("Failed to create grpc client")
	}

	client := bpfmanclient.NewBpfmanClient(conn)

	return &client
}
