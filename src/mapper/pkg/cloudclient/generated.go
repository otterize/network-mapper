// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package cloudclient

import (
	"context"
	"time"

	"github.com/Khan/genqlient/graphql"
)

type ComponentType string

const (
	ComponentTypeIntentsOperator     ComponentType = "INTENTS_OPERATOR"
	ComponentTypeCredentialsOperator ComponentType = "CREDENTIALS_OPERATOR"
	ComponentTypeNetworkMapper       ComponentType = "NETWORK_MAPPER"
)

type DNSIPPairInput struct {
	DnsName string   `json:"dnsName"`
	Ips     []string `json:"ips"`
}

// GetDnsName returns DNSIPPairInput.DnsName, and is useful for accessing the field via an interface.
func (v *DNSIPPairInput) GetDnsName() string { return v.DnsName }

// GetIps returns DNSIPPairInput.Ips, and is useful for accessing the field via an interface.
func (v *DNSIPPairInput) GetIps() []string { return v.Ips }

type DatabaseConfigInput struct {
	Dbname     *string              `json:"dbname"`
	Table      *string              `json:"table"`
	Operations []*DatabaseOperation `json:"operations"`
}

// GetDbname returns DatabaseConfigInput.Dbname, and is useful for accessing the field via an interface.
func (v *DatabaseConfigInput) GetDbname() *string { return v.Dbname }

// GetTable returns DatabaseConfigInput.Table, and is useful for accessing the field via an interface.
func (v *DatabaseConfigInput) GetTable() *string { return v.Table }

// GetOperations returns DatabaseConfigInput.Operations, and is useful for accessing the field via an interface.
func (v *DatabaseConfigInput) GetOperations() []*DatabaseOperation { return v.Operations }

type DatabaseOperation string

const (
	DatabaseOperationAll    DatabaseOperation = "ALL"
	DatabaseOperationSelect DatabaseOperation = "SELECT"
	DatabaseOperationInsert DatabaseOperation = "INSERT"
	DatabaseOperationUpdate DatabaseOperation = "UPDATE"
	DatabaseOperationDelete DatabaseOperation = "DELETE"
)

type DiscoveredIntentInput struct {
	DiscoveredAt *time.Time   `json:"discoveredAt"`
	Intent       *IntentInput `json:"intent"`
}

// GetDiscoveredAt returns DiscoveredIntentInput.DiscoveredAt, and is useful for accessing the field via an interface.
func (v *DiscoveredIntentInput) GetDiscoveredAt() *time.Time { return v.DiscoveredAt }

// GetIntent returns DiscoveredIntentInput.Intent, and is useful for accessing the field via an interface.
func (v *DiscoveredIntentInput) GetIntent() *IntentInput { return v.Intent }

type ExternalTrafficDiscoveredIntentInput struct {
	DiscoveredAt time.Time                  `json:"discoveredAt"`
	Intent       ExternalTrafficIntentInput `json:"intent"`
}

// GetDiscoveredAt returns ExternalTrafficDiscoveredIntentInput.DiscoveredAt, and is useful for accessing the field via an interface.
func (v *ExternalTrafficDiscoveredIntentInput) GetDiscoveredAt() time.Time { return v.DiscoveredAt }

// GetIntent returns ExternalTrafficDiscoveredIntentInput.Intent, and is useful for accessing the field via an interface.
func (v *ExternalTrafficDiscoveredIntentInput) GetIntent() ExternalTrafficIntentInput {
	return v.Intent
}

type ExternalTrafficIntentInput struct {
	Namespace  string         `json:"namespace"`
	ClientName string         `json:"clientName"`
	Target     DNSIPPairInput `json:"target"`
}

// GetNamespace returns ExternalTrafficIntentInput.Namespace, and is useful for accessing the field via an interface.
func (v *ExternalTrafficIntentInput) GetNamespace() string { return v.Namespace }

// GetClientName returns ExternalTrafficIntentInput.ClientName, and is useful for accessing the field via an interface.
func (v *ExternalTrafficIntentInput) GetClientName() string { return v.ClientName }

// GetTarget returns ExternalTrafficIntentInput.Target, and is useful for accessing the field via an interface.
func (v *ExternalTrafficIntentInput) GetTarget() DNSIPPairInput { return v.Target }

type HTTPConfigInput struct {
	Path    *string       `json:"path"`
	Methods []*HTTPMethod `json:"methods"`
}

// GetPath returns HTTPConfigInput.Path, and is useful for accessing the field via an interface.
func (v *HTTPConfigInput) GetPath() *string { return v.Path }

// GetMethods returns HTTPConfigInput.Methods, and is useful for accessing the field via an interface.
func (v *HTTPConfigInput) GetMethods() []*HTTPMethod { return v.Methods }

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
	HTTPMethodAll     HTTPMethod = "ALL"
)

