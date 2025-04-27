package incomingtrafficholder

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/concurrentconnectioncounter"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type IP string

type IncomingTrafficIntent struct {
	Server           model.OtterizeServiceIdentity `json:"client"`
	LastSeen         time.Time
	IP               string
	SrcPorts         []int64
	ConnectionsCount *cloudclient.ConnectionsCount
}

type TimestampedIncomingTrafficIntent struct {
	Timestamp time.Time
	Intent    IncomingTrafficIntent
}

type IncomingTrafficKey struct {
	ServerName      string
	ServerNamespace string
	IP              string
}

type IncomingTrafficIntentsHolder struct {
	intents               map[IncomingTrafficKey]TimestampedIncomingTrafficIntent
	lock                  sync.Mutex
	callbacks             []IncomingTrafficCallbackFunc
	connectionCountDiffer *concurrentconnectioncounter.ConnectionCountDiffer[IncomingTrafficKey, *concurrentconnectioncounter.CountableIncomingInternetTrafficIntent]
}

type IncomingTrafficCallbackFunc func(context.Context, []TimestampedIncomingTrafficIntent)
type IntentsConnectionCounter map[IncomingTrafficKey]*concurrentconnectioncounter.ConnectionCounter[*concurrentconnectioncounter.CountableIncomingInternetTrafficIntent]

func NewIncomingTrafficIntentsHolder() *IncomingTrafficIntentsHolder {
	return &IncomingTrafficIntentsHolder{
		intents:               make(map[IncomingTrafficKey]TimestampedIncomingTrafficIntent),
		connectionCountDiffer: concurrentconnectioncounter.NewConnectionCountDiffer[IncomingTrafficKey, *concurrentconnectioncounter.CountableIncomingInternetTrafficIntent](),
	}
}

func (h *IncomingTrafficIntentsHolder) RegisterNotifyIntents(callback IncomingTrafficCallbackFunc) {
	h.callbacks = append(h.callbacks, callback)
}

func (h *IncomingTrafficIntentsHolder) PeriodicIntentsUpload(ctx context.Context, interval time.Duration) {
	logrus.Info("Starting periodic external traffic intents upload")

	for {
		select {
		case <-time.After(interval):
			if len(h.callbacks) == 0 {
				continue
			}

			intents := h.GetNewIntentsSinceLastGet()
			if len(intents) == 0 {
				continue
			}
			for _, callback := range h.callbacks {
				callback(ctx, intents)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (h *IncomingTrafficIntentsHolder) GetNewIntentsSinceLastGet() []TimestampedIncomingTrafficIntent {
	h.lock.Lock()
	defer h.lock.Unlock()

	intents := make([]TimestampedIncomingTrafficIntent, 0, len(h.intents))

	for _, intent := range h.intents {
		key := IncomingTrafficKey{
			ServerName:      intent.Intent.Server.Name,
			ServerNamespace: intent.Intent.Server.Namespace,
			IP:              intent.Intent.IP,
		}
		connectionsCount, connectionsCountValid := h.connectionCountDiffer.GetDiff(key)
		if connectionsCountValid {
			intent.Intent.ConnectionsCount = lo.ToPtr(connectionsCount)
		}
		intents = append(intents, intent)
	}

	h.intents = make(map[IncomingTrafficKey]TimestampedIncomingTrafficIntent)
	h.connectionCountDiffer.Reset()

	return intents
}

func (h *IncomingTrafficIntentsHolder) AddIntent(intent IncomingTrafficIntent) {
	if config.ExcludedNamespaces().Contains(intent.Server.Namespace) {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	key := IncomingTrafficKey{
		ServerName:      intent.Server.Name,
		ServerNamespace: intent.Server.Namespace,
		IP:              intent.IP,
	}

	h.connectionCountDiffer.Increment(key, concurrentconnectioncounter.CounterInput[*concurrentconnectioncounter.CountableIncomingInternetTrafficIntent]{
		Intent:      concurrentconnectioncounter.NewCountableIncomingInternetTrafficIntent(),
		SourcePorts: intent.SrcPorts,
	})

	mergedIntent, ok := h.intents[key]
	if !ok {
		h.intents[key] = TimestampedIncomingTrafficIntent{
			Timestamp: intent.LastSeen,
			Intent:    intent,
		}
		return
	}

	if intent.LastSeen.After(mergedIntent.Timestamp) {
		mergedIntent.Timestamp = intent.LastSeen
	}

	h.intents[key] = mergedIntent
}
