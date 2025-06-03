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
	"time"
)

type CacheValue []byte

type WebhookServicesCache struct {
	cache *expirable.LRU[string, CacheValue]
}

func NewWebhookServicesCache() *WebhookServicesCache {
	size := viper.GetInt(config.WebhookServicesCacheSizeKey)
	cache := expirable.NewLRU[string, CacheValue](size, OnEvict, 5*time.Hour)

	return &WebhookServicesCache{
		cache: cache,
	}
}

func (c *WebhookServicesCache) Get() (CacheValue, bool) {
	return c.cache.Get("webhooks")
}

func (c *WebhookServicesCache) Set(value CacheValue) bool {
	return c.cache.Add("webhooks", value)
}

func K8sWebhookServiceInputKey(webhookService cloudclient.K8sWebhookServiceInput) string {
	return fmt.Sprintf("%s#%s#%s#%s",
		webhookService.Identity.Namespace,
		webhookService.Identity.Name,
		webhookService.WebhookName,
		webhookService.WebhookType)
}

func (c *WebhookServicesCache) GenerateValue(webhookServices []cloudclient.K8sWebhookServiceInput) (CacheValue, error) {
	values := lo.Map(webhookServices, func(item cloudclient.K8sWebhookServiceInput, _ int) string {
		return K8sWebhookServiceInputKey(item)
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
