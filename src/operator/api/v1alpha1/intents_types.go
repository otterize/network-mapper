/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type IntentType string

const (
	IntentTypeHTTP  IntentType = "HTTP"
	IntentTypeKafka IntentType = "Kafka"
	IntentTypeGRPC  IntentType = "gRPC"
	IntentTypeRedis IntentType = "Redis"
)

type KafkaOperation string

const (
	KafkaOperationConsume         KafkaOperation = "consume"
	KafkaOperationProduce         KafkaOperation = "produce"
	KafkaOperationCreate          KafkaOperation = "create"
	KafkaOperationAlter           KafkaOperation = "alter"
	KafkaOperationDelete          KafkaOperation = "delete"
	KafkaOperationDescribe        KafkaOperation = "describe"
	KafkaOperationClusterAction   KafkaOperation = "ClusterAction"
	KafkaOperationDescribeConfigs KafkaOperation = "DescribeConfigs"
	KafkaOperationAlterConfigs    KafkaOperation = "AlterConfigs"
	KafkaOperationIdempotentWrite KafkaOperation = "IdempotentWrite"
)

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

// IntentsSpec defines the desired state of Intents
type IntentsSpec struct {
	Service Service `json:"service"`
}

type Service struct {
	Name  string   `json:"name"`
	Calls []Intent `json:"calls"`
}

type HTTPResource struct {
	Path   string     `json:"path"`
	Method HTTPMethod `json:"method"`
}

type Intent struct {
	Type   IntentType `json:"type"`
	Server string     `json:"server"`

	//+optional
	Topics []KafkaTopic `json:"topics"`
	//+optional
	HTTPResources []HTTPResource `json:"http_resources"`
}

type KafkaTopic struct {
	Name      string         `json:"name"`
	Operation KafkaOperation `json:"operation"`
}

// IntentsStatus defines the observed state of Intents
type IntentsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Intents is the Schema for the intents API
type Intents struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntentsSpec   `json:"spec,omitempty"`
	Status IntentsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IntentsList contains a list of Intents
type IntentsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Intents `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Intents{}, &IntentsList{})
}
