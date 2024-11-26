package azureintentsholder

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

type key struct {
	client types.NamespacedName
	scope  string
}

type AzureIntentsHolder struct {
	intents   map[key]model.AzureOperation
	lock      sync.Mutex
	callbacks []Callback
}

type Callback func(context.Context, []model.AzureOperation)

func New() *AzureIntentsHolder {
	return &AzureIntentsHolder{
		intents: make(map[key]model.AzureOperation),
	}
}

func (h *AzureIntentsHolder) RegisterNotifyIntents(callback Callback) {
	h.callbacks = append(h.callbacks, callback)
}

func (h *AzureIntentsHolder) AddOperation(serviceId model.OtterizeServiceIdentity, op model.AzureOperation) {
	h.lock.Lock()
	defer h.lock.Unlock()

	k := key{
		client: types.NamespacedName{
			Namespace: serviceId.Namespace,
			Name:      serviceId.Name,
		},
		scope: op.Scope,
	}

	_, found := h.intents[k]

	if !found {
		h.intents[k] = model.AzureOperation{
			Scope:        op.Scope,
			Actions:      op.Actions,
			DataActions:  op.DataActions,
			PodName:      serviceId.Name,
			PodNamespace: serviceId.Namespace,
		}
	} else {
		h.intents[k] = model.AzureOperation{
			Scope:        op.Scope,
			Actions:      lo.Union(h.intents[k].Actions, op.Actions),
			DataActions:  lo.Union(h.intents[k].DataActions, op.DataActions),
			PodName:      h.intents[k].PodName,
			PodNamespace: h.intents[k].PodNamespace,
		}
	}
}

func (h *AzureIntentsHolder) PeriodicIntentsUpload(ctx context.Context, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			if len(h.callbacks) == 0 {
				continue
			}

			intents := h.getOperations()

			if len(intents) == 0 {
				continue
			}

			for _, callback := range h.callbacks {
				callback(ctx, intents)
			}
		}
	}
}

func (h *AzureIntentsHolder) getOperations() []model.AzureOperation {
	h.lock.Lock()
	defer h.lock.Unlock()

	intents := lo.Values(h.intents)
	h.intents = make(map[key]model.AzureOperation)

	return intents
}
