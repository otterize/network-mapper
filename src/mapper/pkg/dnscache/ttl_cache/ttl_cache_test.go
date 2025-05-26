package ttl_cache

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type TTLCacheTestSuite struct {
	suite.Suite
}

func (s *TTLCacheTestSuite) TestCacheFilterWhileWrite() {
	cache := NewTTLCache[string, string](100)
	stop := make(chan struct{})

	go func() {
		// Intensive write to the cache
		for {
			cache.Insert("example.com", fmt.Sprintf("192.0.2.%d", time.Now().UnixNano()%255), 5*time.Second)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		// Iterating over the cache
		for {
			_ = cache.Filter(func(key string) bool {
				return true
			})
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Let them race for 15 seconds
	// Unfortunately, there isn't a way to make sure that they won't race (which will yield fatal error: concurrent map iteration and map write)
	// so we hope that 15 seconds are good enough interval to reproduce the error if it exists
	time.Sleep(15 * time.Second)
	close(stop)
}

func (s *TTLCacheTestSuite) TestTTL() {
	cache := NewTTLCache[string, string](100)

	cache.Insert("my-future-blog.de", "ip1", 1*time.Second)
	ips := cache.Get("my-future-blog.de")
	s.Require().Len(ips, 1)
	s.Require().Equal("ip1", ips[0])

	// This is the only place where we sleep in the test, to make sure the TTL works as expected
	time.Sleep(2 * time.Second)

	cache.cleanupExpired()

	ips = cache.Get("my-future-blog.de")
	s.Require().Len(ips, 0)
}

func TestTTLCacheTestSuite(t *testing.T) {
	suite.Run(t, new(TTLCacheTestSuite))
}
