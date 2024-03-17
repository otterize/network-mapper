package dnsintentspublisher

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func InitWithManager(ctx context.Context, mgr manager.Manager, dnsCache *dnscache.DNSCache) (*Publisher, bool, error) {
	if !viper.GetBool(config.DNSClientIntentsUpdateEnabledKey) {
		return nil, false, nil
	}

	dnsPublisher := NewPublisher(mgr.GetClient(), dnsCache)
	err := dnsPublisher.InitIndices(ctx, mgr)
	if err != nil {
		discoveryErr := (&apiutil.ErrResourceDiscoveryFailed{})
		if errors.As(err, &discoveryErr) {
			for gvk := range *discoveryErr {
				if gvk.Group == "k8s.otterize.com" {
					// This can happen if the network mapper is deployed without the intents operator, which is normal.
					logrus.Debugf("DNS client intents publishing is not enabled due to missing CRD %v", gvk)
					return nil, false, nil
				}
			}
		}
		return nil, false, errors.Wrap(err)
	}

	return dnsPublisher, true, nil
}
