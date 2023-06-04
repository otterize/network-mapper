// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type CaptureResultForSrcIP struct {
	Src          *OtterizeServiceIdentityInput `json:"src"`
	Destinations []Destination                 `json:"destinations"`
}

type CaptureResults struct {
	Results []CaptureResultForSrcIP `json:"results"`
}

type Destination struct {
	Destination *OtterizeServiceIdentityInput `json:"destination"`
	LastSeen    time.Time                     `json:"lastSeen"`
}

type GroupVersionKind struct {
	Group   *string `json:"group"`
	Version string  `json:"version"`
	Kind    string  `json:"kind"`
}

type HTTPResource struct {
	Path    string       `json:"path"`
	Methods []HTTPMethod `json:"methods"`
}

type Intent struct {
	Client        *OtterizeServiceIdentity `json:"client"`
	Server        *OtterizeServiceIdentity `json:"server"`
	Type          *IntentType              `json:"type"`
	KafkaTopics   []KafkaConfig            `json:"kafkaTopics"`
	HTTPResources []HTTPResource           `json:"httpResources"`
}

type IstioConnection struct {
	SrcWorkload          string       `json:"srcWorkload"`
	SrcWorkloadNamespace string       `json:"srcWorkloadNamespace"`
	DstWorkload          string       `json:"dstWorkload"`
	DstWorkloadNamespace string       `json:"dstWorkloadNamespace"`
	Path                 string       `json:"path"`
	Methods              []HTTPMethod `json:"methods"`
	LastSeen             time.Time    `json:"lastSeen"`
}

type IstioConnectionResults struct {
	Results []IstioConnection `json:"results"`
}

type KafkaConfig struct {
	Name       string           `json:"name"`
	Operations []KafkaOperation `json:"operations"`
}

type KafkaMapperResult struct {
	SrcIP           string    `json:"srcIp"`
	ServerPodName   string    `json:"serverPodName"`
	ServerNamespace string    `json:"serverNamespace"`
	Topic           string    `json:"topic"`
	Operation       string    `json:"operation"`
	LastSeen        time.Time `json:"lastSeen"`
}

type KafkaMapperResults struct {
	Results []KafkaMapperResult `json:"results"`
}

type OtterizeServiceIdentity struct {
	Name      string     `json:"name"`
	Namespace string     `json:"namespace"`
	Labels    []PodLabel `json:"labels"`
	// If the service identity was resolved from a pod owner, the GroupVersionKind of the pod owner.
	PodOwnerKind *GroupVersionKind `json:"podOwnerKind"`
}

type OtterizeServiceIdentityInput struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type PodLabel struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ServiceIntents struct {
	Client  *OtterizeServiceIdentity  `json:"client"`
	Intents []OtterizeServiceIdentity `json:"intents"`
}

type SocketScanResultForSrcIP struct {
	Src          *OtterizeServiceIdentityInput `json:"src"`
	Destinations []Destination                 `json:"destinations"`
}

type SocketScanResults struct {
	Results []SocketScanResultForSrcIP `json:"results"`
}

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

var AllHTTPMethod = []HTTPMethod{
	HTTPMethodGet,
	HTTPMethodPost,
	HTTPMethodPut,
	HTTPMethodDelete,
	HTTPMethodOptions,
	HTTPMethodTrace,
	HTTPMethodPatch,
	HTTPMethodConnect,
	HTTPMethodAll,
}

func (e HTTPMethod) IsValid() bool {
	switch e {
	case HTTPMethodGet, HTTPMethodPost, HTTPMethodPut, HTTPMethodDelete, HTTPMethodOptions, HTTPMethodTrace, HTTPMethodPatch, HTTPMethodConnect, HTTPMethodAll:
		return true
	}
	return false
}

func (e HTTPMethod) String() string {
	return string(e)
}

func (e *HTTPMethod) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = HTTPMethod(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid HttpMethod", str)
	}
	return nil
}

func (e HTTPMethod) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type IntentType string

const (
	IntentTypeKafka IntentType = "KAFKA"
	IntentTypeHTTP  IntentType = "HTTP"
)

var AllIntentType = []IntentType{
	IntentTypeKafka,
	IntentTypeHTTP,
}

func (e IntentType) IsValid() bool {
	switch e {
	case IntentTypeKafka, IntentTypeHTTP:
		return true
	}
	return false
}

func (e IntentType) String() string {
	return string(e)
}

func (e *IntentType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = IntentType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid IntentType", str)
	}
	return nil
}

func (e IntentType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

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
	KafkaOperationIDEmpotentWrite KafkaOperation = "IDEMPOTENT_WRITE"
)

var AllKafkaOperation = []KafkaOperation{
	KafkaOperationAll,
	KafkaOperationConsume,
	KafkaOperationProduce,
	KafkaOperationCreate,
	KafkaOperationAlter,
	KafkaOperationDelete,
	KafkaOperationDescribe,
	KafkaOperationClusterAction,
	KafkaOperationDescribeConfigs,
	KafkaOperationAlterConfigs,
	KafkaOperationIDEmpotentWrite,
}

func (e KafkaOperation) IsValid() bool {
	switch e {
	case KafkaOperationAll, KafkaOperationConsume, KafkaOperationProduce, KafkaOperationCreate, KafkaOperationAlter, KafkaOperationDelete, KafkaOperationDescribe, KafkaOperationClusterAction, KafkaOperationDescribeConfigs, KafkaOperationAlterConfigs, KafkaOperationIDEmpotentWrite:
		return true
	}
	return false
}

func (e KafkaOperation) String() string {
	return string(e)
}

func (e *KafkaOperation) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = KafkaOperation(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid KafkaOperation", str)
	}
	return nil
}

func (e KafkaOperation) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
