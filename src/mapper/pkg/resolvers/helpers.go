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

type intentsHolderStore map[model.OtterizeServiceIdentity]map[string]*goset.Set[model.OtterizeServiceIdentity]

func (s intentsHolderStore) MarshalJSON() ([]byte, error) {
	// OtterizeServiceIdentity cannot be serialized as map key in JSON, because it is represented as a map itself
	// therefore, we serialize the store as a slice of [Key, Value] "tuples"
	return json.Marshal(lo.ToPairs(s))
}

func (s intentsHolderStore) UnmarshalJSON(b []byte) error {
	var pairs []lo.Entry[model.OtterizeServiceIdentity, map[string]*goset.Set[model.OtterizeServiceIdentity]]
	err := json.Unmarshal(b, &pairs)
	if err != nil {
		return err
	}
	for _, pair := range pairs {
		s[pair.Key] = pair.Value
	}
	return nil
}

type intentsHolder struct {
	store  intentsHolderStore
	lock   sync.Mutex
	client client.Client
}

func NewIntentsHolder(client client.Client) *intentsHolder {
	return &intentsHolder{
		store:  make(intentsHolderStore),
		lock:   sync.Mutex{},
		client: client,
	}
}

func (i *intentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store = make(intentsHolderStore)
}

func (i *intentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity) {
	i.lock.Lock()
	defer i.lock.Unlock()
	intentMap, ok := i.store[srcService]
	if !ok {
		intentMap = make(map[string]*goset.Set[model.OtterizeServiceIdentity])
		i.store[srcService] = intentMap
	}
	namespace := ""
	if srcService.Namespace != dstService.Namespace {
		// namespace is only needed if it's a connection between different namespaces.
		namespace = dstService.Namespace
	}
	if intentMap[srcService.Namespace] == nil {
		intentMap[srcService.Namespace] = goset.NewSet[model.OtterizeServiceIdentity]()
	}
	intentMap[srcService.Namespace].Add(model.OtterizeServiceIdentity{Name: dstService.Name, Namespace: namespace})
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
			serviceIntents.Add(serviceIdentities.Items()...)
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

func (i *intentsHolder) getConfigMapNamespace() (string, error) {
	namespace := viper.GetString(config.NamespaceKey)
	if namespace != "" {
		return namespace, nil
	}
	namespace, err := kubeutils.GetCurrentNamespace()
	if err != nil {
		return "", fmt.Errorf("could not deduce the store's configmap namespace: %w", err)
	}
	return namespace, nil
}

func (i *intentsHolder) WriteStore(ctx context.Context) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	jsonBytes, err := json.Marshal(i.store)
	if err != nil {
		return err
	}
	namespace, err := i.getConfigMapNamespace()
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
	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: viper.GetString(config.StoreConfigMapKey), Namespace: namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, i.client, configmap, func() error {
		configmap.Data = nil
		configmap.BinaryData = map[string][]byte{"store": compressedJson.Bytes()}
		return nil
	})
	return err
}

func (i *intentsHolder) LoadStore(ctx context.Context) error {
	i.lock.Lock()
	defer i.lock.Unlock()
	configmap := &corev1.ConfigMap{}
	namespace, err := i.getConfigMapNamespace()
	if err != nil {
		return err
	}
	err = i.client.Get(ctx, types.NamespacedName{Name: viper.GetString(config.StoreConfigMapKey), Namespace: namespace}, configmap)
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
