package resolvers

import (
	"github.com/amit7itz/goset"
	"sync"
)

type intentsHolder struct {
	store map[string]*goset.Set[string]
	lock  sync.Mutex
}

func NewIntentsHolder() *intentsHolder {
	return &intentsHolder{
		store: make(map[string]*goset.Set[string]),
		lock:  sync.Mutex{},
	}
}

func (i *intentsHolder) AddIntent(srcService string, dstService string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	set, ok := i.store[srcService]
	if !ok {
		set = goset.NewSet[string]()
		i.store[srcService] = set
	}
	set.Add(dstService)
}

func (i *intentsHolder) GetIntentsPerService() map[string][]string {
	i.lock.Lock()
	defer i.lock.Unlock()
	result := make(map[string][]string)
	for service, intents := range i.store {
		result[service] = intents.Items()
	}
	return result
}
