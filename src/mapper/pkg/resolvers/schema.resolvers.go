package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"github.com/otterize/otternose/mapper/pkg/config"
	"github.com/otterize/otternose/mapper/pkg/graph/generated"
	"github.com/otterize/otternose/mapper/pkg/graph/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func (r *mutationResolver) ReportCaptureResults(ctx context.Context, results model.CaptureResults) (*bool, error) {
	for _, captureItem := range results.Results {
		srcPod, ok := r.podsReconciler.ResolveIpToPod(captureItem.SrcIP)
		if !ok {
			logrus.Warningf("Ip %s didn't match any pod", captureItem.SrcIP)
			continue
		}
		srcIdentity, err := r.podsReconciler.ResolvePodToServiceIdentity(ctx, srcPod)
		if err != nil {
			return nil, err
		}
		for _, dest := range captureItem.Destinations {
			if !strings.HasSuffix(dest, viper.GetString(config.ClusterDomainKey)) {
				// not a k8s service, ignore
				continue
			}
			ips, ok := r.endpointsReconciler.ResolveServiceAddressToIps(dest)
			if !ok {
				logrus.Warningf("Could not resolve service address %s", dest)
				continue
			}
			destPod, ok := r.podsReconciler.ResolveIpToPod(ips[0])
			if !ok {
				logrus.Warningf("Could not resolve pod IP %s", ips[0])
				continue
			}
			dstIdentity, err := r.podsReconciler.ResolvePodToServiceIdentity(ctx, destPod)
			if err != nil {
				logrus.Warningf("Could not resolve pod %s to identity", ips[0])
				continue
			}
			r.intentsHolder.AddIntent(srcIdentity, dstIdentity)
		}
	}
	return nil, nil
}

func (r *queryResolver) GetIntents(ctx context.Context) ([]model.ServiceIntents, error) {
	result := make([]model.ServiceIntents, 0)
	for service, intents := range r.intentsHolder.GetIntentsPerService() {
		result = append(result, model.ServiceIntents{Name: service, Intents: intents})
	}
	return result, nil
}

func (r *queryResolver) GetCrds(ctx context.Context) (string, error) {
	t, err := template.New("crd").Parse(crdTemplate)
	if err != nil {
		return "", err
	}
	stringBuffer := bytes.NewBufferString("")
	err = t.Execute(stringBuffer, r.intentsHolder.GetIntentsPerService())
	if err != nil {
		return "", err
	}
	return stringBuffer.String(), nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
