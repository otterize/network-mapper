package resourcevisibility

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/otterize/intents-operator/src/shared/errors"
	"github.com/otterize/intents-operator/src/shared/injectablerecorder"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"hash/crc32"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"slices"
	"time"
)

type ServiceReconciler struct {
	client.Client
	injectablerecorder.InjectableRecorder
	otterizeCloud                    cloudclient.CloudClient
	kubeFinder                       KubeFinder
	namespaceToReportedServicesCache *expirable.LRU[string, []byte]
}

type KubeFinder interface {
	ResolveOtterizeIdentityForService(ctx context.Context, service *corev1.Service, now time.Time) (model.OtterizeServiceIdentity, bool, error)
}

func OnEvict(key string, _ []byte) {
	logrus.WithField("namespace", key).Debug("key evicted from cache, you may change configuration to increase cache size or TTL")
}

func NewServiceReconciler(client client.Client, otterizeCloudClient cloudclient.CloudClient, kubeFinder KubeFinder) *ServiceReconciler {
	size := viper.GetInt(config.ServiceCacheSizeKey)
	ttl := viper.GetDuration(config.ServiceCacheTTLDurationKey)
	cache := expirable.NewLRU[string, []byte](size, OnEvict, ttl)
	return &ServiceReconciler{
		Client:                           client,
		otterizeCloud:                    otterizeCloudClient,
		kubeFinder:                       kubeFinder,
		namespaceToReportedServicesCache: cache,
	}
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	recorder := mgr.GetEventRecorderFor("intents-operator")
	r.InjectRecorder(recorder)

	// We subscribe to the Endpoints resource, to make sure that any changes in pods that is selected by a service
	// will trigger a reconciliation of the namespace.
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		WithOptions(controller.Options{RecoverPanic: lo.ToPtr(true)}).
		Complete(r)
}

func (r *ServiceReconciler) InjectRecorder(recorder record.EventRecorder) {
	r.Recorder = recorder
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	namespace := req.Namespace
	var ServiceList corev1.ServiceList
	err := r.List(ctx, &ServiceList, client.InNamespace(namespace))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	servicesToReport, err := r.convertToCloudServices(ctx, ServiceList.Items)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	hashSum, err := r.getCachedValue(servicesToReport)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	val, found := r.namespaceToReportedServicesCache.Get(namespace)
	if found && bytes.Equal(val, hashSum) {
		logrus.WithField("namespace", namespace).Debug("Skipping reporting of services in namespace due to cache")
		return ctrl.Result{}, nil
	}

	err = r.otterizeCloud.ReportK8sServices(ctx, namespace, servicesToReport)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err)
	}

	r.namespaceToReportedServicesCache.Add(namespace, hashSum)

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) getCachedValue(servicesToReport []cloudclient.K8sServiceInput) ([]byte, error) {
	values := lo.Map(servicesToReport, func(service cloudclient.K8sServiceInput, _ int) string {
		return reportedServicesCacheValuePart(service)
	})

	slices.Sort(values)

	hash := crc32.NewIEEE()
	for _, value := range values {
		_, err := hash.Write([]byte(value))
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	hashSum := hash.Sum(nil)
	return hashSum, nil
}

func reportedServicesCacheValuePart(service cloudclient.K8sServiceInput) string {
	namedPortsStr := ""
	for _, namedPort := range service.Service.TargetNamedPorts {
		namedPortsStr += fmt.Sprintf("%s-%d-%s", namedPort.Name, namedPort.Port, namedPort.Protocol)
	}
	return fmt.Sprintf("%s-%s-%s-%s", service.ResourceName, service.OtterizeServer, service.Service.Spec.Type.Item, namedPortsStr)
}

func (r *ServiceReconciler) convertToCloudServices(ctx context.Context, services []corev1.Service) ([]cloudclient.K8sServiceInput, error) {
	cloudServices := make([]cloudclient.K8sServiceInput, 0)
	for _, service := range services {
		cloudService, ok, err := r.convertToCloudService(ctx, service)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if !ok {
			continue
		}

		cloudServices = append(cloudServices, cloudService)
	}

	return cloudServices, nil
}

func (r *ServiceReconciler) convertToCloudService(ctx context.Context, service corev1.Service) (cloudclient.K8sServiceInput, bool, error) {
	if !service.DeletionTimestamp.IsZero() {
		return cloudclient.K8sServiceInput{}, false, nil
	}

	identity, found, err := r.kubeFinder.ResolveOtterizeIdentityForService(ctx, &service, time.Now())
	if err != nil {
		return cloudclient.K8sServiceInput{}, false, errors.Wrap(err)
	}
	if !found {
		return cloudclient.K8sServiceInput{}, false, nil
	}

	serviceInput, err := convertServiceResource(ctx, r.Client, service)
	if err != nil {
		return cloudclient.K8sServiceInput{}, false, errors.Wrap(err)
	}

	cloudService := cloudclient.K8sServiceInput{
		OtterizeServer: identity.Name,
		ResourceName:   service.Name,
		Namespace:      identity.Namespace,
		Service:        serviceInput,
	}

	return cloudService, true, nil
}
