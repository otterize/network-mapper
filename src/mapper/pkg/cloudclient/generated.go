// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package cloudclient

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type ComponentType string

const (
	ComponentTypeIntentsOperator     ComponentType = "INTENTS_OPERATOR"
	ComponentTypeCredentialsOperator ComponentType = "CREDENTIALS_OPERATOR"
	ComponentTypeNetworkMapper       ComponentType = "NETWORK_MAPPER"
)

type HTTPConfigInput struct {
	Path   *string     `json:"path"`
	Method *HTTPMethod `json:"method"`
}

// GetPath returns HTTPConfigInput.Path, and is useful for accessing the field via an interface.
func (v *HTTPConfigInput) GetPath() *string { return v.Path }

// GetMethod returns HTTPConfigInput.Method, and is useful for accessing the field via an interface.
func (v *HTTPConfigInput) GetMethod() *HTTPMethod { return v.Method }

type HTTPMethod string

const (
	HTTPMethodGet     HTTPMethod = "GET"
	HTTPMethodPost    HTTPMethod = "POST"
	HTTPMethodPut     HTTPMethod = "PUT"
	HTTPMethodDelete  HTTPMethod = "DELETE"
	HTTPMethodOptions HTTPMethod = "OPTIONS"
	HTTPMethodTrace   HTTPMethod = "TRACE"
	HTTPMethodPatch   HTTPMethod = "PATCH"
	HTTPMethodConnect HTTPMethod = "CONNECT"
)

type IntentBody struct {
	Type      *IntentType         `json:"type"`
	Topics    []*KafkaConfigInput `json:"topics"`
	Resources []*HTTPConfigInput  `json:"resources"`
}

// GetType returns IntentBody.Type, and is useful for accessing the field via an interface.
func (v *IntentBody) GetType() *IntentType { return v.Type }

// GetTopics returns IntentBody.Topics, and is useful for accessing the field via an interface.
func (v *IntentBody) GetTopics() []*KafkaConfigInput { return v.Topics }

// GetResources returns IntentBody.Resources, and is useful for accessing the field via an interface.
func (v *IntentBody) GetResources() []*HTTPConfigInput { return v.Resources }

type IntentInput struct {
	Namespace       *string     `json:"namespace"`
	ClientName      *string     `json:"clientName"`
	ServerName      *string     `json:"serverName"`
	ServerNamespace *string     `json:"serverNamespace"`
	Body            *IntentBody `json:"body"`
}

// GetNamespace returns IntentInput.Namespace, and is useful for accessing the field via an interface.
func (v *IntentInput) GetNamespace() *string { return v.Namespace }

// GetClientName returns IntentInput.ClientName, and is useful for accessing the field via an interface.
func (v *IntentInput) GetClientName() *string { return v.ClientName }

// GetServerName returns IntentInput.ServerName, and is useful for accessing the field via an interface.
func (v *IntentInput) GetServerName() *string { return v.ServerName }

// GetServerNamespace returns IntentInput.ServerNamespace, and is useful for accessing the field via an interface.
func (v *IntentInput) GetServerNamespace() *string { return v.ServerNamespace }

// GetBody returns IntentInput.Body, and is useful for accessing the field via an interface.
func (v *IntentInput) GetBody() *IntentBody { return v.Body }

type IntentType string

const (
	IntentTypeHttp  IntentType = "HTTP"
	IntentTypeKafka IntentType = "KAFKA"
	IntentTypeGrpc  IntentType = "GRPC"
	IntentTypeRedis IntentType = "REDIS"
)

type KafkaConfigInput struct {
	Name       *string           `json:"name"`
	Operations []*KafkaOperation `json:"operations"`
}

// GetName returns KafkaConfigInput.Name, and is useful for accessing the field via an interface.
func (v *KafkaConfigInput) GetName() *string { return v.Name }

// GetOperations returns KafkaConfigInput.Operations, and is useful for accessing the field via an interface.
func (v *KafkaConfigInput) GetOperations() []*KafkaOperation { return v.Operations }

type KafkaOperation string

const (
	KafkaOperationConsume         KafkaOperation = "CONSUME"
	KafkaOperationProduce         KafkaOperation = "PRODUCE"
	KafkaOperationCreate          KafkaOperation = "CREATE"
	KafkaOperationAlter           KafkaOperation = "ALTER"
	KafkaOperationDelete          KafkaOperation = "DELETE"
	KafkaOperationDescribe        KafkaOperation = "DESCRIBE"
	KafkaOperationClusterAction   KafkaOperation = "CLUSTER_ACTION"
	KafkaOperationDescribeConfigs KafkaOperation = "DESCRIBE_CONFIGS"
	KafkaOperationAlterConfigs    KafkaOperation = "ALTER_CONFIGS"
	KafkaOperationIdempotentWrite KafkaOperation = "IDEMPOTENT_WRITE"
)

// ReportComponentStatusResponse is returned by ReportComponentStatus on success.
type ReportComponentStatusResponse struct {
	ReportComponentStatus string `json:"reportComponentStatus"`
}

// GetReportComponentStatus returns ReportComponentStatusResponse.ReportComponentStatus, and is useful for accessing the field via an interface.
func (v *ReportComponentStatusResponse) GetReportComponentStatus() string {
	return v.ReportComponentStatus
}

// ReportDiscoveredIntentsResponse is returned by ReportDiscoveredIntents on success.
type ReportDiscoveredIntentsResponse struct {
	ReportDiscoveredIntents *bool `json:"reportDiscoveredIntents"`
}

// GetReportDiscoveredIntents returns ReportDiscoveredIntentsResponse.ReportDiscoveredIntents, and is useful for accessing the field via an interface.
func (v *ReportDiscoveredIntentsResponse) GetReportDiscoveredIntents() *bool {
	return v.ReportDiscoveredIntents
}

// __ReportComponentStatusInput is used internally by genqlient
type __ReportComponentStatusInput struct {
	Component ComponentType `json:"component"`
}

// GetComponent returns __ReportComponentStatusInput.Component, and is useful for accessing the field via an interface.
func (v *__ReportComponentStatusInput) GetComponent() ComponentType { return v.Component }

// __ReportDiscoveredIntentsInput is used internally by genqlient
type __ReportDiscoveredIntentsInput struct {
	Intents []*IntentInput `json:"intents"`
}

// GetIntents returns __ReportDiscoveredIntentsInput.Intents, and is useful for accessing the field via an interface.
func (v *__ReportDiscoveredIntentsInput) GetIntents() []*IntentInput { return v.Intents }

func ReportComponentStatus(
	ctx context.Context,
	client graphql.Client,
	component ComponentType,
) (*ReportComponentStatusResponse, error) {
	req := &graphql.Request{
		OpName: "ReportComponentStatus",
		Query: `
mutation ReportComponentStatus ($component: ComponentType!) {
	reportComponentStatus(component: $component)
}
`,
		Variables: &__ReportComponentStatusInput{
			Component: component,
		},
	}
	var err error

	var data ReportComponentStatusResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

func ReportDiscoveredIntents(
	ctx context.Context,
	client graphql.Client,
	intents []*IntentInput,
) (*ReportDiscoveredIntentsResponse, error) {
	req := &graphql.Request{
		OpName: "ReportDiscoveredIntents",
		Query: `
mutation ReportDiscoveredIntents ($intents: [IntentInput!]!) {
	reportDiscoveredIntents(intents: $intents)
}
`,
		Variables: &__ReportDiscoveredIntentsInput{
			Intents: intents,
		},
	}
	var err error

	var data ReportDiscoveredIntentsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
