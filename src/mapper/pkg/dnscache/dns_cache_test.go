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
	viper.Reset()
}

func (s *DNSCacheTestSuite) TestDNSCache() {
	cache := NewDNSCache()
	cache.AddOrUpdateDNSData("good-news.com", IP1, 60*time.Second)
	ip, found := cache.GetResolvedIP("good-news.com")
	s.Require().True(found)
	s.Require().Equal(IP1, ip)

	cache.AddOrUpdateDNSData("good-news.com", IP2, 60*time.Second)
	ip, found = cache.GetResolvedIP("good-news.com")
	s.Require().True(found)
	s.Require().Equal(IP2, ip)

	_, found = cache.GetResolvedIP("bad-news.de")
	s.Require().False(found)

	cache.AddOrUpdateDNSData("bad-news.de", IP1, 60*time.Second)
	ip, found = cache.GetResolvedIP("bad-news.de")
	s.Require().True(found)
	s.Require().Equal(IP1, ip)
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
		_, found := cache.GetResolvedIP(dnsName)
		if i == 0 {
			s.Require().False(found)
		} else {
			s.Require().True(found)
		}
	}
}

func (s *DNSCacheTestSuite) TestTTL() {
	cache := NewDNSCache()

	cache.AddOrUpdateDNSData("my-future-blog.de", IP1, 1*time.Second)
	ip, found := cache.GetResolvedIP("my-future-blog.de")
	s.Require().True(found)
	s.Require().Equal(IP1, ip)

	// This is the only place where we sleep in the test, to make sure the TTL works as expected
	time.Sleep(1100 * time.Millisecond)
	_, found = cache.GetResolvedIP("my-future-blog.de")
	s.Require().False(found)

}

func TestDNSCacheTestSuite(t *testing.T) {
	suite.Run(t, new(DNSCacheTestSuite))
}
