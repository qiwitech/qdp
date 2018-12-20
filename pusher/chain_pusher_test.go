package pusher

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/pt"
)

func TestChaintPusher(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	c := mocks.NewMockChain(mock)

	c.EXPECT().PutTo(pt.AccID(20), gomock.Any())
	c.EXPECT().PutTo(pt.AccID(30), gomock.Any())

	p := NewChainReceiversPusher(c)

	txns := []pt.Txn{{Sender: 10, Receiver: 20}, {Sender: 10, Receiver: 30}}
	err := p.Push(context.TODO(), txns)
	assert.NoError(t, err)
}
