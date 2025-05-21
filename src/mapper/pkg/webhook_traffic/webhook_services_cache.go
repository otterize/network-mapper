package webhook_traffic

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

type WebhookServicesCache struct {
	cache *expirable.LRU[string, CacheValue]
}

func NewWebhookServicesCache() *WebhookServicesCache {
	size := viper.GetInt(config.WebhookServicesCacheSizeKey)
	// We don't want the cache to expire. It does not contain a lot of data, and we want to keep it as long as possible
	// so we won't send unnecessary requests to the cloud.
	cache := expirable.NewLRU[string, CacheValue](size, OnEvict, 0)

	return &WebhookServicesCache{
		cache: cache,
	}
}

func (c *WebhookServicesCache) Get(namespace string, serviceName string, webhookName string, webhookType cloudclient.WebhookType) (CacheValue, bool) {
	return c.cache.Get(c.key(namespace, serviceName, webhookName, webhookType))
}

func (c *WebhookServicesCache) Set(namespace string, serviceName string, webhookName string, webhookType cloudclient.WebhookType, value CacheValue) bool {
	return c.cache.Add(c.key(namespace, serviceName, webhookName, webhookType), value)
}

func (c *WebhookServicesCache) key(namespace string, serviceName string, webhookName string, webhookType cloudclient.WebhookType) string {
	return fmt.Sprintf("%s#%s#%s#%s", namespace, serviceName, webhookName, webhookType)
}

func (c *WebhookServicesCache) GenerateValue(webhookServices []cloudclient.K8sWebhookServiceInput) (CacheValue, error) {
	values := lo.Map(webhookServices, func(item cloudclient.K8sWebhookServiceInput, _ int) string {
		return c.key(item.Namespace, item.ServiceName, item.WebhookName, item.WebhookType)
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
