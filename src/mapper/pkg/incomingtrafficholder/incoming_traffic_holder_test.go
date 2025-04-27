package incomingtrafficholder

import (
	"context"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

const (
	testServerName      = "testServerName"
	testServerNamespace = "testServerNamespace"
	ipAddressA          = "1.1.1.1"
	ipAddressB          = "2.2.2.2"
	ipAddressC          = "3.3.3.3"
	testUploadInterval  = time.Millisecond * 1
	testMaxTimeout      = time.Millisecond * 10
)

type IncomingTrafficHolderSuite struct {
	suite.Suite
	holder *IncomingTrafficIntentsHolder
}

func (s *IncomingTrafficHolderSuite) SetupTest() {
	s.holder = NewIncomingTrafficIntentsHolder()
}

func (s *IncomingTrafficHolderSuite) TestIncomingTrafficHolder() {
	timestamp := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	server := model.OtterizeServiceIdentity{
		Name:      testServerName,
		Namespace: testServerNamespace,
	}
	incomingA := IncomingTrafficIntent{
		Server:   server,
		LastSeen: timestamp,
		IP:       ipAddressA,
		SrcPorts: []int64{1, 2, 3},
	}

	var called bool
	uploaded := make([]TimestampedIncomingTrafficIntent, 0)
	validationFunc := func(_ context.Context, intents []TimestampedIncomingTrafficIntent) {
		uploaded = intents
		called = true
	}
	s.holder.RegisterNotifyIntents(validationFunc)
	s.holder.AddIntent(incomingA)

	timeoutContext, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()
	s.holder.PeriodicIntentsUpload(timeoutContext, testUploadInterval)
	s.Require().True(called)
	s.Require().Len(uploaded, 1)
	s.Require().Equal(incomingA, uploaded[0].Intent)
	s.Require().Equal(timestamp, uploaded[0].Timestamp)
	s.Require().Equal(incomingA.Server.Name, uploaded[0].Intent.Server.Name)
	s.Require().Equal(incomingA.Server.Namespace, uploaded[0].Intent.Server.Namespace)
	s.Require().Equal(incomingA.IP, uploaded[0].Intent.IP)
	s.Require().NotNil(uploaded[0].ConnectionsCount)
	s.Require().Equal(3, lo.FromPtr(uploaded[0].ConnectionsCount.Current))

	called = false
	uploaded = make([]TimestampedIncomingTrafficIntent, 0)
	s.holder.PeriodicIntentsUpload(timeoutContext, testUploadInterval)
	s.Require().False(called)
	s.Require().Len(uploaded, 0)

	incomingB := IncomingTrafficIntent{
		Server:   server,
		LastSeen: timestamp.Add(time.Second),
		IP:       ipAddressB,
		SrcPorts: []int64{10},
	}

	incomingC := IncomingTrafficIntent{
		Server:   server,
		LastSeen: timestamp.Add(time.Second * 2),
		IP:       ipAddressC,
		SrcPorts: []int64{10},
	}

	s.holder.AddIntent(incomingB)
	s.holder.AddIntent(incomingC)

	called = false
	uploaded = make([]TimestampedIncomingTrafficIntent, 0)
	anotherContext, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()

	s.holder.PeriodicIntentsUpload(anotherContext, testUploadInterval)
	s.Require().True(called)
	s.Require().Len(uploaded, 2)
	uploadedB, found := lo.Find(uploaded, func(intent TimestampedIncomingTrafficIntent) bool {
		return intent.Intent.IP == ipAddressB
	})
	s.Require().True(found)
	s.Require().Equal(incomingB.Server.Name, uploadedB.Intent.Server.Name)
	s.Require().Equal(incomingB.Server.Namespace, uploadedB.Intent.Server.Namespace)
	s.Require().Equal(incomingB.IP, uploadedB.Intent.IP)
	s.Require().NotNil(uploadedB.ConnectionsCount)
	s.Require().Equal(1, lo.FromPtr(uploadedB.ConnectionsCount.Current))

	uploadedC, found := lo.Find(uploaded, func(intent TimestampedIncomingTrafficIntent) bool {
		return intent.Intent.IP == ipAddressC
	})
	s.Require().True(found)
	s.Require().Equal(incomingC.Server.Name, uploadedC.Intent.Server.Name)
	s.Require().Equal(incomingC.Server.Namespace, uploadedC.Intent.Server.Namespace)
	s.Require().Equal(incomingC.IP, uploadedC.Intent.IP)
	s.Require().NotNil(uploadedC.ConnectionsCount)
	s.Require().Equal(1, lo.FromPtr(uploadedC.ConnectionsCount.Current))
}

func (s *IncomingTrafficHolderSuite) TestReportOnlyLatest() {
	timestamp1 := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	timestamp2 := timestamp1.Add(time.Second)
	timestamp3 := timestamp2.Add(time.Second * 2)
	server := model.OtterizeServiceIdentity{
		Name:      testServerName,
		Namespace: testServerNamespace,
	}
	incoming := IncomingTrafficIntent{
		Server:   server,
		LastSeen: timestamp2,
		IP:       ipAddressA,
	}

	var called bool
	uploaded := make([]TimestampedIncomingTrafficIntent, 0)
	validationFunc := func(_ context.Context, intents []TimestampedIncomingTrafficIntent) {
		uploaded = intents
		called = true
	}

	s.holder.RegisterNotifyIntents(validationFunc)
	s.holder.AddIntent(incoming)
	incoming.LastSeen = timestamp1
	s.holder.AddIntent(incoming)

	timeoutContext, cancel := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel()
	s.holder.PeriodicIntentsUpload(timeoutContext, testUploadInterval)
	s.Require().True(called)
	s.Require().Len(uploaded, 1)
	s.Require().Equal(timestamp2, uploaded[0].Timestamp)

	incoming.LastSeen = timestamp3
	s.holder.AddIntent(incoming)

	timeoutContext1, cancel1 := context.WithTimeout(context.Background(), testMaxTimeout)
	defer cancel1()
	s.holder.PeriodicIntentsUpload(timeoutContext1, testUploadInterval)
	s.Require().True(called)
	s.Require().Len(uploaded, 1)
	s.Require().Equal(timestamp3, uploaded[0].Timestamp)
}

func TestIncomingTrafficHolderSuite(t *testing.T) {
	suite.Run(t, new(IncomingTrafficHolderSuite))
}
