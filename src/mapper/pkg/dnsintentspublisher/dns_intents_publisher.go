package dnsintentspublisher

import (
	"context"
	otterizev1alpha3 "github.com/otterize/intents-operator/src/operator/api/v1alpha3"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	hasAnyDnsIntentsIndexKey   = "hasAnyDnsIntents"
	hasAnyDnsIntentsIndexValue = "true"
)

type DnsCache interface {
	GetResolvedIP(dnsName string) (string, bool)
}

type Publisher struct {
	client         client.Client
	dnsCache       DnsCache
	updateInterval time.Duration
}

func NewPublisher(k8sClient client.Client, dnsCache DnsCache) *Publisher {
	return &Publisher{
		client:         k8sClient,
		dnsCache:       dnsCache,
		updateInterval: viper.GetDuration(config.DNSClientIntentsUpdateIntervalKey),
	}
}

func (p *Publisher) InitIndices(ctx context.Context, mgr ctrl.Manager) error {
	err := mgr.GetCache().IndexField(
		ctx,
		&otterizev1alpha3.ClientIntents{},
		hasAnyDnsIntentsIndexKey,
		func(object client.Object) []string {
			intents := object.(*otterizev1alpha3.ClientIntents)
			if intents.Spec == nil {
				return nil
			}

			if lo.ContainsBy(intents.GetCallsList(), func(intent otterizev1alpha3.Intent) bool {
				return intent.Type == otterizev1alpha3.IntentTypeInternet
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
	var intentsList otterizev1alpha3.ClientIntentsList

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

func (p *Publisher) updateResolvedIPs(ctx context.Context, clientIntents otterizev1alpha3.ClientIntents) error {
	resolvedIPsMap := lo.SliceToMap(clientIntents.Status.ResolvedIPs, func(resolvedIPs otterizev1alpha3.ResolvedIPs) (string, []string) {
		return resolvedIPs.DNS, resolvedIPs.IPs
	})

	shouldUpdate := false
	for _, intent := range clientIntents.GetCallsList() {
		newDnsFound := p.appendResolvedIps(intent, resolvedIPsMap)
		if newDnsFound {
			shouldUpdate = true
		}
	}

	if len(resolvedIPsMap) == 0 || !shouldUpdate {
		return nil
	}

	updatedResolvedIPs := make([]otterizev1alpha3.ResolvedIPs, 0, len(resolvedIPsMap))
	for dnsName, ips := range resolvedIPsMap {
		updatedResolvedIPs = append(updatedResolvedIPs, otterizev1alpha3.ResolvedIPs{
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

func (p *Publisher) appendResolvedIps(intent otterizev1alpha3.Intent, resolvedIPsMap map[string][]string) bool {
	if intent.Type != otterizev1alpha3.IntentTypeInternet {
		return false
	}

	dnsName := intent.Internet.Dns
	if len(dnsName) == 0 {
		return false
	}

	resolvedIP, ok := p.dnsCache.GetResolvedIP(dnsName)
	if !ok {
		// TODO: Add event on the ClientIntents to indicate that the DNS name is not resolved
		return false
	}

	ips, ok := resolvedIPsMap[dnsName]
	if !ok {
		ips = make([]string, 0)
	}

	if slices.Contains(ips, resolvedIP) {
		return false
	}

	ips = append(ips, resolvedIP)
	resolvedIPsMap[dnsName] = ips
	return true
}
