package dnscache

import (
	"context"
	"github.com/jellydator/ttlcache/v3"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

type DNSCache struct {
	cache         *ttlcache.Cache[string, string]
	ipToNameCache *ttlcache.Cache[string, string]
}

func NewDNSCache() *DNSCache {
	capacity := viper.GetInt(config.DNSCacheItemsMaxCapacityKey)
	dnsRecordCache := ttlcache.New[string, string](ttlcache.WithCapacity[string, string](uint64(capacity)))
	go dnsRecordCache.Start()
	ipToNameCache := ttlcache.New[string, string](ttlcache.WithCapacity[string, string](uint64(capacity)))
	go ipToNameCache.Start()

	lastCapacityReachedErrorPrint := time.Time{}
	ipToNameLastCapacityReachedErrorPrint := time.Time{}
	dnsRecordCache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, string]) {
		if reason == ttlcache.EvictionReasonCapacityReached && time.Since(lastCapacityReachedErrorPrint) > time.Minute {
			logrus.Warningf("DNS cache capacity reached entries are being dropped, consider increasing config '%s'",
				config.DNSCacheItemsMaxCapacityKey)
			lastCapacityReachedErrorPrint = time.Now()
		}
	})

	ipToNameCache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, string]) {
		if reason == ttlcache.EvictionReasonCapacityReached && time.Since(ipToNameLastCapacityReachedErrorPrint) > time.Minute {
			logrus.Warningf("DNS cache capacity reached entries are being dropped, consider increasing config '%s'",
				config.DNSCacheItemsMaxCapacityKey)
			ipToNameLastCapacityReachedErrorPrint = time.Now()
		}
	})

	return &DNSCache{
		cache:         dnsRecordCache,
		ipToNameCache: ipToNameCache,
	}
}

func (d *DNSCache) AddOrUpdateDNSData(dnsName string, ip string, ttl time.Duration) {
	d.cache.Set(dnsName, ip, ttl)
	d.ipToNameCache.Set(ip, dnsName, ttl)
}

func (d *DNSCache) GetResolvedIP(dnsName string) (string, bool) {
	entry := d.cache.Get(dnsName)
	if entry == nil {
		return "", false
	}
	return entry.Value(), true
}

func (d *DNSCache) GetResolvedDNSName(ip string) (string, bool) {
	entry := d.ipToNameCache.Get(ip)
	if entry == nil {
		return "", false
	}
	return entry.Value(), true
}
