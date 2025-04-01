package metrics_collection_traffic

import (
	"fmt"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"hash/crc32"
)

type CacheValue []byte

type MetricsCollectionTrafficCache struct {
	cache *expirable.LRU[string, CacheValue]
}

func NewMetricsCollectionTrafficCache() *MetricsCollectionTrafficCache {
	size := viper.GetInt(config.MetricsCollectionTrafficCacheSizeKey)
	// We don't want the cache to expire. It does not contain a lot of data, and we want to keep it as long as possible
	// so we won't send unnecessary requests to the cloud.
	cache := expirable.NewLRU[string, CacheValue](size, OnEvict, 0)

	return &MetricsCollectionTrafficCache{
		cache: cache,
	}
}

func (c *MetricsCollectionTrafficCache) Get(namespace string, reason cloudclient.EligibleForMetricsCollectionReason) (CacheValue, bool) {
	return c.cache.Get(c.key(namespace, reason))
}

func (c *MetricsCollectionTrafficCache) Set(namespace string, reason cloudclient.EligibleForMetricsCollectionReason, value CacheValue) bool {
	return c.cache.Add(c.key(namespace, reason), value)
}

func (c *MetricsCollectionTrafficCache) key(namespace string, reason cloudclient.EligibleForMetricsCollectionReason) string {
	return fmt.Sprintf("%s#%s", namespace, reason)
}

func (c *MetricsCollectionTrafficCache) GenerateValue(pods []cloudclient.K8sResourceEligibleForMetricsCollectionInput) (CacheValue, error) {
	values := lo.Map(pods, func(resource cloudclient.K8sResourceEligibleForMetricsCollectionInput, _ int) string {
		return fmt.Sprintf("%s#%s", resource.Name, resource.Kind)
	})

	slices.Sort(values)

	hash := crc32.NewIEEE()
	for _, value := range values {
		_, err := hash.Write([]byte(value))
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	hashSum := hash.Sum(nil)

	return hashSum, nil
}

func OnEvict(key string, _ CacheValue) {
	logrus.WithField("namespace", key).Debug("key evicted from cache, you may change configuration to increase cache size")
}
