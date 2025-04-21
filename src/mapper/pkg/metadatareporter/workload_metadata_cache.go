package metadatareporter

import (
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"hash/crc32"
	"sort"
	"strings"
	"time"
)

type serviceIdentityKey string
type metadataChecksum uint32

type workloadMetadataCache struct {
	cache *expirable.LRU[serviceIdentityKey, metadataChecksum]
}

func newWorkloadMetadataCache(size int, ttl time.Duration) *workloadMetadataCache {
	cache := expirable.NewLRU[serviceIdentityKey, metadataChecksum](size, nil, ttl)
	return &workloadMetadataCache{
		cache: cache,
	}
}

func (c *workloadMetadataCache) Add(key serviceIdentityKey, value metadataChecksum) {
	c.cache.Add(key, value)
}

func (c *workloadMetadataCache) IsCached(key serviceIdentityKey, value metadataChecksum) bool {
	cachedValue, ok := c.cache.Get(key)
	if !ok {
		return false
	}
	return cachedValue == value
}

func checksumMetadata(labels map[string]string, podIps []string, serviceIps []string) metadataChecksum {
	labelKeys := make([]string, 0, len(labels))
	for key := range labels {
		labelKeys = append(labelKeys, key)
	}
	sort.Strings(labelKeys)
	labelString := ""
	for _, key := range labelKeys {
		labelString += key + labels[key]
	}
	sort.Strings(podIps)
	sort.Strings(serviceIps)

	ipsString := strings.Join(append(podIps, serviceIps...), ",")

	hash := crc32.ChecksumIEEE([]byte(labelString + ipsString))
	return metadataChecksum(hash)
}

func serviceIdentityToCacheKey(identity serviceidentity.ServiceIdentity) serviceIdentityKey {
	return serviceIdentityKey(identity.GetNameWithKind())
}
