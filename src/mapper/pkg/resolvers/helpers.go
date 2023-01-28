package resolvers

import (
	"fmt"
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

type SourceDestPair struct {
	Source      model.OtterizeServiceIdentity
	Destination model.OtterizeServiceIdentity
}

type DiscoveredIntent struct {
	Source      model.OtterizeServiceIdentity `json:"source"`
	Destination model.OtterizeServiceIdentity `json:"destination"`
	Timestamp   time.Time                     `json:"timestamp"`
}

type IntentsHolderConfig struct {
	StoreConfigMap string
	Namespace      string
}

func namespaceFromConfig() (string, error) {
	if viper.IsSet(config.NamespaceKey) {
		return viper.GetString(config.NamespaceKey), nil
	}
	namespace, err := kubeutils.GetCurrentNamespace()
	if err != nil {
		return "", fmt.Errorf("could not deduce the store's configmap namespace: %w", err)
	}
	return namespace, nil
}

func IntentsHolderConfigFromViper() (IntentsHolderConfig, error) {
	namespace, err := namespaceFromConfig()
	if err != nil {
		return IntentsHolderConfig{}, err
	}
	return IntentsHolderConfig{
		StoreConfigMap: viper.GetString(config.StoreConfigMapKey),
		Namespace:      namespace,
	}, nil
}

type IntentsHolder struct {
	accumulatingStore map[SourceDestPair]time.Time
	sinceLastGetStore map[SourceDestPair]time.Time
	lock              sync.Mutex
	client            client.Client
	config            IntentsHolderConfig
}

func NewIntentsHolder(client client.Client, config IntentsHolderConfig) *IntentsHolder {
	return &IntentsHolder{
		accumulatingStore: make(map[SourceDestPair]time.Time),
		sinceLastGetStore: make(map[SourceDestPair]time.Time),
		lock:              sync.Mutex{},
		client:            client,
		config:            config,
	}
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.accumulatingStore = make(map[SourceDestPair]time.Time)
}

func (i *IntentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity, newTimestamp time.Time) {
	i.lock.Lock()
	defer i.lock.Unlock()

	pair := SourceDestPair{Source: srcService, Destination: dstService}
	currentTimestamp, alreadyExists := i.accumulatingStore[pair]
	if !alreadyExists || newTimestamp.After(currentTimestamp) {
		i.accumulatingStore[pair] = newTimestamp
		i.sinceLastGetStore[pair] = newTimestamp
	}

}

func (i *IntentsHolder) GetIntents(namespaces []string) []DiscoveredIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.getIntentsFromStore(i.accumulatingStore, namespaces...)
}

func (i *IntentsHolder) GetNewIntentsSinceLastGet() []DiscoveredIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	intents := i.getIntentsFromStore(i.sinceLastGetStore)
	i.sinceLastGetStore = make(map[SourceDestPair]time.Time)
	return intents
}

func (i *IntentsHolder) getIntentsFromStore(store map[SourceDestPair]time.Time, namespaces ...string) []DiscoveredIntent {
	namespacesSet := goset.FromSlice(namespaces)
	result := make([]DiscoveredIntent, 0)
	for pair, timestamp := range store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}

		result = append(result, DiscoveredIntent{
			Source:      pair.Source,
			Destination: pair.Destination,
			Timestamp:   timestamp,
		})
	}
	return result
}

func groupDestinationsBySource(discoveredIntents []DiscoveredIntent) map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity {
	serviceMap := make(map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity, 0)
	for _, intents := range discoveredIntents {
		if _, ok := serviceMap[intents.Source]; !ok {
			serviceMap[intents.Source] = make([]model.OtterizeServiceIdentity, 0)
		}

		serviceMap[intents.Source] = append(serviceMap[intents.Source], intents.Destination)
	}
	return serviceMap
}
