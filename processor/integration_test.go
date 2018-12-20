package processor

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/pt"
)

func TestChainIntegration(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	c := mocks.NewMockChain(mock)

	p := NewProcessor(c)

	respHash := pt.HashFromString("123456abcdef")

	c.EXPECT().GetLastHash(gomock.Any()).Times(1).Return(respHash)

	h, err := p.GetPrevHash(context.TODO(), 4)
	assert.NoError(t, err)
	assert.Equal(t, respHash, h)

	c.EXPECT().GetBalance(gomock.Any()).Times(1).Return(int64(10))

	b, err := p.GetBalance(context.TODO(), 4)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), b)
}

func TestPusherIntegration(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	c := mocks.NewMockChain(mock)
	push := mocks.NewMockPusher(mock)

	p := NewProcessor(c)

	p.SetPusher(push)

	respTxn := &pt.Txn{ID: 2, Sender: 10, Balance: 1000}

	c.EXPECT().GetLastTxn(gomock.Any()).Times(1).Return(respTxn)
	c.EXPECT().GetBalance(gomock.Any()).Times(1).Return(respTxn.Balance)
	c.EXPECT().ListUnspentTxns(gomock.Any()).Times(1).Return(nil)
	c.EXPECT().PutTo(pt.AccID(10), gomock.Any())

	push.EXPECT().Push(gomock.Any(), gomock.Any())

	resp, err := p.ProcessTransfer(context.TODO(), pt.Transfer{
		Sender: 10,
		Batch:  []*pt.TransferItem{{Receiver: 20, Amount: 300}},
	})
	assert.NoError(t, err)
	assert.Equal(t, pt.TransferResult{
		TxnID: pt.TxnID{AccID: 10, ID: 3},
		Hash:  pt.HashFromString("65145111132e2def3f240ebc8007e40b718a4b1087fa87a502605e6eb378a485"),
	}, resp)
}

func TestPreloaderIntegration(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	c := mocks.NewMockChain(mock)
	prel := mocks.NewMockPreloader(mock)

	p := NewProcessor(c)

	p.SetPreloader(prel)

	respTxn := &pt.Txn{ID: 2, Sender: 10, Balance: 1000}

	c.EXPECT().GetLastTxn(gomock.Any()).Times(1).Return(respTxn)
	c.EXPECT().GetBalance(gomock.Any()).Times(1).Return(respTxn.Balance)
	c.EXPECT().ListUnspentTxns(gomock.Any()).Times(1).Return(nil)
	c.EXPECT().PutTo(pt.AccID(10), gomock.Any())

	prel.EXPECT().Preload(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, acc pt.AccID) {
		assert.Equal(t, pt.AccID(10), acc)
	}).Return(nil)

	resp, err := p.ProcessTransfer(context.TODO(), pt.Transfer{
		Sender: 10,
		Batch:  []*pt.TransferItem{{Receiver: 20, Amount: 300}},
	})
	assert.NoError(t, err)
	assert.Equal(t, pt.TransferResult{
		TxnID: pt.TxnID{AccID: 10, ID: 3},
		Hash:  pt.HashFromString("65145111132e2def3f240ebc8007e40b718a4b1087fa87a502605e6eb378a485"),
	}, resp)
}
