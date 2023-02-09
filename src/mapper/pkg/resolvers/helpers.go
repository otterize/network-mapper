package resolvers

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

type nameNamespaceIdentity struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type clientWithDestinations struct {
	Client       model.OtterizeServiceIdentity
	Destinations []model.OtterizeServiceIdentity
}

type FullInfoIntentWithTime struct {
	SourceFullInfo      model.OtterizeServiceIdentity
	DestinationFullInfo model.OtterizeServiceIdentity
	Timestamp           time.Time
}

type SourceDestPair struct {
	Source      nameNamespaceIdentity
	Destination nameNamespaceIdentity
}

func serviceIdentityToNameNamespacePair(identity model.OtterizeServiceIdentity) nameNamespaceIdentity {
	return nameNamespaceIdentity{
		Name:      identity.Name,
		Namespace: identity.Namespace,
	}
}

type DiscoveredIntent struct {
	Source      model.OtterizeServiceIdentity `json:"source"`
	Destination model.OtterizeServiceIdentity `json:"destination"`
	Timestamp   time.Time                     `json:"timestamp"`
}

type IntentsHolder struct {
	store             map[SourceDestPair]FullInfoIntentWithTime
	lock              sync.Mutex
	client            client.Client
	lastIntentsUpdate time.Time
}

func NewIntentsHolder(client client.Client) *IntentsHolder {
	return &IntentsHolder{
		store:  make(map[SourceDestPair]FullInfoIntentWithTime),
		lock:   sync.Mutex{},
		client: client,
	}
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store = make(map[SourceDestPair]FullInfoIntentWithTime)
}

func (i *IntentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity, newTimestamp time.Time) {
	i.lock.Lock()
	defer i.lock.Unlock()

	pair := SourceDestPair{Source: serviceIdentityToNameNamespacePair(srcService), Destination: serviceIdentityToNameNamespacePair(dstService)}
	timestampedPair, alreadyExists := i.store[pair]
	if !alreadyExists || newTimestamp.After(timestampedPair.Timestamp) {
		i.store[pair] = FullInfoIntentWithTime{SourceFullInfo: srcService, DestinationFullInfo: dstService, Timestamp: newTimestamp}
		i.lastIntentsUpdate = time.Now()
	}
}

func (i *IntentsHolder) LastIntentsUpdate() time.Time {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.lastIntentsUpdate
}

func (i *IntentsHolder) GetIntents(namespaces []string) []DiscoveredIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.getIntents(namespaces)
}

func (i *IntentsHolder) getIntents(namespaces []string) []DiscoveredIntent {
	namespacesSet := goset.FromSlice(namespaces)
	result := make([]DiscoveredIntent, 0)
	for pair, timestampedInfo := range i.store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}

		result = append(result, DiscoveredIntent{
			Source:      timestampedInfo.SourceFullInfo,
			Destination: timestampedInfo.DestinationFullInfo,
			Timestamp:   timestampedInfo.Timestamp,
		})
	}
	return result
}

func groupDestinationsBySource(discoveredIntents []DiscoveredIntent) []clientWithDestinations {
	serviceMap := make(map[nameNamespaceIdentity]*clientWithDestinations, 0)
	for _, intents := range discoveredIntents {
		srcIdentity := serviceIdentityToNameNamespacePair(intents.Source)
		if _, ok := serviceMap[srcIdentity]; !ok {
			serviceMap[srcIdentity] = &clientWithDestinations{
				Client:       intents.Source,
				Destinations: make([]model.OtterizeServiceIdentity, 0),
			}
		}

		destinations := append(serviceMap[srcIdentity].Destinations, intents.Destination)
		serviceMap[srcIdentity].Destinations = destinations
	}
	return lo.MapToSlice(serviceMap, func(_ nameNamespaceIdentity, client *clientWithDestinations) clientWithDestinations {
		return *client
	})
}