type IntentInput struct {
	Namespace         *string                `json:"namespace"`
	ClientName        *string                `json:"clientName"`
	ServerName        *string                `json:"serverName"`
	ServerNamespace   *string                `json:"serverNamespace"`
	Type              *IntentType            `json:"type"`
	Topics            []*KafkaConfigInput    `json:"topics"`
	Resources         []*HTTPConfigInput     `json:"resources"`
	DatabaseResources []*DatabaseConfigInput `json:"databaseResources"`
	AwsActions        []*string              `json:"awsActions"`
	Internet          *InternetConfigInput   `json:"internet"`
	Status            *IntentStatusInput     `json:"status"`
}

// GetNamespace returns IntentInput.Namespace, and is useful for accessing the field via an interface.
func (v *IntentInput) GetNamespace() *string { return v.Namespace }

// GetClientName returns IntentInput.ClientName, and is useful for accessing the field via an interface.
func (v *IntentInput) GetClientName() *string { return v.ClientName }

// GetServerName returns IntentInput.ServerName, and is useful for accessing the field via an interface.
func (v *IntentInput) GetServerName() *string { return v.ServerName }

// GetServerNamespace returns IntentInput.ServerNamespace, and is useful for accessing the field via an interface.
func (v *IntentInput) GetServerNamespace() *string { return v.ServerNamespace }

// GetType returns IntentInput.Type, and is useful for accessing the field via an interface.
func (v *IntentInput) GetType() *IntentType { return v.Type }

// GetTopics returns IntentInput.Topics, and is useful for accessing the field via an interface.
func (v *IntentInput) GetTopics() []*KafkaConfigInput { return v.Topics }

// GetResources returns IntentInput.Resources, and is useful for accessing the field via an interface.
func (v *IntentInput) GetResources() []*HTTPConfigInput { return v.Resources }

// GetDatabaseResources returns IntentInput.DatabaseResources, and is useful for accessing the field via an interface.
func (v *IntentInput) GetDatabaseResources() []*DatabaseConfigInput { return v.DatabaseResources }

// GetAwsActions returns IntentInput.AwsActions, and is useful for accessing the field via an interface.
func (v *IntentInput) GetAwsActions() []*string { return v.AwsActions }

// GetInternet returns IntentInput.Internet, and is useful for accessing the field via an interface.
func (v *IntentInput) GetInternet() *InternetConfigInput { return v.Internet }

// GetStatus returns IntentInput.Status, and is useful for accessing the field via an interface.
func (v *IntentInput) GetStatus() *IntentStatusInput { return v.Status }

type IntentStatusInput struct {
	IstioStatus *IstioStatusInput `json:"istioStatus"`
}

// GetIstioStatus returns IntentStatusInput.IstioStatus, and is useful for accessing the field via an interface.
func (v *IntentStatusInput) GetIstioStatus() *IstioStatusInput { return v.IstioStatus }

type IntentType string

const (
	IntentTypeHttp     IntentType = "HTTP"
	IntentTypeKafka    IntentType = "KAFKA"
	IntentTypeDatabase IntentType = "DATABASE"
	IntentTypeAws      IntentType = "AWS"
	IntentTypeS3       IntentType = "S3"
	IntentTypeInternet IntentType = "INTERNET"
)

type InternetConfigInput struct {
	Ips   []*string `json:"ips"`
	Ports []*int    `json:"ports"`
}

// GetIps returns InternetConfigInput.Ips, and is useful for accessing the field via an interface.
func (v *InternetConfigInput) GetIps() []*string { return v.Ips }

// GetPorts returns InternetConfigInput.Ports, and is useful for accessing the field via an interface.
func (v *InternetConfigInput) GetPorts() []*int { return v.Ports }

type IstioStatusInput struct {
	ServiceAccountName     *string `json:"serviceAccountName"`
	IsServiceAccountShared *bool   `json:"isServiceAccountShared"`
	IsServerMissingSidecar *bool   `json:"isServerMissingSidecar"`
	IsClientMissingSidecar *bool   `json:"isClientMissingSidecar"`
}

// GetServiceAccountName returns IstioStatusInput.ServiceAccountName, and is useful for accessing the field via an interface.
func (v *IstioStatusInput) GetServiceAccountName() *string { return v.ServiceAccountName }

// GetIsServiceAccountShared returns IstioStatusInput.IsServiceAccountShared, and is useful for accessing the field via an interface.
func (v *IstioStatusInput) GetIsServiceAccountShared() *bool { return v.IsServiceAccountShared }

// GetIsServerMissingSidecar returns IstioStatusInput.IsServerMissingSidecar, and is useful for accessing the field via an interface.
func (v *IstioStatusInput) GetIsServerMissingSidecar() *bool { return v.IsServerMissingSidecar }

// GetIsClientMissingSidecar returns IstioStatusInput.IsClientMissingSidecar, and is useful for accessing the field via an interface.
func (v *IstioStatusInput) GetIsClientMissingSidecar() *bool { return v.IsClientMissingSidecar }

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
	KafkaOperationAll             KafkaOperation = "ALL"
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
	// Report integration components status
	ReportIntegrationComponentStatus bool `json:"reportIntegrationComponentStatus"`
}

