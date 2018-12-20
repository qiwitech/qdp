package remotepusher

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
)

func TestTxnsFromProto(t *testing.T) {
	res, err := txnsFromProto([]*chainpb.Txn{
		{ID: 10},
	})

	assert.NoError(t, err)
	assert.Equal(t, []pt.Txn{
		{ID: 10},
	}, res)
}

func TestTxnsToProto(t *testing.T) {
	res := txnsToProto([]pt.Txn{
		{ID: 10},
	})

	assert.Equal(t, []*chainpb.Txn{
		{ID: 10, PrevHash: pt.ZeroHash[:]},
	}, res)
}

type fakeFailingPusher struct{}

func (p *fakeFailingPusher) Push(ctx context.Context, txns []pt.Txn) error {
	return errors.New("fake internal error")
}

type stubPusher struct{}

func (p *stubPusher) Push(ctx context.Context, txns []pt.Txn) error {
	return nil
}

func TestServicePushInternalError(t *testing.T) {
	s := NewService(&fakeFailingPusher{})

	resp, err := s.Push(context.TODO(), &pusherpb.PushRequest{
		Txns: []*chainpb.Txn{
			{ID: 1, Sender: 20, Receiver: 20, Amount: 40, PrevHash: pt.ZeroHash[:]},
			{ID: 2, Sender: 30, Receiver: 40, Amount: 400, PrevHash: pt.ZeroHash[:]},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, &pusherpb.PushResponse{
		Status: &pusherpb.Status{
			Code:    int32(pusherpb.PushCode_INTERNAL_ERROR),
			Message: "pusher: fake internal error",
		},
	}, resp)
}

func TestServicePush(t *testing.T) {
	s := NewService(&stubPusher{})

	resp, err := s.Push(context.TODO(), &pusherpb.PushRequest{
		Txns: []*chainpb.Txn{
			{ID: 1, Sender: 20, Receiver: 20, Amount: 40, PrevHash: pt.ZeroHash[:]},
			{ID: 2, Sender: 30, Receiver: 40, Amount: 400, PrevHash: pt.ZeroHash[:]},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, &pusherpb.PushResponse{
		Status: &pusherpb.Status{
			Code:    int32(pusherpb.PushCode_OK),
			Message: "",
		},
	}, resp)
}

func TestServicePushInvalidTxn(t *testing.T) {
	s := NewService(&fakeFailingPusher{})

	resp, err := s.Push(context.TODO(), &pusherpb.PushRequest{
		Txns: []*chainpb.Txn{
			{ID: 1, Sender: 20, Receiver: 20, Amount: 40, PrevHash: []byte("10")},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, &pusherpb.PushResponse{
		Status: &pusherpb.Status{
			Code:    int32(pusherpb.PushCode_INTERNAL_ERROR),
			Message: "validator: invalid prev_hash size 2 for txn_id=1, sender_id=20",
		},
	}, resp)
}
