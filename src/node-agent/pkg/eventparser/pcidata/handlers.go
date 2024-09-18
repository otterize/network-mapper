package pcidata

import (
	"github.com/otterize/iamlive/iamlivecore"
	"github.com/otterize/intents-operator/src/shared/errors"
	ebpftypes "github.com/otterize/network-mapper/src/node-agent/pkg/ebpf/types"
	"io"
	"net/http"
)

const AWSHost = "amazonaws.com"

func HandleAwsRequest(ctx ebpftypes.EventContext, req *http.Request) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return errors.Wrap(err)
	}

	// Check if the event is an AWS request - called to host "amazonaws.com"
	if req.Host != AWSHost {
		return nil
	}

	// Check if the event is an egress event
	if ebpftypes.Direction(ctx.Event.Meta.Direction) != ebpftypes.DirectionEgress {
		return nil
	}

	iamlivecore.HandleAWSRequest(req, body, 200)
	return nil
}
