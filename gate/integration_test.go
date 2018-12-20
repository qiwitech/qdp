package gate

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/chain"
	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/processor"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/pt"
)

func TestProcessorIntegration(t *testing.T) {
	c := chain.NewChain()
	p := processor.NewProcessor(c)
	g := NewGate(p, nil)

	c.PutTo(10, []pt.Txn{{
		ID:       1,
		Sender:   10,
		Receiver: 20,
		Amount:   100,
		Balance:  1000,
	}})

	resp, err := g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{
		Sender: 10,
		Batch:  []*gatepb.TransferItem{{Receiver: 30, Amount: 10}},
	})
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.TransferResponse{
		Status:  &gatepb.Status{},
		TxnId:   "10_2",
		Account: 10,
		Id:      2,
		Hash:    "c2345333b8a30ef05830c11bd18420ada30720b860c213452b1badab54e5f557",
	}, resp)
}

func TestRouterIntegrationSelfNode(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	c := chain.NewChain()
	p := processor.NewProcessor(c)
	g := NewGate(p, nil)

	r := mocks.NewMockRouter(mock)

	g.SetRouter(r)

	// set prev transaction
	c.PutTo(10, []pt.Txn{{
		ID:       1,
		Sender:   10,
		Receiver: 20,
		Amount:   100,
		Balance:  1000,
	}})

	r.EXPECT().GetHostByKey(gomock.Any()).Do(func(k string) {
		assert.Equal(t, "10", k)
	}).Return("some_host")
	r.EXPECT().IsSelf(gomock.Any()).Do(func(h string) {
		assert.Equal(t, "some_host", h)
	}).Return(true)

	resp, err := g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{
		Sender: 10,
		Batch:  []*gatepb.TransferItem{{Receiver: 30, Amount: 10}},
	})
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.TransferResponse{
		Status:  &gatepb.Status{},
		TxnId:   "10_2",
		Account: 10,
		Id:      2,
		Hash:    "c2345333b8a30ef05830c11bd18420ada30720b860c213452b1badab54e5f557",
	}, resp)
}

func TestRouterIntegrationReroute(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	c := chain.NewChain()
	p := processor.NewProcessor(c)
	g := NewGate(p, nil)

	r := mocks.NewMockRouter(mock)

	g.SetRouter(r)

	// set prev transaction
	c.PutTo(10, []pt.Txn{{
		ID:       1,
		Sender:   10,
		Receiver: 20,
		Amount:   100,
		Balance:  1000,
	}})

	r.EXPECT().GetHostByKey(gomock.Any()).Do(func(k string) {
		assert.Equal(t, "10", k)
	}).Return("some_host")
	r.EXPECT().IsSelf(gomock.Any()).Do(func(h string) {
		assert.Equal(t, "some_host", h)
	}).Return(false)
	r.EXPECT().Nodes().Return([]string{"host_a", "host_b", "host_c"})

	resp, err := g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{
		Sender: 10,
		Batch:  []*gatepb.TransferItem{{Receiver: 30, Amount: 10}},
	})
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.TransferResponse{Status: &gatepb.Status{
		Code:    gatepb.TransferCode_SEE_OTHER,
		Message: "route error: see other node some_host",
		Details: []*any.Any{{TypeUrl: "gate.RouteMap", Value: []byte("\032\tsome_host\"\006host_a\"\006host_b\"\006host_c")}},
	}}, resp)
}
