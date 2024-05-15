package dnsintentspublisher

import (
	"context"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/dnscache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	clientIntentsCRDName = "clientintents.k8s.otterize.com"
)

func InitWithManager(ctx context.Context, mgr manager.Manager, dnsCache *dnscache.DNSCache) (*Publisher, bool, error) {
	if !viper.GetBool(config.DNSClientIntentsUpdateEnabledKey) {
		return nil, false, nil
	}

	installed, err := IsClientIntentsInstalled(ctx, mgr)
	if err != nil {
		return nil, false, errors.Wrap(err)
	}

	if !installed {
		logrus.Debugf("DNS client intents publishing is not enabled due to missing CRD %s", clientIntentsCRDName)
		return nil, false, nil
	}

	dnsPublisher := NewPublisher(mgr.GetClient(), dnsCache)
	err = dnsPublisher.InitIndices(ctx, mgr)
	if err != nil {
		return nil, false, errors.Wrap(err)
	}

	return dnsPublisher, true, nil
}

func IsClientIntentsInstalled(ctx context.Context, mgr manager.Manager) (bool, error) {
	directClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		logrus.WithError(err).Error("unable to create kubernetes API client")
		return false, errors.Wrap(err)
	}

	crd := apiextensionsv1.CustomResourceDefinition{}
	err = directClient.Get(ctx, types.NamespacedName{Name: clientIntentsCRDName}, &crd)
	if err != nil && !k8serrors.IsNotFound(err) {
		return false, errors.Wrap(err)
	}

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	return true, nil
}
