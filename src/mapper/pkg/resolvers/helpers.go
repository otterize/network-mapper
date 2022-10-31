package resolvers

import (
	"encoding/json"
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"os"
	"sort"
	"sync"
)

type intentsHolderStore map[model.OtterizeServiceIdentity]map[string]*goset.Set[model.OtterizeServiceIdentity]

func (s intentsHolderStore) MarshalJSON() ([]byte, error) {
	// OtterizeServiceIdentity cannot be serialized as map key in JSON, because it is represented as a map itself
	// therefore, we serialize the store as a slice of [Key, Value] "tuples"
	return json.Marshal(lo.ToPairs(s))
}

func (s intentsHolderStore) UnmarshalJSON(b []byte) error {
	var pairs []lo.Entry[model.OtterizeServiceIdentity, map[string]*goset.Set[model.OtterizeServiceIdentity]]
	err := json.Unmarshal(b, &pairs)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		s[pair.Key] = pair.Value
	}
	return nil
}

type intentsHolder struct {
	store intentsHolderStore
	lock  sync.Mutex
}

func NewIntentsHolder() *intentsHolder {
	return &intentsHolder{
		store: make(intentsHolderStore),
		lock:  sync.Mutex{},
	}
}

func (i *intentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store = make(intentsHolderStore)
}

func (i *intentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	intentMap, ok := i.store[srcService]
	if !ok {
		intentMap = make(map[string]*goset.Set[model.OtterizeServiceIdentity])
		i.store[srcService] = intentMap
	}
	namespace := ""
	if srcService.Namespace != dstService.Namespace {
		// namespace is only needed if it's a connection between different namespaces.
		namespace = dstService.Namespace
	}
	if intentMap[srcService.Namespace] == nil {
		intentMap[srcService.Namespace] = goset.NewSet[model.OtterizeServiceIdentity]()
	}
	intentMap[srcService.Namespace].Add(model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: namespace})
}

func (i *intentsHolder) GetIntentsPerService(namespaces []string) map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity {
	i.lock.Lock()
	defer i.lock.Unlock()
	result := make(map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity)
	for service, intentsMap := range i.store {
		serviceIntents := goset.NewSet[model.OtterizeServiceIdentity]()
		for namespace, serviceIdentities := range intentsMap {
			if len(namespaces) != 0 && !lo.Contains(namespaces, namespace) {
				continue
			}
			serviceIntents.Add(serviceIdentities.Items()...)
		}
		intentsSlice := serviceIntents.Items()
		// sorting the intents so results will be consistent
		sort.Slice(intentsSlice, func(i, j int) bool {
			// Primary sort by name
			if intentsSlice[i].Name != intentsSlice[j].Name {
				return intentsSlice[i].Name < intentsSlice[j].Name
			}
			// Secondary sort by namespace
			return intentsSlice[i].Namespace < intentsSlice[j].Namespace
		})
		if len(intentsSlice) != 0 {
			result[service] = intentsSlice
		}
	}
	return result
}

func (i *intentsHolder) WriteStore(path string) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	jsonBytes, err := json.MarshalIndent(i.store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonBytes, 0600)
}

func (i *intentsHolder) LoadStore(path string) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, &i.store)
}
