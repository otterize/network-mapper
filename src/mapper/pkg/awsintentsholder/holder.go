package awsintentsholder

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type AWSIntent struct {
	Client  model.OtterizeServiceIdentity `json:"client"`
	Actions []string
	ARN     string
}

type AWSIntentKey struct {
	ClientName      string
	ClientNamespace string
	ARN             string
}

type TimestampedAWSIntent struct {
	Timestamp time.Time
	AWSIntent
}

type AWSIntentsHolder struct {
	intents   map[AWSIntentKey]TimestampedAWSIntent
	lock      sync.Mutex
	callbacks []AWSIntentCallbackFunc
}

type AWSIntentCallbackFunc func(context.Context, []AWSIntent)

func New() *AWSIntentsHolder {
	notifier := &AWSIntentsHolder{
		intents: make(map[AWSIntentKey]TimestampedAWSIntent),
	}

	return notifier
}

func (h *AWSIntentsHolder) RegisterNotifyIntents(callback AWSIntentCallbackFunc) {
	h.callbacks = append(h.callbacks, callback)
}

func (h *AWSIntentsHolder) AddIntent(intent AWSIntent) {
	h.lock.Lock()
	defer h.lock.Unlock()

	logrus.Debugf("Adding intent: %+v", intent)

	key := AWSIntentKey{
		ClientName:      intent.Client.Name,
		ClientNamespace: intent.Client.Namespace,
		ARN:             intent.ARN,
	}

	_, found := h.intents[key]
	now := time.Now()

	if !found {
		h.intents[key] = TimestampedAWSIntent{
			Timestamp: now,
			AWSIntent: intent,
		}
	}

	mergedIntent := h.intents[key]
	mergedIntent.Timestamp = now
	mergedIntent.Actions = lo.Union(mergedIntent.Actions, intent.Actions)
	h.intents[key] = mergedIntent
}

func (h *AWSIntentsHolder) PeriodicIntentsUpload(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			if len(h.callbacks) == 0 {
				continue
			}

			intents := h.getNewIntentsSinceLastGet()
			if len(intents) == 0 {
				continue
			}

			for _, callback := range h.callbacks {
				callback(ctx, intents)
			}
		}
	}
}

func (h *AWSIntentsHolder) getNewIntentsSinceLastGet() []AWSIntent {
	h.lock.Lock()
	defer h.lock.Unlock()

	transformers := AWSIntentRootTransformer{}
	intents := lo.MapToSlice(h.intents, func(key AWSIntentKey, value TimestampedAWSIntent) AWSIntent {
		return value.AWSIntent
	})

	return transformers.Transform(intents)
}
