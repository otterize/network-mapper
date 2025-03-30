package labelreporter

import (
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"hash/crc32"
	"sort"
	"time"
)

type serviceIdentityKey string
type labelsChecksum uint32

type serviceIdLabelsCache struct {
	cache *expirable.LRU[serviceIdentityKey, labelsChecksum]
}

func newServiceIdLabelsCache(size int, ttl time.Duration) *serviceIdLabelsCache {
	cache := expirable.NewLRU[serviceIdentityKey, labelsChecksum](size, nil, ttl)
	return &serviceIdLabelsCache{
		cache: cache,
	}
}

func (c *serviceIdLabelsCache) Add(key serviceIdentityKey, value labelsChecksum) {
	c.cache.Add(key, value)
}

func (c *serviceIdLabelsCache) IsCached(key serviceIdentityKey, value labelsChecksum) bool {
	cachedValue, ok := c.cache.Get(key)
	if !ok {
		return false
	}
	return cachedValue == value
}

func checksumLabels(labels map[string]string) labelsChecksum {
	labelKeys := make([]string, 0, len(labels))
	for key := range labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)
	labelString := ""
	for _, key := range labelKeys {
		labelString += key + labels[key]
	}

	labelHash := crc32.ChecksumIEEE([]byte(labelString))
	return labelsChecksum(labelHash)
}

func serviceIdentityToServiceIdentityInput(identity serviceidentity.ServiceIdentity) cloudclient.ServiceIdentityInput {
	wi := cloudclient.ServiceIdentityInput{
		Namespace: identity.Namespace,
		Name:      identity.Name,
		Kind:      identity.Kind,
	}
	if identity.ResolvedUsingOverrideAnnotation != nil {
		wi.NameResolvedUsingAnnotation = nilable.From(*identity.ResolvedUsingOverrideAnnotation)
	}

	return wi
}
