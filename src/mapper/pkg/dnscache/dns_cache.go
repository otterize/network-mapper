package dnscache

import (
	"container/list"
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"sync"
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

// CacheValue holds the value and its expiration time
type CacheValue[V any] struct {
	Value      V
	Expiration time.Time
}

// CacheEntry represents an entry in the cache, linking the key with its list element for LRU
type CacheEntry[K comparable, V comparable] struct {
	Key   K
	Value CacheValue[V]
}

// TTLCache is a generic TTL cache that stores unique items with individual TTLs and LRU eviction
type TTLCache[K comparable, V comparable] struct {
	items     map[K]map[V]*list.Element // Key to map of values, each value points to an LRU element
	lru       *list.List                // List for LRU eviction, stores CacheEntry[K, V]
	maxSize   int                       // Maximum size of the cache
	mu        sync.Mutex
	cleanupCh chan struct{}
}

// NewTTLCache creates a new TTL cache with the specified maxSize
func NewTTLCache[K comparable, V comparable](maxSize int) *TTLCache[K, V] {
	cache := &TTLCache[K, V]{
		items:     make(map[K]map[V]*list.Element),
		lru:       list.New(),
		maxSize:   maxSize,
		cleanupCh: make(chan struct{}),
	}

	// Start the cleanup process
	go cache.startCleanup()

	return cache
}

// Insert adds a unique value to the cache under the specified key with its own TTL
// and manages the LRU eviction when the cache exceeds the max size.
func (c *TTLCache[K, V]) Insert(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If the key doesn't exist, create an entry for it
	if _, exists := c.items[key]; !exists {
		c.items[key] = make(map[V]*list.Element)
	}

	// Check if the value already exists under this key and remove it from LRU if so
	if elem, exists := c.items[key][value]; exists {
		c.lru.Remove(elem)
	}

	// Insert or update the value with its expiration time and add it to the LRU list
	cacheEntry := CacheEntry[K, V]{Key: key, Value: CacheValue[V]{Value: value, Expiration: time.Now().Add(ttl)}}
	lruElem := c.lru.PushFront(cacheEntry)
	c.items[key][value] = lruElem

	// Manage the cache size, evict the least recently used item if needed
	if c.lru.Len() > c.maxSize {
		c.evict()
	}

}

// evict removes the least recently used item from the cache
func (c *TTLCache[K, V]) evict() {
	// Remove the least recently used item (which is at the back of the LRU list)
	lruElem := c.lru.Back()
	if lruElem == nil {
		return
	}

	cacheEntry := lruElem.Value.(CacheEntry[K, V])
	key, value := cacheEntry.Key, cacheEntry.Value

	// Remove the value from the cache
	if _, exists := c.items[key]; exists {
		delete(c.items[key], value.Value)

		// If no more values exist under this key, remove the key itself
		if len(c.items[key]) == 0 {
			delete(c.items, key)
		}
	}

	// Remove from the LRU list
	c.lru.Remove(lruElem)
}

// Get retrieves the values for a specific key and removes any expired values
// Returns a slice of valid values for the given key
func (c *TTLCache[K, V]) Get(key K) []V {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the key exists
	if _, exists := c.items[key]; !exists {
		return make([]V, 0)
	}

	// Filter out expired values and prepare the result
	var result []V
	for value, lruElem := range c.items[key] {
		cacheEntry := lruElem.Value.(CacheEntry[K, V])

		// If the value has expired, remove it
		if time.Now().After(c.lruValueExpiration(lruElem)) {
			c.lru.Remove(lruElem)
			delete(c.items[key], value)
			continue
		}

		// Add valid values to the result
		result = append(result, cacheEntry.Value.Value)

		// Move the accessed item to the front of the LRU list (mark as recently used)
		c.lru.MoveToFront(lruElem)
	}

	// If all values are expired, remove the key entirely
	if len(c.items[key]) == 0 {
		delete(c.items, key)
	}

	return result
}

// cleanupExpired removes expired values from the cache
func (c *TTLCache[K, V]) cleanupExpired() {
	for key, values := range c.items {
		for value, lruElem := range values {
			// If a value has expired, remove it
			if time.Now().After(c.lruValueExpiration(lruElem)) {
				c.lru.Remove(lruElem)
				delete(values, value)
			}
		}

		// If all values are expired, remove the key entirely
		if len(values) == 0 {
			delete(c.items, key)
		}
	}
}

// lruValueExpiration gets the expiration time for a given LRU element
func (c *TTLCache[K, V]) lruValueExpiration(elem *list.Element) time.Time {
	cacheEntry := elem.Value.(CacheEntry[K, V])
	return cacheEntry.Value.Expiration
}

// startCleanup periodically cleans up expired items
func (c *TTLCache[K, V]) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute) // Cleanup interval
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			c.cleanupExpired()
			c.mu.Unlock()
		case <-c.cleanupCh:
			return
		}
	}
}

// Stop stops the cache cleanup process
func (c *TTLCache[K, V]) Stop() {
	close(c.cleanupCh)
}