// GetReportIntegrationComponentStatus returns ReportComponentStatusResponse.ReportIntegrationComponentStatus, and is useful for accessing the field via an interface.
func (v *ReportComponentStatusResponse) GetReportIntegrationComponentStatus() bool {
	return v.ReportIntegrationComponentStatus
}

// ReportDiscoveredIntentsResponse is returned by ReportDiscoveredIntents on success.
type ReportDiscoveredIntentsResponse struct {
	ReportDiscoveredIntents *bool `json:"reportDiscoveredIntents"`
}

// GetReportDiscoveredIntents returns ReportDiscoveredIntentsResponse.ReportDiscoveredIntents, and is useful for accessing the field via an interface.
func (v *ReportDiscoveredIntentsResponse) GetReportDiscoveredIntents() *bool {
	return v.ReportDiscoveredIntents
}

// ReportExternalTrafficDiscoveredIntentsResponse is returned by ReportExternalTrafficDiscoveredIntents on success.
type ReportExternalTrafficDiscoveredIntentsResponse struct {
	ReportExternalTrafficDiscoveredIntents bool `json:"reportExternalTrafficDiscoveredIntents"`
}

// GetReportExternalTrafficDiscoveredIntents returns ReportExternalTrafficDiscoveredIntentsResponse.ReportExternalTrafficDiscoveredIntents, and is useful for accessing the field via an interface.
func (v *ReportExternalTrafficDiscoveredIntentsResponse) GetReportExternalTrafficDiscoveredIntents() bool {
	return v.ReportExternalTrafficDiscoveredIntents
}

// __ReportComponentStatusInput is used internally by genqlient
type __ReportComponentStatusInput struct {
	Component ComponentType `json:"component"`
}

// GetComponent returns __ReportComponentStatusInput.Component, and is useful for accessing the field via an interface.
func (v *__ReportComponentStatusInput) GetComponent() ComponentType { return v.Component }

// __ReportDiscoveredIntentsInput is used internally by genqlient
type __ReportDiscoveredIntentsInput struct {
	Intents []*DiscoveredIntentInput `json:"intents"`
}

// GetIntents returns __ReportDiscoveredIntentsInput.Intents, and is useful for accessing the field via an interface.
func (v *__ReportDiscoveredIntentsInput) GetIntents() []*DiscoveredIntentInput { return v.Intents }

// __ReportExternalTrafficDiscoveredIntentsInput is used internally by genqlient
type __ReportExternalTrafficDiscoveredIntentsInput struct {
	Intents []ExternalTrafficDiscoveredIntentInput `json:"intents"`
}

// GetIntents returns __ReportExternalTrafficDiscoveredIntentsInput.Intents, and is useful for accessing the field via an interface.
func (v *__ReportExternalTrafficDiscoveredIntentsInput) GetIntents() []ExternalTrafficDiscoveredIntentInput {
	return v.Intents
}

// The query or mutation executed by ReportComponentStatus.
const ReportComponentStatus_Operation = `
mutation ReportComponentStatus ($component: ComponentType!) {
	reportIntegrationComponentStatus(component: $component)
}
`

func ReportComponentStatus(
	ctx context.Context,
	client graphql.Client,
	component ComponentType,
) (*ReportComponentStatusResponse, error) {
	req := &graphql.Request{
		OpName: "ReportComponentStatus",
		Query:  ReportComponentStatus_Operation,
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

// The query or mutation executed by ReportDiscoveredIntents.
const ReportDiscoveredIntents_Operation = `
mutation ReportDiscoveredIntents ($intents: [DiscoveredIntentInput!]!) {
	reportDiscoveredIntents(intents: $intents)
}
`

func ReportDiscoveredIntents(
	ctx context.Context,
	client graphql.Client,
	intents []*DiscoveredIntentInput,
) (*ReportDiscoveredIntentsResponse, error) {
	req := &graphql.Request{
		OpName: "ReportDiscoveredIntents",
		Query:  ReportDiscoveredIntents_Operation,
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

// The query or mutation executed by ReportExternalTrafficDiscoveredIntents.
const ReportExternalTrafficDiscoveredIntents_Operation = `
mutation ReportExternalTrafficDiscoveredIntents ($intents: [ExternalTrafficDiscoveredIntentInput!]!) {
	reportExternalTrafficDiscoveredIntents(intents: $intents)
}
`

func ReportExternalTrafficDiscoveredIntents(
	ctx context.Context,
	client graphql.Client,
	intents []ExternalTrafficDiscoveredIntentInput,
) (*ReportExternalTrafficDiscoveredIntentsResponse, error) {
	req := &graphql.Request{
		OpName: "ReportExternalTrafficDiscoveredIntents",
		Query:  ReportExternalTrafficDiscoveredIntents_Operation,
		Variables: &__ReportExternalTrafficDiscoveredIntentsInput{
			Intents: intents,
		},
	}
	var err error

	var data ReportExternalTrafficDiscoveredIntentsResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
