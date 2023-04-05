package intentsstore

import (
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

type IntentsStoreKey struct {
	Source      types.NamespacedName
	Destination types.NamespacedName
	Type        model.IntentType
}

type TimestampedIntent struct {
	Timestamp time.Time
	Intent    model.Intent
}

type IntentsStore map[IntentsStoreKey]TimestampedIntent

type IntentsHolder struct {
	accumulatingStore IntentsStore
	sinceLastGetStore IntentsStore
	lock              sync.Mutex
}

func NewIntentsHolder() *IntentsHolder {
	return &IntentsHolder{
		accumulatingStore: make(IntentsStore),
		sinceLastGetStore: make(IntentsStore),
		lock:              sync.Mutex{},
	}
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.accumulatingStore = make(IntentsStore)
}

func mergeKafkaTopics(existingTopics []model.KafkaConfig, newTopics []model.KafkaConfig) []model.KafkaConfig {
	existingTopicsByName := lo.SliceToMap(existingTopics, func(topic model.KafkaConfig) (string, model.KafkaConfig) {
		return topic.Name, topic
	})
	for _, newTopic := range newTopics {
		existingTopic, ok := existingTopicsByName[newTopic.Name]
		if ok {
			existingTopic.Operations = lo.Uniq(append(existingTopic.Operations, newTopic.Operations...))
		} else {
			existingTopicsByName[newTopic.Name] = newTopic
		}
	}

	return lo.Values(existingTopicsByName)
}

func mergeHTTPResources(existingResources, newResources []model.HTTPResource) []model.HTTPResource {
	existingResourcesMap := lo.SliceToMap(existingResources, func(resource model.HTTPResource) (string, []model.HTTPMethod) {
		return resource.Path, resource.Methods
	})
	newResourcesMap := lo.SliceToMap(newResources, func(resource model.HTTPResource) (string, []model.HTTPMethod) {
		return resource.Path, resource.Methods
	})
	// Merge methods for existing resources, add path:methods key-value for non-existing ones
	for path, methods := range newResourcesMap {
		currMethods, ok := existingResourcesMap[path]
		if !ok {
			existingResourcesMap[path] = methods
		} else {
			for _, method := range methods {
				if !slices.Contains(currMethods, method) {
					currMethods = append(currMethods, method)
				}
			}
			existingResourcesMap[path] = currMethods
		}
	}

	return lo.MapToSlice(existingResourcesMap, func(path string, methods []model.HTTPMethod) model.HTTPResource {
		return model.HTTPResource{
			Path:    path,
			Methods: methods,
		}
	})
}

func (i *IntentsHolder) addIntentToStore(store IntentsStore, newTimestamp time.Time, intent model.Intent) {
	key := IntentsStoreKey{
		Source:      intent.Client.AsNamespacedName(),
		Destination: intent.Server.AsNamespacedName(),
		Type:        lo.FromPtr(intent.Type),
	}

	existingIntent, ok := store[key]
	if !ok {
		store[key] = TimestampedIntent{
			Timestamp: newTimestamp,
			Intent:    intent,
		}
		return
	}

	// merge into existing intent
	if newTimestamp.After(existingIntent.Timestamp) {
		existingIntent.Timestamp = newTimestamp
	}
	existingIntent.Intent.KafkaTopics = mergeKafkaTopics(existingIntent.Intent.KafkaTopics, intent.KafkaTopics)
	existingIntent.Intent.HTTPResources = mergeHTTPResources(existingIntent.Intent.HTTPResources, intent.HTTPResources)

	store[key] = existingIntent
}

func (i *IntentsHolder) AddIntent(newTimestamp time.Time, intent model.Intent) {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.addIntentToStore(i.accumulatingStore, newTimestamp, intent)
	i.addIntentToStore(i.sinceLastGetStore, newTimestamp, intent)
}

func (i *IntentsHolder) GetIntents(namespaces []string, includeLabels []string, includeAllLabels bool) []TimestampedIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.getIntentsFromStore(i.accumulatingStore, namespaces, includeLabels, includeAllLabels)
}

func (i *IntentsHolder) GetNewIntentsSinceLastGet() []TimestampedIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	intents := i.getIntentsFromStore(i.sinceLastGetStore, nil, nil, false)
	i.sinceLastGetStore = make(IntentsStore)
	return intents
}

func (i *IntentsHolder) getIntentsFromStore(store IntentsStore, namespaces []string, includeLabels []string, includeAllLabels bool) []TimestampedIntent {
	namespacesSet := goset.FromSlice(namespaces)
	includeLabelsSet := goset.FromSlice(includeLabels)
	result := make([]TimestampedIntent, 0)
	for pair, intent := range store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}

		if !includeAllLabels {
			labelsFilter := func(labels []model.PodLabel) []model.PodLabel {
				return lo.Filter(labels, func(label model.PodLabel, _ int) bool {
					return includeLabelsSet.Contains(label.Key)
				})
			}
			intent.Intent.Client.Labels = labelsFilter(intent.Intent.Client.Labels)
			intent.Intent.Server.Labels = labelsFilter(intent.Intent.Server.Labels)
		}

		result = append(result, intent)
	}
	return result
}

func dedupServiceIntentsDests(dests []model.OtterizeServiceIdentity) []model.OtterizeServiceIdentity {
	return lo.UniqBy(dests, func(dest model.OtterizeServiceIdentity) types.NamespacedName {
		return dest.AsNamespacedName()
	})
}

func GroupIntentsBySource(intents []TimestampedIntent) []model.ServiceIntents {
	intentsBySource := make(map[types.NamespacedName]*model.ServiceIntents, 0)
	for _, intent := range intents {
		srcIdentity := intent.Intent.Client.AsNamespacedName()
		if _, ok := intentsBySource[srcIdentity]; !ok {
			intentsBySource[srcIdentity] = &model.ServiceIntents{
				Client:  intent.Intent.Client,
				Intents: make([]model.OtterizeServiceIdentity, 0),
			}
		}

		intentsBySource[srcIdentity].Intents = append(intentsBySource[srcIdentity].Intents, *intent.Intent.Server)
	}

	return lo.Map(lo.Values(intentsBySource), func(serviceIntents *model.ServiceIntents, _ int) model.ServiceIntents {
		return model.ServiceIntents{
			Client:  serviceIntents.Client,
			Intents: dedupServiceIntentsDests(serviceIntents.Intents),
		}
	})
}
