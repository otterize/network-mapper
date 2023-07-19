package intentsstore

import (
	"encoding/json"
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/types"
	"strings"
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

func (ti *TimestampedIntent) containsExcludedLabels(excludedLabelsMap map[string]string) bool {
	for _, podLabel := range ti.Intent.Client.Labels {
		value, ok := excludedLabelsMap[podLabel.Key]
		if ok {
			if value == podLabel.Value {
				return true
			}
		}
	}

	for _, podLabel := range ti.Intent.Server.Labels {
		value, ok := excludedLabelsMap[podLabel.Key]
		if ok {
			if value == podLabel.Value {
				return true
			}
		}
	}

	return false
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.accumulatingStore = make(IntentsStore)
}

func mergeKafkaTopics(existingTopics []model.KafkaConfig, newTopics []model.KafkaConfig) []model.KafkaConfig {
	existingTopicsByName := lo.SliceToMap(existingTopics, func(topic model.KafkaConfig) (string, *model.KafkaConfig) {
		return topic.Name, &topic
	})

	for _, newTopic := range newTopics {
		existingTopic, ok := existingTopicsByName[newTopic.Name]
		if ok {
			existingTopic.Operations = lo.Uniq(append(existingTopic.Operations, newTopic.Operations...))
		} else {
			existingTopicsByName[newTopic.Name] = &newTopic
		}
	}

	var res []model.KafkaConfig

	for _, topic := range existingTopicsByName {
		res = append(res, *topic)
	}

	return res
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

	// Replace labels with latest
	existingIntent.Intent.Client.Labels = intent.Client.Labels
	existingIntent.Intent.Server.Labels = intent.Server.Labels

	store[key] = existingIntent
}

func (i *IntentsHolder) AddIntent(newTimestamp time.Time, intent model.Intent) {
	if config.ExcludedNamespaces().Contains(intent.Client.Namespace) || config.ExcludedNamespaces().Contains(intent.Server.Namespace) {
		return
	}

	i.lock.Lock()
	defer i.lock.Unlock()

	i.addIntentToStore(i.accumulatingStore, newTimestamp, intent)
	i.addIntentToStore(i.sinceLastGetStore, newTimestamp, intent)
}

func (i *IntentsHolder) GetIntents(
	namespaces []string,
	includeLabels []string,
	excludeServiceWithLabels []string,
	includeAllLabels bool,
	serverName string,
) ([]TimestampedIntent, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	result, err := i.getIntentsFromStore(
		i.accumulatingStore,
		namespaces,
		includeLabels,
		excludeServiceWithLabels,
		includeAllLabels,
		serverName)

	if err != nil {
		return []TimestampedIntent{}, err
	}
	return result, nil
}

func (i *IntentsHolder) GetNewIntentsSinceLastGet() []TimestampedIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	intents, _ := i.getIntentsFromStore(
		i.sinceLastGetStore,
		nil,
		nil,
		nil,
		false,
		"")

	i.sinceLastGetStore = make(IntentsStore)
	return intents
}

func (i *IntentsHolder) getIntentsFromStore(
	store IntentsStore,
	namespaces, includeLabels, excludeServiceWithLabels []string,
	includeAllLabels bool,
	serverName string,
) ([]TimestampedIntent, error) {
	namespacesSet := goset.FromSlice(namespaces)
	includeLabelsSet := goset.FromSlice(includeLabels)
	result := make([]TimestampedIntent, 0)
	excludedLabelsMap := lo.SliceToMap(excludeServiceWithLabels, func(label string) (key, value string) {
		labelSlice := strings.Split(label, "=")
		if len(labelSlice) == 1 {
			return label, ""
		}
		return labelSlice[0], labelSlice[1]
	})

	for pair, intent := range store {
		intentCopy, err := getIntentDeepCopy(intent)
		if err != nil {
			return result, err
		}

		if len(excludeServiceWithLabels) != 0 && intentCopy.containsExcludedLabels(excludedLabelsMap) {
			continue
		}

		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}

		if serverName != "" && intent.Intent.Server.Name != serverName {
			continue
		}

		if !includeAllLabels {
			labelsFilter := func(labels []model.PodLabel) []model.PodLabel {
				return lo.Filter(labels, func(label model.PodLabel, _ int) bool {
					return includeLabelsSet.Contains(label.Key)
				})
			}
			intentCopy.Intent.Client.Labels = labelsFilter(intentCopy.Intent.Client.Labels)
			intentCopy.Intent.Server.Labels = labelsFilter(intentCopy.Intent.Server.Labels)
		}

		result = append(result, intent)
	}
	return result, nil
}

func getIntentDeepCopy(intent TimestampedIntent) (TimestampedIntent, error) {
	intentCopy := TimestampedIntent{}
	intentJSON, err := json.Marshal(intent)
	if err != nil {
		return TimestampedIntent{}, err
	}
	if err = json.Unmarshal(intentJSON, &intentCopy); err != nil {
		return TimestampedIntent{}, err
	}
	return intentCopy, nil
}

func dedupeServiceIntentsDests(dests []model.OtterizeServiceIdentity) []model.OtterizeServiceIdentity {
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
			Intents: dedupeServiceIntentsDests(serviceIntents.Intents),
		}
	})
}
