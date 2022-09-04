package resolvers

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/mapper/pkg/graph/model"
	"sort"
	"sync"
)

type intentsHolder struct {
	store map[string]*goset.Set[model.OtterizeServiceIdentity]
	lock  sync.Mutex
}

func NewIntentsHolder() *intentsHolder {
	return &intentsHolder{
		store: make(map[string]*goset.Set[model.OtterizeServiceIdentity]),
		lock:  sync.Mutex{},
	}
}

func (i *intentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	set, ok := i.store[srcService.Name]
	if !ok {
		set = goset.NewSet[model.OtterizeServiceIdentity]()
		i.store[srcService.Name] = set
	}
	namespace := ""
	if srcService.Namespace != dstService.Namespace {
		// namespace is only needed if it's a connection between different namespaces.
		namespace = dstService.Namespace
	}
	set.Add(model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: namespace})
}

func (i *intentsHolder) GetIntentsPerService() map[string][]model.OtterizeServiceIdentity {
	i.lock.Lock()
	defer i.lock.Unlock()
	result := make(map[string][]model.OtterizeServiceIdentity)
	for service, intents := range i.store {
		// sorting the intents so results will be consistent
		intentsSlice := intents.Items()
		sort.Slice(intentsSlice, func(i, j int) bool {
			// Primary sort by name
			if intentsSlice[i].Name != intentsSlice[j].Name {
				return intentsSlice[i].Name < intentsSlice[j].Name
			}
			// Secondary sort by namespace
			return intentsSlice[i].Namespace < intentsSlice[j].Namespace
		})
		result[service] = intentsSlice
	}
	return result
}
