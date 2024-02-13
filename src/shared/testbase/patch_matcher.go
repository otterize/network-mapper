package testbase

import (
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientPatch struct {
	client.Patch
}

func (p ClientPatch) Matches(x interface{}) bool {
	patch := x.(client.Patch)
	actualData, err := patch.Data(nil)
	if err != nil {
		return false
	}

	expectedData, err := p.Data(nil)
	if err != nil {
		return false
	}

	return string(actualData) == string(expectedData) && patch.Type() == p.Type()
}

func (p ClientPatch) String() string {
	data, err := p.Data(nil)
	if err != nil {
		return "format error"
	}
	return string(data)
}

func MatchPatch(patch client.Patch) gomock.Matcher {
	return ClientPatch{patch}
}
