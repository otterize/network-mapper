package externaltrafficholder

import (
	"context"
	"fmt"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ExternalTrafficIntent interface {
	GetClient() model.OtterizeServiceIdentity
	GetKey() ExternalTrafficKey
	GetLastSeen() time.Time
}

type IP string

type DNSExternalTrafficIntent struct {
	Client   model.OtterizeServiceIdentity `json:"client"`
	LastSeen time.Time
	DNSName  string
	IPs      map[IP]struct{}
	TTL      time.Time
}

type IPExternalTrafficIntent struct {
	Client   model.OtterizeServiceIdentity `json:"client"`
	LastSeen time.Time
	IP       IP
}

func (i IPExternalTrafficIntent) GetClient() model.OtterizeServiceIdentity {
	return i.Client
}

func (i IPExternalTrafficIntent) GetKey() ExternalTrafficKey {
	return ExternalTrafficKey{
		ClientName:      i.Client.Name,
		ClientNamespace: i.Client.Namespace,
		DestIP:          i.IP,
	}
}

func (i IPExternalTrafficIntent) GetLastSeen() time.Time {
	return i.LastSeen
}

type TimestampedExternalTrafficIntent struct {
	Timestamp time.Time
	Intent    ExternalTrafficIntent
}

func (i DNSExternalTrafficIntent) GetClient() model.OtterizeServiceIdentity {
	return i.Client
}

func (i DNSExternalTrafficIntent) GetKey() ExternalTrafficKey {
	return ExternalTrafficKey{
		ClientName:      i.Client.Name,
		ClientNamespace: i.Client.Namespace,
		DestDNSName:     i.DNSName,
	}
}

func (i DNSExternalTrafficIntent) GetLastSeen() time.Time {
	return i.LastSeen
}

type ExternalTrafficKey struct {
	ClientName      string
	ClientNamespace string
	// One of...
	DestDNSName string
	DestIP      IP
}

type ExternalTrafficIntentsHolder struct {
	intentsNoDelay   map[ExternalTrafficKey]TimestampedExternalTrafficIntent
	delayedIPIntents map[ExternalTrafficKey]TimestampedExternalTrafficIntent
	lock             sync.Mutex
	callbacks        []ExternalTrafficCallbackFunc
}

type ExternalTrafficCallbackFunc func(context.Context, []TimestampedExternalTrafficIntent)

func NewExternalTrafficIntentsHolder() *ExternalTrafficIntentsHolder {
	return &ExternalTrafficIntentsHolder{
		intentsNoDelay:   make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent),
		delayedIPIntents: make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent),
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

// GetNewIntentsSinceLastGet returns the intents that were added since the last call to this function. It also rotates the intentsNoDelay, so that the next call will return the intentsNoDelay that were added in the next iteration.
func (h *ExternalTrafficIntentsHolder) GetNewIntentsSinceLastGet() []TimestampedExternalTrafficIntent {
	h.lock.Lock()
	defer h.lock.Unlock()

	intents := make([]TimestampedExternalTrafficIntent, 0, len(h.intentsNoDelay))

	for _, intent := range h.intentsNoDelay {
		intents = append(intents, intent)
	}

	// Rotate delayedIPIntents into intentsNoDelay
	h.intentsNoDelay = h.delayedIPIntents
	h.delayedIPIntents = make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent)

	return intents
}

// AddIntent adds a new external traffic intent to the holder. DNS intentsNoDelay are added to the current iteration, while IP intentsNoDelay are added to the next iteration. This is so that DNS traffic is reported first,
// to allow Otterize Cloud to cache the DNS name and IPs before the IP intent is sent.
func (h *ExternalTrafficIntentsHolder) AddIntent(intent ExternalTrafficIntent) {
	if config.ExcludedNamespaces().Contains(intent.GetClient().Namespace) {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	key := intent.GetKey()

	switch typedIntent := intent.(type) {
	case DNSExternalTrafficIntent:
		_, ok := h.intentsNoDelay[key]
		if !ok {
			h.intentsNoDelay[key] = TimestampedExternalTrafficIntent{
				Timestamp: intent.GetLastSeen(),
				Intent:    intent,
			}
			return
		}

		mergedIntent := h.intentsNoDelay[key]
		if intent.GetLastSeen().After(mergedIntent.Timestamp) {
			mergedIntent.Timestamp = intent.GetLastSeen()
		}

		for ip := range typedIntent.IPs {
			mergedIntent.Intent.(DNSExternalTrafficIntent).IPs[ip] = struct{}{}
		}
		h.intentsNoDelay[key] = mergedIntent

	case IPExternalTrafficIntent:
		_, ok := h.delayedIPIntents[key]
		if !ok {
			h.delayedIPIntents[key] = TimestampedExternalTrafficIntent{
				Timestamp: intent.GetLastSeen(),
				Intent:    intent,
			}
			return
		}

		mergedIntent := h.delayedIPIntents[key]
		if intent.GetLastSeen().After(mergedIntent.Timestamp) {
			mergedIntent.Timestamp = intent.GetLastSeen()
		}
		h.delayedIPIntents[key] = mergedIntent

	default:
		panic(fmt.Sprintf("Unexpected external traffic intent type: %T", intent))
	}
}
