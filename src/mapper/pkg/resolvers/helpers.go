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
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sort"
	"sync"
)

type intentsHolderStore map[model.OtterizeServiceIdentity]*goset.Set[model.OtterizeServiceIdentity]

func (s intentsHolderStore) MarshalJSON() ([]byte, error) {
	// OtterizeServiceIdentity cannot be serialized as map key in JSON, because it is represented as a map itself
	// therefore, we serialize the store as a slice of [Key, Value] "tuples"
	return json.Marshal(lo.ToPairs(s))
}

func (s intentsHolderStore) UnmarshalJSON(b []byte) error {
	var pairs []lo.Entry[model.OtterizeServiceIdentity, *goset.Set[model.OtterizeServiceIdentity]]
	err := json.Unmarshal(b, &pairs)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		s[pair.Key] = pair.Value
	}
	return nil
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
	store  intentsHolderStore
	lock   sync.Mutex
	client client.Client
	config IntentsHolderConfig
}

func NewIntentsHolder(client client.Client, config IntentsHolderConfig) *IntentsHolder {
	return &IntentsHolder{
		store:  make(intentsHolderStore),
		lock:   sync.Mutex{},
		client: client,
		config: config,
	}
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store = make(intentsHolderStore)
}

func (i *IntentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	namespace := ""
	if srcService.Namespace != dstService.Namespace {
		// namespace is only needed if it's a connection between different namespaces.
		namespace = dstService.Namespace
	}
	intents, ok := i.store[srcService]
	if !ok {
		intents = goset.NewSet[model.OtterizeServiceIdentity]()
		i.store[srcService] = intents
	}
	intents.Add(model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: namespace})
}

func (i *IntentsHolder) GetIntentsPerService(namespaces []string) map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity {
	i.lock.Lock()
	defer i.lock.Unlock()
	namespacesSet := goset.FromSlice(namespaces)
	result := make(map[model.OtterizeServiceIdentity][]model.OtterizeServiceIdentity)
	for service, intents := range i.store {
		if !namespacesSet.IsEmpty() && !namespacesSet.Contains(service.Namespace) {
			continue
		}
		intentsSlice := intents.Items()
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

func (i *IntentsHolder) WriteStore(ctx context.Context) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	jsonBytes, err := json.Marshal(i.store)
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
	return json.Unmarshal(decompressedJson, &i.store)
}
