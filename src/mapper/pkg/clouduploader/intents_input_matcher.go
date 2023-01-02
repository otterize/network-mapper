package clouduploader

import (
	"fmt"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
)

// IntentsMatcher Implement gomock.Matcher interface for []cloudclient.IntentInput
type IntentsMatcher struct {
	expected []cloudclient.IntentInput
}

func NilCompare[T comparable](a *T, b *T) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func compareIntentInput(a cloudclient.IntentInput, b cloudclient.IntentInput) bool {
	return NilCompare(a.ClientName, b.ClientName) &&
		NilCompare(a.Namespace, b.Namespace) &&
		NilCompare(a.ServerName, b.ServerName) &&
		NilCompare(a.ServerNamespace, b.ServerNamespace)
}

func (m IntentsMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	actualIntents, ok := x.([]cloudclient.IntentInput)
	if !ok {
		return false
	}
	expectedIntents := m.expected
	if len(actualIntents) != len(expectedIntents) {
		return false
	}

	for _, expected := range expectedIntents {
		found := false
		for _, actual := range actualIntents {
			if compareIntentInput(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (m IntentsMatcher) String() string {
	return prettyPrint(m)
}

func prettyPrint(m IntentsMatcher) string {
	expected := m.expected
	var result string
	itemFormat := "IntentInput{ClientName: %s, ServerName: %s, Namespace: %s, ServerNamespace: %s},"
	for _, intent := range expected {
		var clientName, namespace, serverName, serverNamespace string
		if intent.ClientName != nil {
			clientName = *intent.ClientName
		}
		if intent.Namespace != nil {
			namespace = *intent.Namespace
		}
		if intent.ServerName != nil {
			serverName = *intent.ServerName
		}
		if intent.ServerNamespace != nil {
			serverNamespace = *intent.ServerNamespace
		}
		result += fmt.Sprintf(itemFormat, clientName, serverName, namespace, serverNamespace)
	}

	return result
}

func (m IntentsMatcher) Got(got interface{}) string {
	actual, ok := got.([]cloudclient.IntentInput)
	if !ok {
		return fmt.Sprintf("Not an []cloudclient.IntentInput, Got: %v", got)
	}

	return prettyPrint(IntentsMatcher{actual})
}

func GetMatcher(expected []cloudclient.IntentInput) IntentsMatcher {
	return IntentsMatcher{expected}
}
