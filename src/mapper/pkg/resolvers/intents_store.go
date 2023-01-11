package resolvers

import (
	"encoding/json"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"sort"
	"time"
)

type ServiceToDestToTimestamp map[model.OtterizeServiceIdentity]map[model.OtterizeServiceIdentity]time.Time

type DiscoveredIntent struct {
	Source      model.OtterizeServiceIdentity
	Destination model.OtterizeServiceIdentity
	Timestamp   time.Time
}

type IntentsHolderStore struct {
	serviceMap ServiceToDestToTimestamp
}

func NewIntentsHolderStore() IntentsHolderStore {
	return IntentsHolderStore{
		serviceMap: make(ServiceToDestToTimestamp),
	}
}

func (serviceMap ServiceToDestToTimestamp) MarshalJSON() ([]byte, error) {
	// OtterizeServiceIdentity cannot be serialized as map key in JSON, because it is represented as a map itself
	// therefore, we serialize the store as a slice of [Key, Value] "tuples"
	sourceToIntents := make(map[model.OtterizeServiceIdentity][]DiscoveredIntent)
	for source, destinations := range serviceMap {
		var intents []DiscoveredIntent
		for destination, timestamp := range destinations {
			intents = append(intents, DiscoveredIntent{
				Source:      source,
				Destination: destination,
				Timestamp:   timestamp,
			})
		}
		sourceToIntents[source] = intents
	}
	return json.Marshal(lo.ToPairs(sourceToIntents))
}

func (serviceMap ServiceToDestToTimestamp) UnmarshalJSON(b []byte) error {
	var pairs []lo.Entry[model.OtterizeServiceIdentity, []DiscoveredIntent]
	err := json.Unmarshal(b, &pairs)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		src := pair.Key
		serviceMap[src] = make(map[model.OtterizeServiceIdentity]time.Time)
		for _, intent := range pair.Value {
			serviceMap[src][intent.Destination] = intent.Timestamp
		}
	}
	return nil
}

func (s *IntentsHolderStore) MarshalJSON() ([]byte, error) {
	return s.serviceMap.MarshalJSON()
}

func (s *IntentsHolderStore) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &s.serviceMap)
}

func (s *IntentsHolderStore) Update(src model.OtterizeServiceIdentity, dest model.OtterizeServiceIdentity, newTimestamp time.Time) bool {
	updated := false
	if _, ok := s.serviceMap[src]; !ok {
		s.serviceMap[src] = make(map[model.OtterizeServiceIdentity]time.Time)
		updated = true
	}

	timestamp, ok := s.serviceMap[src][dest]
	if !ok {
		s.serviceMap[src][dest] = newTimestamp
		updated = true
	}

	if newTimestamp.After(timestamp) {
		s.serviceMap[src][dest] = newTimestamp
		updated = true
	}

	return updated
}

func (s *IntentsHolderStore) GetIntents(namespaces []string) ServiceToDestToTimestamp {
	result := make(ServiceToDestToTimestamp)
	for service := range s.serviceMap {
		if !shouldGetIntentsForService(namespaces, service) {
			continue
		}

		result[service] = make(map[model.OtterizeServiceIdentity]time.Time)
		pairs := lo.ToPairs(s.serviceMap[service])
		sort.Slice(pairs, func(i, j int) bool {
			return compareOrderedServiceNames(pairs[i].Key, pairs[j].Key)
		})

		for _, pair := range pairs {
			dest := pair.Key
			timestamp := pair.Value
			result[service][dest] = timestamp
		}
	}
	return result
}

func shouldGetIntentsForService(namespaces []string, service model.OtterizeServiceIdentity) bool {
	return len(namespaces) == 0 || lo.Contains(namespaces, service.Namespace)
}

func compareOrderedServiceNames(a model.OtterizeServiceIdentity, b model.OtterizeServiceIdentity) bool {
	// Primary sort by name
	if a.Name != b.Name {
		return a.Name < b.Name
	}

	// Secondary sort by namespace
	return a.Namespace < b.Namespace
}

func (s *IntentsHolderStore) Reset() {
	s.serviceMap = make(ServiceToDestToTimestamp)
}
