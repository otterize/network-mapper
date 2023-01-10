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
	store             intentsHolderStore
	lock              sync.Mutex
	client            client.Client
	config            IntentsHolderConfig
	lastIntentsUpdate time.Time
}

func NewIntentsHolder(client client.Client, config IntentsHolderConfig) *IntentsHolder {
	return &IntentsHolder{
		store:  NewIntentsHolderStore(),
		lock:   sync.Mutex{},
		client: client,
		config: config,
	}
}

func (i *IntentsHolder) Reset() {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.store.Reset()
}

func (i *IntentsHolder) AddIntent(srcService model.OtterizeServiceIdentity, dstService model.OtterizeServiceIdentity, discoveryTime time.Time) {
	i.lock.Lock()
	defer i.lock.Unlock()

	storeUpdated := i.store.Update(srcService, dstService, discoveryTime)
	if storeUpdated {
		i.lastIntentsUpdate = time.Now()
	}
}

func (i *IntentsHolder) LastIntentsUpdate() time.Time {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.lastIntentsUpdate
}

func (i *IntentsHolder) GetIntentsPerNamespace(namespaces []string) map[model.OtterizeServiceIdentity]map[model.OtterizeServiceIdentity]time.Time {
	i.lock.Lock()
	defer i.lock.Unlock()
	if len(namespaces) == 0 {
		return i.store.GetAllIntents()
	}

	uniqueNamespaces := goset.FromSlice(namespaces).Items()
	return i.store.GetIntentsByNamespace(uniqueNamespaces)
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
