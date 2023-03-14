package intentsstore

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"sync"
	"time"
)

type NamespacedName struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type FullInfoIntentWithTime struct {
	SourceFullInfo      model.OtterizeServiceIdentity
	DestinationFullInfo model.OtterizeServiceIdentity
	Timestamp           time.Time
	Intent              cloudclient.IntentInput
}

type SourceDestPair struct {
	Source      NamespacedName
	Destination NamespacedName
}

func serviceIdentityToNameNamespacePair(identity model.OtterizeServiceIdentity) NamespacedName {
	return NamespacedName{
		Name:      identity.Name,
		Namespace: identity.Namespace,
	}
}

type DiscoveredIntent struct {
	Source      model.OtterizeServiceIdentity
	Destination model.OtterizeServiceIdentity
	Timestamp   time.Time
	Intent      cloudclient.IntentInput
}

type IntentsStore map[SourceDestPair]FullInfoIntentWithTime

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

func (i *IntentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity, newTimestamp time.Time, intent cloudclient.IntentInput) {
	i.lock.Lock()
	defer i.lock.Unlock()

	pair := SourceDestPair{Source: serviceIdentityToNameNamespacePair(srcService), Destination: serviceIdentityToNameNamespacePair(dstService)}
	timestampedPair, alreadyExists := i.accumulatingStore[pair]
	if !alreadyExists || newTimestamp.After(timestampedPair.Timestamp) {
		fullIntentInfo := FullInfoIntentWithTime{SourceFullInfo: srcService, DestinationFullInfo: dstService, Timestamp: newTimestamp, Intent: intent}
		i.accumulatingStore[pair] = fullIntentInfo
		i.sinceLastGetStore[pair] = fullIntentInfo
		i.lastIntentsUpdate = time.Now()
	}

}

func (i *IntentsHolder) GetIntents(namespaces []string, includeLabels []string, includeAllLabels bool) []DiscoveredIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.getIntentsFromStore(i.accumulatingStore, namespaces, includeLabels, includeAllLabels)
}

func (i *IntentsHolder) GetNewIntentsSinceLastGet() []DiscoveredIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	intents := i.getIntentsFromStore(i.sinceLastGetStore, nil, nil, false)
	i.sinceLastGetStore = make(IntentsStore)
	return intents
}

func (i *IntentsHolder) getIntentsFromStore(store IntentsStore, namespaces []string, includeLabels []string, includeAllLabels bool) []DiscoveredIntent {
	namespacesSet := goset.FromSlice(namespaces)
	includeLabelsSet := goset.FromSlice(includeLabels)
	result := make([]DiscoveredIntent, 0)
	for pair, timestampedInfo := range store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}
		timestampedInfoCopy := timestampedInfo

		if !includeAllLabels {
			timestampedInfoCopy.SourceFullInfo.Labels = lo.Filter(timestampedInfoCopy.SourceFullInfo.Labels, func(label model.PodLabel, _ int) bool {
				return includeLabelsSet.Contains(label.Key)
			})
			timestampedInfoCopy.DestinationFullInfo.Labels = lo.Filter(timestampedInfoCopy.DestinationFullInfo.Labels, func(label model.PodLabel, _ int) bool {
				return includeLabelsSet.Contains(label.Key)
			})
		}

		result = append(result, DiscoveredIntent{
			Source:      timestampedInfoCopy.SourceFullInfo,
			Destination: timestampedInfoCopy.DestinationFullInfo,
			Timestamp:   timestampedInfoCopy.Timestamp,
			Intent:      timestampedInfoCopy.Intent,
		})
	}
	return result
}

type SourceWithDestinations struct {
	Source       model.OtterizeServiceIdentity
	Destinations []model.OtterizeServiceIdentity
}

func GroupDestinationsBySource(discoveredIntents []DiscoveredIntent) []SourceWithDestinations {
	serviceMap := make(map[NamespacedName]*SourceWithDestinations, 0)
	for _, intents := range discoveredIntents {
		srcIdentity := serviceIdentityToNameNamespacePair(intents.Source)
		if _, ok := serviceMap[srcIdentity]; !ok {
			serviceMap[srcIdentity] = &SourceWithDestinations{
				Source:       intents.Source,
				Destinations: make([]model.OtterizeServiceIdentity, 0),
			}
		}

		destinations := append(serviceMap[srcIdentity].Destinations, intents.Destination)
		serviceMap[srcIdentity].Destinations = destinations
	}
	return lo.MapToSlice(serviceMap, func(_ NamespacedName, client *SourceWithDestinations) SourceWithDestinations {
		return *client
	})
}
