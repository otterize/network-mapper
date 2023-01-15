package resolvers

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/amit7itz/goset"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/shared/kubeutils"
	"github.com/spf13/viper"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sync"
	"time"
)

type SourceDestPair struct {
	Source      model.OtterizeServiceIdentity
	Destination model.OtterizeServiceIdentity
}

type DiscoveredIntent struct {
	Source      model.OtterizeServiceIdentity `json:"source"`
	Destination model.OtterizeServiceIdentity `json:"destination"`
	Timestamp   time.Time                     `json:"timestamp"`
}

type IntentsHolderConfig struct {
	StoreConfigMap string
	Namespace      string
}

func IntentsHolderConfigFromViper() (IntentsHolderConfig, error) {
	namespace := viper.GetString(config.NamespaceKey)
	if namespace != "" {
		var err error
		namespace, err = kubeutils.GetCurrentNamespace()
		if err != nil {
			return IntentsHolderConfig{}, fmt.Errorf("could not deduce the store's configmap namespace: %w", err)
		}
	}
	return IntentsHolderConfig{
		StoreConfigMap: viper.GetString(config.StoreConfigMapKey),
		Namespace:      namespace,
	}, nil
}

type IntentsHolder struct {
	store             map[SourceDestPair]time.Time
	lock              sync.Mutex
	client            client.Client
	config            IntentsHolderConfig
	lastIntentsUpdate time.Time
}

func NewIntentsHolder(client client.Client, config IntentsHolderConfig) *IntentsHolder {
	return &IntentsHolder{
		store:  newIntentsStore(nil),
		lock:   sync.Mutex{},
		client: client,
		config: config,
	}
}

func newIntentsStore(intents []DiscoveredIntent) map[SourceDestPair]time.Time {
	if intents == nil {
		return make(map[SourceDestPair]time.Time)
	}

	result := make(map[SourceDestPair]time.Time)
	for _, intent := range intents {
		result[SourceDestPair{Source: intent.Source, Destination: intent.Destination}] = intent.Timestamp
	}
	return result
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store = make(map[SourceDestPair]time.Time)
}

func (i *IntentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity, newTimestamp time.Time) {
	i.lock.Lock()
	defer i.lock.Unlock()

	pair := SourceDestPair{Source: srcService, Destination: dstService}
	currentTimestamp, alreadyExists := i.store[pair]
	if !alreadyExists || newTimestamp.After(currentTimestamp) {
		i.store[pair] = newTimestamp
		i.lastIntentsUpdate = time.Now()
	}
}

func (i *IntentsHolder) LastIntentsUpdate() time.Time {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.lastIntentsUpdate
}

func (i *IntentsHolder) GetIntents(namespaces []string) []DiscoveredIntent {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.getIntents(namespaces)
}

func (i *IntentsHolder) getIntents(namespaces []string) []DiscoveredIntent {
	namespacesSet := goset.FromSlice(namespaces)
	result := make([]DiscoveredIntent, 0)
	for pair, timestamp := range i.store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(pair.Source.Namespace) {
			continue
		}

		result = append(result, DiscoveredIntent{
			Source:      pair.Source,
			Destination: pair.Destination,
			Timestamp:   timestamp,
		})
	}
	return result
}

func (i *IntentsHolder) WriteStore(ctx context.Context) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	intents := i.getIntents(nil)
	jsonBytes, err := json.Marshal(intents)
	if err != nil {
		return err
	}
	var compressedJson bytes.Buffer
	writer := gzip.NewWriter(&compressedJson)
	_, err = writer.Write(jsonBytes)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: i.config.StoreConfigMap, Namespace: i.config.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, i.client, configmap, func() error {
		configmap.Data = nil
		configmap.BinaryData = map[string][]byte{"store": compressedJson.Bytes()}
		return nil
	})
	return err
}

func (i *IntentsHolder) LoadStore(ctx context.Context) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	configmap := &corev1.ConfigMap{}
	err := i.client.Get(ctx, types.NamespacedName{Name: i.config.StoreConfigMap, Namespace: i.config.Namespace}, configmap)
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	reader, err := gzip.NewReader(bytes.NewReader(configmap.BinaryData["store"]))
	if err != nil {
		return err
	}
	decompressedJson, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var intents []DiscoveredIntent
	err = json.Unmarshal(decompressedJson, &intents)
	if err != nil {
		return err
	}

	i.store = newIntentsStore(intents)
	return nil
}

func groupDestinationsBySource(discoveredIntents []DiscoveredIntent) map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity {
	serviceMap := make(map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity, 0)
	for _, intents := range discoveredIntents {
		if _, ok := serviceMap[intents.Source]; !ok {
			serviceMap[intents.Source] = make([]model.OtterizeServiceIdentity, 0)
		}

		serviceMap[intents.Source] = append(serviceMap[intents.Source], intents.Destination)
	}
	return serviceMap
}
