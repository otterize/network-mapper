package clouduploader

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"golang.org/x/exp/constraints"
	"sort"
)

// IntentsMatcher Implement gomock.Matcher interface for []cloudclient.IntentInput
type IntentsMatcher struct {
	expected []cloudclient.IntentInput
}

func NilCompare[T constraints.Ordered](a *T, b *T) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if *a > *b {
		return 1
	}
	if *a < *b {
		return -1
	}
	return 0
}

func sortIntentInput(intents []cloudclient.IntentInput) {
	for _, intent := range intents {
		if intent.Type == nil {
			continue
		}

		switch *intent.Type {
		case cloudclient.IntentTypeKafka:
			for _, topic := range intent.Topics {
				sort.Slice(topic.Operations, func(i, j int) bool {
					return NilCompare(topic.Operations[i], topic.Operations[j]) < 0
				})
			}
			sort.Slice(intent.Topics, func(i, j int) bool {
				res := NilCompare(intent.Topics[i].Name, intent.Topics[j].Name)
				if res != 0 {
					return res < 0
				}

				return len(intent.Topics[i].Operations) < len(intent.Topics[j].Operations)
			})
		case cloudclient.IntentTypeHttp:
			for _, resource := range intent.Resources {
				sort.Slice(resource.Methods, func(i, j int) bool {
					return NilCompare(resource.Methods[i], resource.Methods[j]) < 0
				})
			}
			sort.Slice(intent.Resources, func(i, j int) bool {
				res := NilCompare(intent.Resources[i].Path, intent.Resources[j].Path)
				if res != 0 {
					return res < 0
				}

				return len(intent.Resources[i].Methods) < len(intent.Resources[j].Methods)
			})
		}
	}
	sort.Slice(intents, func(i, j int) bool {
		res := NilCompare(intents[i].Namespace, intents[j].Namespace)
		if res != 0 {
			return res < 0
		}
		res = NilCompare(intents[i].ClientName, intents[j].ClientName)
		if res != 0 {
			return res < 0
		}
		res = NilCompare(intents[i].ServerName, intents[j].ServerName)
		if res != 0 {
			return res < 0
		}
		res = NilCompare(intents[i].ServerNamespace, intents[j].ServerNamespace)
		if res != 0 {
			return res < 0
		}
		res = NilCompare(intents[i].Type, intents[j].Type)
		if res != 0 {
			return res < 0
		}
		switch *intents[i].Type {
		case cloudclient.IntentTypeKafka:
			return len(intents[i].Topics) < len(intents[j].Topics)
		case cloudclient.IntentTypeHttp:
			return len(intents[i].Resources) < len(intents[j].Resources)
		default:
			panic("Unimplemented intent type")
		}
	})
}

func (m IntentsMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	actualDiscoveredIntents, ok := x.([]*cloudclient.DiscoveredIntentInput)
	if !ok {
		return false
	}
	expectedIntents := m.expected
	actualIntents := discoveredIntentsPtrToIntents(actualDiscoveredIntents)

	if len(actualIntents) != len(expectedIntents) {
		return false
	}

	sortIntentInput(actualIntents)
	sortIntentInput(expectedIntents)

	diff := cmp.Diff(expectedIntents, actualIntents)
	if diff != "" {
		fmt.Println(diff)
	}
	return cmp.Equal(expectedIntents, actualIntents)
}

func discoveredIntentsPtrToIntents(actualDiscoveredIntents []*cloudclient.DiscoveredIntentInput) []cloudclient.IntentInput {
	actualIntents := make([]cloudclient.IntentInput, 0)
	for _, intent := range actualDiscoveredIntents {
		intentObject := *intent.Intent
		actualIntents = append(actualIntents, intentObject)
	}
	return actualIntents
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
	actual, ok := got.([]*cloudclient.DiscoveredIntentInput)
	if !ok {
		return fmt.Sprintf("Not an []*cloudclient.DiscoveredIntentInput, Got: %v", got)
	}

	return prettyPrint(IntentsMatcher{discoveredIntentsPtrToIntents(actual)})
}

func GetMatcher(expected []cloudclient.IntentInput) IntentsMatcher {
	return IntentsMatcher{expected}
}
