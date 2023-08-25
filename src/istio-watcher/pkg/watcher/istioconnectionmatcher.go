package istiowatcher

import (
	"fmt"
	"github.com/otterize/network-mapper/src/istio-watcher/mapperclient"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

// IstioConnectionResultMatcher Implement gomock.Matcher interface for []mapperclient.IstioConnectionResults
type IstioConnectionResultMatcher struct {
	mapperclient.IstioConnectionResults
}

func (m *IstioConnectionResultMatcher) Matches(x interface{}) bool {
	actual, ok := x.(mapperclient.IstioConnectionResults)
	if !ok {
		return false
	}

	if len(actual.Results) != len(m.Results) {
		return false
	}

	for _, actualResult := range actual.Results {
		anyResultsEqual := lo.SomeBy(m.Results, func(expectedResult mapperclient.IstioConnection) bool {
			return compareConnections(actualResult, expectedResult)
		})
		if !anyResultsEqual {
			return false
		}
	}

	return true
}

func compareConnections(actualResult mapperclient.IstioConnection, expectedResult mapperclient.IstioConnection) bool {
	if actualResult.SrcWorkload != expectedResult.SrcWorkload {
		return false
	}
	if actualResult.SrcWorkloadNamespace != expectedResult.SrcWorkloadNamespace {
		return false
	}
	if actualResult.DstWorkload != expectedResult.DstWorkload {
		return false
	}
	if actualResult.DstWorkloadNamespace != expectedResult.DstWorkloadNamespace {
		return false
	}
	if actualResult.Path != expectedResult.Path {
		return false
	}
	if len(actualResult.Methods) != len(expectedResult.Methods) {
		return false
	}
	slices.Sort(actualResult.Methods)
	slices.Sort(expectedResult.Methods)
	for j, actualMethod := range actualResult.Methods {
		expectedMethod := expectedResult.Methods[j]
		if actualMethod != expectedMethod {
			return false
		}
	}

	// We ignore last seen during testing
	return true
}

func (m *IstioConnectionResultMatcher) String() string {
	return fmt.Sprintf("%v", m.Results)
}

func GetMatcher(results mapperclient.IstioConnectionResults) *IstioConnectionResultMatcher {
	return &IstioConnectionResultMatcher{results}
}
