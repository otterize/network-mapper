package dnscache

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache/ttl_cache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"strings"
	"time"
)

type DNSCache struct {
	cache *ttl_cache.TTLCache[string, string]
}

type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func NewDNSCache() *DNSCache {
	capacity := viper.GetInt(config.DNSCacheItemsMaxCapacityKey)
	if capacity == 0 {
		logrus.Panic("Capacity cannot be 0")
	}
	dnsRecordCache := ttl_cache.NewTTLCache[string, string](capacity)

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
	result := d.cache.Filter(func(key string) bool {
		return strings.HasSuffix(key, dnsSuffix)
	})

	return result
}
