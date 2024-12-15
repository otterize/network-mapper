package dnscache

import (
	"fmt"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

const (
	IP1 = "10.0.0.1"
	IP2 = "10.0.0.2"
)

type DNSCacheTestSuite struct {
	suite.Suite
}

func (s *DNSCacheTestSuite) TearDownTest() {
	viper.Set(config.DNSCacheItemsMaxCapacityKey, config.DNSCacheItemsMaxCapacityDefault)
}

func (s *DNSCacheTestSuite) TestDNSCache() {
	cache := NewDNSCache()
	cache.AddOrUpdateDNSData("good-news.com", IP1, 60*time.Second)
	ips := cache.GetResolvedIPs("good-news.com")
	s.Require().Len(ips, 1)
	s.Require().Equal(IP1, ips[0])

	cache.AddOrUpdateDNSData("good-news.com", IP2, 60*time.Second)
	ips = cache.GetResolvedIPs("good-news.com")
	s.Require().Len(ips, 2)
	s.Require().Contains(ips, IP1)
	s.Require().Contains(ips, IP2)

	ips = cache.GetResolvedIPs("bad-news.de")
	s.Require().Len(ips, 0)

	cache.AddOrUpdateDNSData("bad-news.de", IP1, 60*time.Second)
	ips = cache.GetResolvedIPs("bad-news.de")
	s.Require().Len(ips, 1)
	s.Require().Equal(IP1, ips[0])
}

func (s *DNSCacheTestSuite) TestCapacityConfig() {
	capacityLimit := 2
	viper.Set(config.DNSCacheItemsMaxCapacityKey, capacityLimit)
	cache := NewDNSCache()
	names := make([]string, 0)
	for i := 0; i < capacityLimit+1; i++ {
		dnsName := fmt.Sprintf("dns-%d.com", i)
		cache.AddOrUpdateDNSData(dnsName, IP1, 60*time.Second)
		names = append(names, dnsName)
	}

	for i, dnsName := range names {
		vals := cache.GetResolvedIPs(dnsName)
		if i == 0 {
			s.Require().Len(vals, 0)
		} else {
			s.Require().Len(vals, 1)
		}
	}
}

func (s *DNSCacheTestSuite) TestTTL() {
	cache := NewDNSCache()

	cache.AddOrUpdateDNSData("my-future-blog.de", IP1, 1*time.Second)
	ips := cache.GetResolvedIPs("my-future-blog.de")
	s.Require().Len(ips, 1)
	s.Require().Equal(IP1, ips[0])

	// This is the only place where we sleep in the test, to make sure the TTL works as expected
	time.Sleep(2 * time.Second)

	cache.cache.cleanupExpired()

	ips = cache.GetResolvedIPs("my-future-blog.de")
	s.Require().Len(ips, 0)

}

func (s *DNSCacheTestSuite) TestWildcardIP() {
	cache := NewDNSCache()
	cache.AddOrUpdateDNSData("www.surf-forecast.com", IP1, 60*time.Second)
	ips := cache.GetResolvedIPsForWildcard("*.surf-forecast.com")
	s.Require().Len(ips, 1)
	s.Require().Equal(ips[0], IP1)
}

func (s *DNSCacheTestSuite) TestMultipleWildcardIPs() {
	cache := NewDNSCache()
	cache.AddOrUpdateDNSData("www.surf-forecast.com", IP1, 60*time.Second)
	cache.AddOrUpdateDNSData("api.surf-forecast.com", IP2, 60*time.Second)
	ips := cache.GetResolvedIPsForWildcard("*.surf-forecast.com")
	s.Require().Len(ips, 2)
	s.Require().Equal(ips, []string{IP1, IP2})
}

func TestDNSCacheTestSuite(t *testing.T) {
	suite.Run(t, new(DNSCacheTestSuite))
}
