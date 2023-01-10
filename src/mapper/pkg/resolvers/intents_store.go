package resolvers

import (
	"encoding/json"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"time"
)

type serviceToDestToTimestamp map[model.OtterizeServiceIdentity]map[model.OtterizeServiceIdentity]time.Time

type DiscoveredIntent struct {
	Source      model.OtterizeServiceIdentity
	Destination model.OtterizeServiceIdentity
	Timestamp   time.Time
}

type intentsHolderStore struct {
	serviceMap serviceToDestToTimestamp
}

func NewIntentsHolderStore() intentsHolderStore {
	return intentsHolderStore{
		serviceMap: make(serviceToDestToTimestamp),
	}
}

func (s intentsHolderStore) MarshalJSON() ([]byte, error) {
	// OtterizeServiceIdentity cannot be serialized as map key in JSON, because it is represented as a map itself
	// therefore, we serialize the store as a slice of [Key, Value] "tuples"
	sourceToIntents := make(map[model.OtterizeServiceIdentity][]DiscoveredIntent)
	for source, destinations := range s.serviceMap {
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

func (s intentsHolderStore) UnmarshalJSON(b []byte) error {
	var pairs []lo.Entry[model.OtterizeServiceIdentity, []DiscoveredIntent]
	err := json.Unmarshal(b, &pairs)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		src := pair.Key
		s.serviceMap[src] = make(map[model.OtterizeServiceIdentity]time.Time)
		for _, intent := range pair.Value {
			s.serviceMap[src][intent.Destination] = intent.Timestamp
		}
	}
	return nil
}

func (s *intentsHolderStore) Update(src model.OtterizeServiceIdentity, dest model.OtterizeServiceIdentity, newTimestamp time.Time) bool {
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

func (s *intentsHolderStore) GetIntentsByNamespace(namespaces []string) serviceToDestToTimestamp {
	result := make(serviceToDestToTimestamp)
	for service := range s.serviceMap {
		if !lo.Contains(namespaces, service.Namespace) {
			continue
		}

		result[service] = make(map[model.OtterizeServiceIdentity]time.Time)
		for dest, timestamp := range s.serviceMap[service] {
			result[service][dest] = timestamp
		}
	}
	return result

}

func (s *intentsHolderStore) GetAllIntents() serviceToDestToTimestamp {
	result := make(serviceToDestToTimestamp)
	for service := range s.serviceMap {
		result[service] = make(map[model.OtterizeServiceIdentity]time.Time)
		for dest, timestamp := range s.serviceMap[service] {
			result[service][dest] = timestamp
		}
	}
	return result
}

func (s *intentsHolderStore) Reset() {
	s.serviceMap = make(serviceToDestToTimestamp)
}
