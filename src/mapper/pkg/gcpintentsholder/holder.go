package gcpintentsholder

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type GCPIntent struct {
	Client      model.OtterizeServiceIdentity `json:"client"`
	Permissions []string
	Resource    string
}

type GCPIntentKey struct {
	ClientName      string
	ClientNamespace string
	Resource        string
}

type TimestampedGCPIntent struct {
	Timestamp time.Time
	GCPIntent
}

type GCPIntentsHolder struct {
	intents   map[GCPIntentKey]TimestampedGCPIntent
	lock      sync.Mutex
	callbacks []GCPIntentCallbackFunc
}

type GCPIntentCallbackFunc func(context.Context, []GCPIntent)

func New() *GCPIntentsHolder {
	notifier := &GCPIntentsHolder{
		intents: make(map[GCPIntentKey]TimestampedGCPIntent),
	}

	return notifier
}

func (h *GCPIntentsHolder) RegisterNotifyIntents(callback GCPIntentCallbackFunc) {
	h.callbacks = append(h.callbacks, callback)
}

func (h *GCPIntentsHolder) AddIntent(intent GCPIntent) {
	h.lock.Lock()
	defer h.lock.Unlock()

	logrus.Debugf("Adding intent: %+v", intent)

	key := GCPIntentKey{
		ClientName:      intent.Client.Name,
		ClientNamespace: intent.Client.Namespace,
		Resource:        intent.Resource,
	}

	_, found := h.intents[key]
	now := time.Now()

	if !found {
		h.intents[key] = TimestampedGCPIntent{
			Timestamp: now,
			GCPIntent: intent,
		}
	}

	mergedIntent := h.intents[key]
	mergedIntent.Timestamp = now
	mergedIntent.Permissions = lo.Union(mergedIntent.Permissions, intent.Permissions)
	h.intents[key] = mergedIntent
}

func (h *GCPIntentsHolder) PeriodicIntentsUpload(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
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
		}
	}
}

func (h *GCPIntentsHolder) GetNewIntentsSinceLastGet() []GCPIntent {
	h.lock.Lock()
	defer h.lock.Unlock()

	intents := make([]GCPIntent, 0, len(h.intents))

	for _, intent := range h.intents {
		intents = append(intents, intent.GCPIntent)
	}

	h.intents = make(map[GCPIntentKey]TimestampedGCPIntent)

	return intents
}
