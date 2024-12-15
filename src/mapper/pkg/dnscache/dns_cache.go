package dnscache

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"strings"
	"time"
)

type DNSCache struct {
	cache *TTLCache[string, string]
}

type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func NewDNSCache() *DNSCache {
	capacity := viper.GetInt(config.DNSCacheItemsMaxCapacityKey)
	if capacity == 0 {
		logrus.Panic("Capacity cannot be 0")
	}
	dnsRecordCache := NewTTLCache[string, string](capacity)

	return &DNSCache{
		cache: dnsRecordCache,
	}
}

func (d *DNSCache) AddOrUpdateDNSData(dnsName string, ip string, ttl time.Duration) {
	d.cache.Insert(dnsName, ip, ttl)
}

func (d *DNSCache) GetResolvedIPs(dnsName string) []string {
	entry := d.cache.Get(dnsName)
	return entry
}

func (d *DNSCache) GetResolvedIPsForWildcard(dnsName string) []string {
	dnsSuffix := strings.TrimPrefix(dnsName, "*") // Strip the wildcard, leave the '.example.com' suffix
	result := make([]string, 0)
	for entry := range d.cache.items {
		if strings.HasSuffix(entry, dnsSuffix) {
			// Calling cache.Get() to utilize the LRU instead of iterating over the value too
			result = append(result, d.cache.Get(entry)...)
		}
	}
	return result
}

// CacheValue holds the value and its expiration time
type CacheValue[V any] struct {
	Value      V
	Expiration time.Time
}
