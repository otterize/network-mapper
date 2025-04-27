package externaltrafficholder

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

type ExternalTrafficIntent struct {
	Client   model.OtterizeServiceIdentity `json:"client"`
	LastSeen time.Time
	DNSName  string
	IPs      map[IP]struct{}
}

type TimestampedExternalTrafficIntent struct {
	Timestamp        time.Time
	Intent           ExternalTrafficIntent
	ConnectionsCount *cloudclient.ConnectionsCount
}

type ExternalTrafficKey struct {
	ClientName      string
	ClientNamespace string
	DestDNSName     string
}

type IntentsConnectionCounter map[ExternalTrafficKey]*concurrentconnectioncounter.ConnectionCounter[*concurrentconnectioncounter.CountableIntentExternalTrafficIntent]

type ExternalTrafficIntentsHolder struct {
	intents               map[ExternalTrafficKey]TimestampedExternalTrafficIntent
	lock                  sync.Mutex
	callbacks             []ExternalTrafficCallbackFunc
	connectionCountDiffer *concurrentconnectioncounter.ConnectionCountDiffer[ExternalTrafficKey, *concurrentconnectioncounter.CountableIntentExternalTrafficIntent]
}

type ExternalTrafficCallbackFunc func(context.Context, []TimestampedExternalTrafficIntent)

func NewExternalTrafficIntentsHolder() *ExternalTrafficIntentsHolder {
	return &ExternalTrafficIntentsHolder{
		intents:               make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent),
		connectionCountDiffer: concurrentconnectioncounter.NewConnectionCountDiffer[ExternalTrafficKey, *concurrentconnectioncounter.CountableIntentExternalTrafficIntent](),
	}
}

func (h *ExternalTrafficIntentsHolder) RegisterNotifyIntents(callback ExternalTrafficCallbackFunc) {
	h.callbacks = append(h.callbacks, callback)
}

func (h *ExternalTrafficIntentsHolder) PeriodicIntentsUpload(ctx context.Context, interval time.Duration) {
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

func (h *ExternalTrafficIntentsHolder) GetNewIntentsSinceLastGet() []TimestampedExternalTrafficIntent {
	h.lock.Lock()
	defer h.lock.Unlock()

	intents := make([]TimestampedExternalTrafficIntent, 0, len(h.intents))

	for key, intent := range h.intents {
		// Add connection count value
		connectionsCount, connectionsCountValid := h.connectionCountDiffer.GetDiff(key)
		if connectionsCountValid {
			intent.ConnectionsCount = lo.ToPtr(connectionsCount)
		}

		intents = append(intents, intent)
	}

	h.intents = make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent)
	h.connectionCountDiffer.Reset()

	return intents
}

func (h *ExternalTrafficIntentsHolder) AddIntent(intent ExternalTrafficIntent) {
	if config.ExcludedNamespaces().Contains(intent.Client.Namespace) {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	key := ExternalTrafficKey{
		ClientName:      intent.Client.Name,
		ClientNamespace: intent.Client.Namespace,
		DestDNSName:     intent.DNSName,
	}
	_, found := h.intents[key]
	h.connectionCountDiffer.Increment(key, concurrentconnectioncounter.CounterInput[*concurrentconnectioncounter.CountableIntentExternalTrafficIntent]{
		Intent:      concurrentconnectioncounter.NewCountableIntentExternalTrafficIntent(),
		SourcePorts: make([]int64, 0),
	})

	if !found {
		h.intents[key] = TimestampedExternalTrafficIntent{
			Timestamp: intent.LastSeen,
			Intent:    intent,
		}
		return
	}

	mergedIntent := h.intents[key]

	for ip := range intent.IPs {
		mergedIntent.Intent.IPs[ip] = struct{}{}
	}
	if intent.LastSeen.After(mergedIntent.Timestamp) {
		mergedIntent.Timestamp = intent.LastSeen
	}

	h.intents[key] = mergedIntent
}
