package resolvers

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
	"sync"
)

type intentsHolder struct {
	store map[string]*goset.Set[model.ServiceIdentity]
	lock  sync.Mutex
}

func NewIntentsHolder() *intentsHolder {
	return &intentsHolder{
		store: make(map[string]*goset.Set[model.ServiceIdentity]),
		lock:  sync.Mutex{},
	}
}

func (i *intentsHolder) AddIntent(srcService model.ServiceIdentity, dstService model.ServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	set, ok := i.store[srcService.Name]
	if !ok {
		set = goset.NewSet[model.ServiceIdentity]()
		i.store[srcService.Name] = set
	}
	namespace := ""
	if srcService.Namespace != dstService.Namespace {
		// namespace is only needed if it's a connection between different namespaces.
		namespace = dstService.Namespace
	}
	set.Add(model.ServiceIdentity{Name: dstService.Name, Namespace: namespace})
}

func (i *intentsHolder) GetIntentsPerService() map[string][]model.ServiceIdentity {
	i.lock.Lock()
	defer i.lock.Unlock()
	result := make(map[string][]model.ServiceIdentity)
	for service, intents := range i.store {
		result[service] = intents.Items()
	}
	return result
}

const crdTemplate = `{{range $key, $value := .}}---
apiVersion: k8s.otterize.com/v1
kind: ClientIntents
metadata:
  name: {{$key}}
spec:
  service:
    name: {{$key}}
    calls:{{range $service := $value}}
      - name: {{$service.Name}}{{if ne $service.Namespace "" }}
        namespace: {{$service.Namespace}}{{end}}{{end}}
{{end}}`
