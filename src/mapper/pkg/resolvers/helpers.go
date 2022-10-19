package resolvers

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"sort"
	"sync"
)

type intentsHolder struct {
	store map[model.OtterizeServiceIdentity]map[string][]model.OtterizeServiceIdentity
	lock  sync.Mutex
}

func NewIntentsHolder() *intentsHolder {
	return &intentsHolder{
		store: make(map[model.OtterizeServiceIdentity]map[string][]model.OtterizeServiceIdentity),
		lock:  sync.Mutex{},
	}
}

func (i *intentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store = make(map[model.OtterizeServiceIdentity]map[string][]model.OtterizeServiceIdentity)
}

func (i *intentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	intentMap, ok := i.store[srcService]
	if !ok {
		intentMap = make(map[string][]model.OtterizeServiceIdentity)
		i.store[srcService] = intentMap
	}
	namespace := ""
	if srcService.Namespace != dstService.Namespace {
		// namespace is only needed if it's a connection between different namespaces.
		namespace = dstService.Namespace
	}
	intentMap[srcService.Namespace] = append(intentMap[srcService.Namespace],
		model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: namespace})
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

			serviceIntents.Add(serviceIdentities...)
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
