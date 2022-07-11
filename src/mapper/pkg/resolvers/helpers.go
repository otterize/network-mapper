package resolvers

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/otternose/mapper/pkg/reconcilers"
	"sync"
)

type intentsHolder struct {
	store map[string]*goset.Set[reconcilers.ServiceIdentity]
	lock  sync.Mutex
}

func NewIntentsHolder() *intentsHolder {
	return &intentsHolder{
		store: make(map[string]*goset.Set[reconcilers.ServiceIdentity]),
		lock:  sync.Mutex{},
	}
}

func (i *intentsHolder) AddIntent(srcService reconcilers.ServiceIdentity, dstService reconcilers.ServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	set, ok := i.store[srcService.Name]
	if !ok {
		set = goset.NewSet[reconcilers.ServiceIdentity]()
		i.store[srcService.Name] = set
	}
	if srcService.Namespace == dstService.Namespace {
		set.Add(reconcilers.ServiceIdentity{Name: dstService.Name, Namespace: ""})
	} else {
		set.Add(dstService)
	}
}

func (i *intentsHolder) GetIntentsPerService() map[string][]reconcilers.ServiceIdentity {
	i.lock.Lock()
	defer i.lock.Unlock()
	result := make(map[string][]reconcilers.ServiceIdentity)
	for service, intents := range i.store {
		result[service] = intents.Items()
	}
	return result
}
