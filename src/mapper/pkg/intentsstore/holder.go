package intentsstore

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

type SourceDestPair struct {
	Source      types.NamespacedName
	Destination types.NamespacedName
}

type TimestampedIntent struct {
	Timestamp time.Time
	Intent    model.Intent
}

type IntentsStore map[SourceDestPair]TimestampedIntent

type IntentsHolder struct {
	accumulatingStore IntentsStore
	sinceLastGetStore IntentsStore
	lock              sync.Mutex
	lastIntentsUpdate time.Time
}

func NewIntentsHolder() *IntentsHolder {
	return &IntentsHolder{
		accumulatingStore: make(IntentsStore),
		sinceLastGetStore: make(IntentsStore),
		lock:              sync.Mutex{},
	}
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.accumulatingStore = make(IntentsStore)
}

func (i *IntentsHolder) AddIntent(newTimestamp time.Time, intent model.Intent) {
	i.lock.Lock()
	defer i.lock.Unlock()

	pair := SourceDestPair{
		Source:      intent.Client.AsNamespacedName(),
		Destination: intent.Server.AsNamespacedName(),
	}
	existingIntent, ok := i.accumulatingStore[pair]
	if !ok || newTimestamp.After(existingIntent.Timestamp) {
		timestampedIntent := TimestampedIntent{
			Timestamp: newTimestamp,
			Intent:    intent,
		}
		i.accumulatingStore[pair] = timestampedIntent
		i.sinceLastGetStore[pair] = timestampedIntent
		i.lastIntentsUpdate = time.Now()
	}

}

func (i *IntentsHolder) GetIntents(namespaces []string, includeLabels []string, includeAllLabels bool) []TimestampedIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.getIntentsFromStore(i.accumulatingStore, namespaces, includeLabels, includeAllLabels)
}

func (i *IntentsHolder) GetNewIntentsSinceLastGet() []TimestampedIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	intents := i.getIntentsFromStore(i.sinceLastGetStore, nil, nil, false)
	i.sinceLastGetStore = make(IntentsStore)
	return intents
}

func (i *IntentsHolder) getIntentsFromStore(store IntentsStore, namespaces []string, includeLabels []string, includeAllLabels bool) []TimestampedIntent {
	namespacesSet := goset.FromSlice(namespaces)
	includeLabelsSet := goset.FromSlice(includeLabels)
	result := make([]TimestampedIntent, 0)
	for pair, intent := range store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}

		if !includeAllLabels {
			labelsFilter := func(labels []model.PodLabel) []model.PodLabel {
				return lo.Filter(labels, func(label model.PodLabel, _ int) bool {
					return includeLabelsSet.Contains(label.Key)
				})
			}
			intent.Intent.Client.Labels = labelsFilter(intent.Intent.Client.Labels)
			intent.Intent.Server.Labels = labelsFilter(intent.Intent.Server.Labels)
		}

		result = append(result, intent)
	}
	return result
}

func GroupIntentsBySource(intents []TimestampedIntent) []model.ServiceIntents {
	intentsBySource := make(map[types.NamespacedName]*model.ServiceIntents, 0)
	for _, intent := range intents {
		srcIdentity := intent.Intent.Client.AsNamespacedName()
		if _, ok := intentsBySource[srcIdentity]; !ok {
			intentsBySource[srcIdentity] = &model.ServiceIntents{
				Client:  intent.Intent.Client,
				Intents: make([]model.OtterizeServiceIdentity, 0),
			}
		}

		intentsBySource[srcIdentity].Intents = append(intentsBySource[srcIdentity].Intents, *intent.Intent.Server)
	}
	return lo.Map(lo.Values(intentsBySource), func(serviceIntents *model.ServiceIntents, _ int) model.ServiceIntents {
		return *serviceIntents
	})
}
