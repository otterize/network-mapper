package dnsintentspublisher

import (
	"context"
	otterizev2alpha1 "github.com/otterize/intents-operator/src/operator/api/v2alpha1"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	hasAnyDnsIntentsIndexKey   = "hasAnyDnsIntents"
	hasAnyDnsIntentsIndexValue = "true"
)

type Publisher struct {
	client         client.Client
	dnsCache       *dnscache.DNSCache
	updateInterval time.Duration
}

func NewPublisher(k8sClient client.Client, dnsCache *dnscache.DNSCache) *Publisher {
	return &Publisher{
		client:         k8sClient,
		dnsCache:       dnsCache,
		updateInterval: viper.GetDuration(config.DNSClientIntentsUpdateIntervalKey),
	}
}

func (p *Publisher) InitIndices(ctx context.Context, mgr ctrl.Manager) error {
	err := mgr.GetCache().IndexField(
		ctx,
		&otterizev2alpha1.ClientIntents{},
		hasAnyDnsIntentsIndexKey,
		func(object client.Object) []string {
			intents := object.(*otterizev2alpha1.ClientIntents)
			if intents.Spec == nil {
				return nil
			}

			if lo.ContainsBy(intents.GetTargetList(), func(intent otterizev2alpha1.Target) bool {
				return intent.Internet != nil
			}) {
				return []string{hasAnyDnsIntentsIndexValue}
			}

			return nil
		})
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (p *Publisher) RunForever(ctx context.Context) {
	ticker := time.NewTicker(p.updateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := p.PublishDNSIntents(ctx)
			if err != nil {
				logrus.WithError(err).Error("Failed to publish DNS intents")
			}
		}
	}
}

func (p *Publisher) PublishDNSIntents(ctx context.Context) error {
	var intentsList otterizev2alpha1.ClientIntentsList

	err := p.client.List(ctx, &intentsList, client.MatchingFields{hasAnyDnsIntentsIndexKey: hasAnyDnsIntentsIndexValue})
	if err != nil {
		return errors.Wrap(err)
	}

	for _, clientIntents := range intentsList.Items {
		err := p.updateResolvedIPs(ctx, clientIntents)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func (p *Publisher) updateResolvedIPs(ctx context.Context, clientIntents otterizev2alpha1.ClientIntents) error {
	resolvedIPsMap, shouldUpdate := p.compareIntentsAndStatus(clientIntents)
	if !shouldUpdate {
		return nil
	}

	updatedResolvedIPs := make([]otterizev2alpha1.ResolvedIPs, 0, len(resolvedIPsMap))
	for dnsName, ips := range resolvedIPsMap {
		updatedResolvedIPs = append(updatedResolvedIPs, otterizev2alpha1.ResolvedIPs{
			DNS: dnsName,
			IPs: ips,
		})
	}

	updateClientIntents := clientIntents.DeepCopy()
	updateClientIntents.Status.ResolvedIPs = updatedResolvedIPs
	err := p.client.Status().Patch(ctx, updateClientIntents, client.MergeFrom(&clientIntents))
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (p *Publisher) compareIntentsAndStatus(clientIntents otterizev2alpha1.ClientIntents) (map[string][]string, bool) {
	resolvedIPsMap := lo.SliceToMap(clientIntents.Status.ResolvedIPs, func(resolvedIPs otterizev2alpha1.ResolvedIPs) (string, []string) {
		return resolvedIPs.DNS, resolvedIPs.IPs
	})

	dnsIntents := lo.Reduce(clientIntents.GetTargetList(), func(names []string, intent otterizev2alpha1.Target, _ int) []string {
		if intent.Internet == nil {
			return names
		}
		names = append(names, intent.Internet.Domains...)
		return names
	}, make([]string, 0))

	shouldUpdate := false
	for _, dns := range dnsIntents {
		newDnsFound := p.appendResolvedIps(dns, resolvedIPsMap)
		if newDnsFound {
			shouldUpdate = true
		}
	}

	for resolvedDNS := range resolvedIPsMap {
		notPresentOnAnyIntent := !slices.Contains(dnsIntents, resolvedDNS)
		if notPresentOnAnyIntent {
			delete(resolvedIPsMap, resolvedDNS)
			shouldUpdate = true
		}
	}
	return resolvedIPsMap, shouldUpdate
}

func (p *Publisher) appendResolvedIps(dnsName string, resolvedIPsMap map[string][]string) bool {
	resolvedIP, ipResolved := p.dnsCache.GetResolvedIP(dnsName)

	ips, ok := resolvedIPsMap[dnsName]
	if !ok {
		ips = make([]string, 0)
		if !ipResolved {
			// Try to resolve it ourselves
			ctxTimeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			logrus.WithField("dnsName", dnsName).Warn("DNS cache miss, resolving it ourselves")
			ipaddrs, err := net.DefaultResolver.LookupIPAddr(ctxTimeout, dnsName)
			if err != nil {
				logrus.WithError(err).WithField("dnsName", dnsName).Error("Failed to resolve DNS")
				return false
			}

			for _, ip := range ipaddrs {
				ips = append(ips, ip.String())
				p.dnsCache.AddOrUpdateDNSData(dnsName, ip.String(), 60)
			}
			resolvedIPsMap[dnsName] = ips
			return true
		}
	}

	// This happens when we've resolved the IP ourselves in a prior run, and still have no new passive resolution this time.
	if !ipResolved {
		return false
	}

	if slices.Contains(ips, resolvedIP) {
		return false
	}

	ips = append(ips, resolvedIP)
	resolvedIPsMap[dnsName] = ips
	return true
}
