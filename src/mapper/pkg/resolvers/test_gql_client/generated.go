// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package test_gql_client

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type CaptureResultForSrcIp struct {
	SrcIp        string   `json:"srcIp"`
	Destinations []string `json:"destinations"`
}

// GetSrcIp returns CaptureResultForSrcIp.SrcIp, and is useful for accessing the field via an interface.
func (v *CaptureResultForSrcIp) GetSrcIp() string { return v.SrcIp }

// GetDestinations returns CaptureResultForSrcIp.Destinations, and is useful for accessing the field via an interface.
func (v *CaptureResultForSrcIp) GetDestinations() []string { return v.Destinations }

type CaptureResults struct {
	Results []CaptureResultForSrcIp `json:"results"`
}

// GetResults returns CaptureResults.Results, and is useful for accessing the field via an interface.
func (v *CaptureResults) GetResults() []CaptureResultForSrcIp { return v.Results }

// ReportCaptureResultsResponse is returned by ReportCaptureResults on success.
type ReportCaptureResultsResponse struct {
	ReportCaptureResults bool `json:"reportCaptureResults"`
}

// GetReportCaptureResults returns ReportCaptureResultsResponse.ReportCaptureResults, and is useful for accessing the field via an interface.
func (v *ReportCaptureResultsResponse) GetReportCaptureResults() bool { return v.ReportCaptureResults }

// ReportSocketScanResultsResponse is returned by ReportSocketScanResults on success.
type ReportSocketScanResultsResponse struct {
	ReportSocketScanResults bool `json:"reportSocketScanResults"`
}

// GetReportSocketScanResults returns ReportSocketScanResultsResponse.ReportSocketScanResults, and is useful for accessing the field via an interface.
func (v *ReportSocketScanResultsResponse) GetReportSocketScanResults() bool {
	return v.ReportSocketScanResults
}

// ServiceIntentsResponse is returned by ServiceIntents on success.
type ServiceIntentsResponse struct {
	ServiceIntents []ServiceIntentsServiceIntents `json:"serviceIntents"`
}

// GetServiceIntents returns ServiceIntentsResponse.ServiceIntents, and is useful for accessing the field via an interface.
func (v *ServiceIntentsResponse) GetServiceIntents() []ServiceIntentsServiceIntents {
	return v.ServiceIntents
}

// ServiceIntentsServiceIntents includes the requested fields of the GraphQL type ServiceIntents.
type ServiceIntentsServiceIntents struct {
	Client  ServiceIntentsServiceIntentsClientOtterizeServiceIdentity    `json:"client"`
	Intents []ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity `json:"intents"`
}

// GetClient returns ServiceIntentsServiceIntents.Client, and is useful for accessing the field via an interface.
func (v *ServiceIntentsServiceIntents) GetClient() ServiceIntentsServiceIntentsClientOtterizeServiceIdentity {
	return v.Client
}

// GetIntents returns ServiceIntentsServiceIntents.Intents, and is useful for accessing the field via an interface.
func (v *ServiceIntentsServiceIntents) GetIntents() []ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity {
	return v.Intents
}

// ServiceIntentsServiceIntentsClientOtterizeServiceIdentity includes the requested fields of the GraphQL type OtterizeServiceIdentity.
type ServiceIntentsServiceIntentsClientOtterizeServiceIdentity struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// GetName returns ServiceIntentsServiceIntentsClientOtterizeServiceIdentity.Name, and is useful for accessing the field via an interface.
func (v *ServiceIntentsServiceIntentsClientOtterizeServiceIdentity) GetName() string { return v.Name }

// GetNamespace returns ServiceIntentsServiceIntentsClientOtterizeServiceIdentity.Namespace, and is useful for accessing the field via an interface.
func (v *ServiceIntentsServiceIntentsClientOtterizeServiceIdentity) GetNamespace() string {
	return v.Namespace
}

// ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity includes the requested fields of the GraphQL type OtterizeServiceIdentity.
type ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// GetName returns ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity.Name, and is useful for accessing the field via an interface.
func (v *ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity) GetName() string { return v.Name }

// GetNamespace returns ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity.Namespace, and is useful for accessing the field via an interface.
func (v *ServiceIntentsServiceIntentsIntentsOtterizeServiceIdentity) GetNamespace() string {
	return v.Namespace
}

type SocketScanResultForSrcIp struct {
	SrcIp   string   `json:"srcIp"`
	DestIps []string `json:"destIps"`
}

// GetSrcIp returns SocketScanResultForSrcIp.SrcIp, and is useful for accessing the field via an interface.
func (v *SocketScanResultForSrcIp) GetSrcIp() string { return v.SrcIp }

// GetDestIps returns SocketScanResultForSrcIp.DestIps, and is useful for accessing the field via an interface.
func (v *SocketScanResultForSrcIp) GetDestIps() []string { return v.DestIps }

type SocketScanResults struct {
	Results []SocketScanResultForSrcIp `json:"results"`
}

// GetResults returns SocketScanResults.Results, and is useful for accessing the field via an interface.
func (v *SocketScanResults) GetResults() []SocketScanResultForSrcIp { return v.Results }

// __ReportCaptureResultsInput is used internally by genqlient
type __ReportCaptureResultsInput struct {
	Results CaptureResults `json:"results"`
}

// GetResults returns __ReportCaptureResultsInput.Results, and is useful for accessing the field via an interface.
func (v *__ReportCaptureResultsInput) GetResults() CaptureResults { return v.Results }

// __ReportSocketScanResultsInput is used internally by genqlient
type __ReportSocketScanResultsInput struct {
	Results SocketScanResults `json:"results"`
}

// GetResults returns __ReportSocketScanResultsInput.Results, and is useful for accessing the field via an interface.
func (v *__ReportSocketScanResultsInput) GetResults() SocketScanResults { return v.Results }

// __ServiceIntentsInput is used internally by genqlient
type __ServiceIntentsInput struct {
	Namespaces []string `json:"namespaces"`
}

// GetNamespaces returns __ServiceIntentsInput.Namespaces, and is useful for accessing the field via an interface.
func (v *__ServiceIntentsInput) GetNamespaces() []string { return v.Namespaces }

func ReportCaptureResults(
	ctx context.Context,
	client graphql.Client,
	results CaptureResults,
) (*ReportCaptureResultsResponse, error) {
	req := &graphql.Request{
		OpName: "ReportCaptureResults",
		Query: `
mutation ReportCaptureResults ($results: CaptureResults!) {
	reportCaptureResults(results: $results)
}
`,
		Variables: &__ReportCaptureResultsInput{
			Results: results,
		},
	}
	var err error

	var data ReportCaptureResultsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

func ReportSocketScanResults(
	ctx context.Context,
	client graphql.Client,
	results SocketScanResults,
) (*ReportSocketScanResultsResponse, error) {
	req := &graphql.Request{
		OpName: "ReportSocketScanResults",
		Query: `
mutation ReportSocketScanResults ($results: SocketScanResults!) {
	reportSocketScanResults(results: $results)
}
`,
		Variables: &__ReportSocketScanResultsInput{
			Results: results,
		},
	}
	var err error

	var data ReportSocketScanResultsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

func ServiceIntents(
	ctx context.Context,
	client graphql.Client,
	namespaces []string,
) (*ServiceIntentsResponse, error) {
	req := &graphql.Request{
		OpName: "ServiceIntents",
		Query: `
query ServiceIntents ($namespaces: [String!]) {
	serviceIntents(namespaces: $namespaces) {
		client {
			name
			namespace
		}
		intents {
			name
			namespace
		}
	}
}
`,
		Variables: &__ServiceIntentsInput{
			Namespaces: namespaces,
		},
	}
	var err error

	var data ServiceIntentsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
