package service

import (
	"github.com/sirupsen/logrus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	crtClient "sigs.k8s.io/controller-runtime/pkg/client"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func CreateControllerRuntimeComponentsOrDie() (manager.Manager, crtClient.Client) {
	options := manager.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
	}

	mgr, err := manager.New(clientconfig.GetConfigOrDie(), options)

	if err != nil {
		logrus.Panicf("unable to set up overall controller manager: %s", err)
	}

	client, err := crtClient.New(mgr.GetConfig(), crtClient.Options{Scheme: scheme})

	if err != nil {
		logrus.Panicf("unable to set up client: %s", err)
	}

	return mgr, client
}
