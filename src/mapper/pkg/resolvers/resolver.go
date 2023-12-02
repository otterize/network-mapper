package resolvers

import (
	"context"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/labstack/echo/v4"
	"github.com/otterize/intents-operator/src/shared/serviceidresolver"
	"github.com/otterize/network-mapper/src/mapper/pkg/config"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/generated"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/otterize/network-mapper/src/mapper/pkg/intentsstore"
	"github.com/otterize/network-mapper/src/mapper/pkg/kubefinder"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	kubeFinder                   *kubefinder.KubeFinder
	serviceIdResolver            *serviceidresolver.Resolver
	intentsHolder                *intentsstore.IntentsHolder
	externalTrafficIntentsHolder *ExternalTrafficIntentsHolder
}

type IP string

type ExternalTrafficIntent struct {
	Client   model.OtterizeServiceIdentity `json:"client"`
	LastSeen time.Time
	DNSName  string
	IPs      map[IP]struct{}
}

type TimestampedExternalTrafficIntent struct {
	Timestamp time.Time
	Intent    ExternalTrafficIntent
}

type ExternalTrafficKey struct {
	ClientName      string
	ClientNamespace string
	DestDNSName     string
}

type ExternalTrafficIntentsHolder struct {
	intents   map[ExternalTrafficKey]TimestampedExternalTrafficIntent
	lock      sync.Mutex
	callbacks []ExternalTrafficCallbackFunc
}

type ExternalTrafficCallbackFunc func(context.Context, []TimestampedExternalTrafficIntent)

func NewExternalTrafficIntentsHolder() *ExternalTrafficIntentsHolder {
	return &ExternalTrafficIntentsHolder{
		intents: make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent),
	}
}

func (h *ExternalTrafficIntentsHolder) RegisterNotifyIntents(callback ExternalTrafficCallbackFunc) {
	h.callbacks = append(h.callbacks, callback)
}

func (h *ExternalTrafficIntentsHolder) PeriodicIntentsUpload(ctx context.Context, interval time.Duration) {
	logrus.Info("Starting periodic external traffic intents upload")

	for {
		select {
		case <-time.After(interval):
			if len(h.callbacks) == 0 {
				continue
			}

			intents := h.GetNewIntentsSinceLastGet()
			if len(intents) == 0 {
				continue
			}
			for _, callback := range h.callbacks {
				callback(ctx, intents)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (h *ExternalTrafficIntentsHolder) GetNewIntentsSinceLastGet() []TimestampedExternalTrafficIntent {
	h.lock.Lock()
	defer h.lock.Unlock()

	intents := make([]TimestampedExternalTrafficIntent, 0, len(h.intents))

	for _, intent := range h.intents {
		intents = append(intents, intent)
	}

	h.intents = make(map[ExternalTrafficKey]TimestampedExternalTrafficIntent)

	return intents
}

func (h *ExternalTrafficIntentsHolder) AddIntent(intent ExternalTrafficIntent) {
	if config.ExcludedNamespaces().Contains(intent.Client.Namespace) {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	key := ExternalTrafficKey{
		ClientName:      intent.Client.Name,
		ClientNamespace: intent.Client.Namespace,
		DestDNSName:     intent.DNSName,
	}
	_, ok := h.intents[key]
	if !ok {
		h.intents[key] = TimestampedExternalTrafficIntent{
			Timestamp: intent.LastSeen,
			Intent:    intent,
		}
		return
	}

	mergedIntent := h.intents[key]

	for ip := range intent.IPs {
		mergedIntent.Intent.IPs[ip] = struct{}{}
	}
	if intent.LastSeen.After(mergedIntent.Timestamp) {
		mergedIntent.Timestamp = intent.LastSeen
	}

	h.intents[key] = mergedIntent
}

func NewResolver(kubeFinder *kubefinder.KubeFinder, serviceIdResolver *serviceidresolver.Resolver, intentsHolder *intentsstore.IntentsHolder, externalTrafficHolder *ExternalTrafficIntentsHolder) *Resolver {
	return &Resolver{
		kubeFinder:                   kubeFinder,
		serviceIdResolver:            serviceIdResolver,
		intentsHolder:                intentsHolder,
		externalTrafficIntentsHolder: externalTrafficHolder,
	}
}

func (r *Resolver) Register(e *echo.Echo) {
	c := generated.Config{Resolvers: r}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(c))
	e.Any("/query", func(c echo.Context) error {
		srv.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
