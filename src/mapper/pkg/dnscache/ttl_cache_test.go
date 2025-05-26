package dnscache

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type TTLCacheTestSuite struct {
	suite.Suite
}

func (s *DNSCacheTestSuite) TestCacheFilterWhileWrite() {
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

func TestTTLCacheTestSuite(t *testing.T) {
	suite.Run(t, new(TTLCacheTestSuite))
}
