package bigchain

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
	"github.com/qiwitech/qdp/pt"
)

func TestTxnsConvertation(t *testing.T) {
	txns := []*chainpb.Txn{
		{ID: 4, Sender: 5, Receiver: 8, Amount: 10, Balance: 100, SpentBy: 3, Hash: sliceFromString("1423"), PrevHash: sliceFromString("6798")},
		{ID: 5, Sender: 5, Receiver: 9, Amount: 11, Balance: 89, SpentBy: 1, Hash: sliceFromString("679811"), PrevHash: sliceFromString("123456")},
	}

	res := pbTxnsToPt(txns)

	assert.Equal(t, []pt.Txn{
		{ID: 4, Sender: 5, Receiver: 8, Amount: 10, Balance: 100, SpentBy: 3, Hash: pt.HashFromString("1423"), PrevHash: pt.HashFromString("6798")},
		{ID: 5, Sender: 5, Receiver: 9, Amount: 11, Balance: 89, SpentBy: 1, Hash: pt.HashFromString("679811"), PrevHash: pt.HashFromString("123456")},
	}, res)

	// nil case
	res = pbTxnsToPt(nil)
	assert.Nil(t, res)
}

func TestSettingsConvertation(t *testing.T) {
	sett := &chainpb.Settings{
		ID:                 4,
		Account:            6,
		PublicKey:          []byte("abcb1212"),
		Hash:               sliceFromString("123456"),
		PrevHash:           sliceFromString("654321"),
		DataHash:           sliceFromString("345678"),
		Sign:               sliceFromString("567890"),
		VerifyTransferSign: true,
	}

	res := pbSettingsToPt(sett)

	assert.Equal(t, &pt.Settings{
		ID:                 4,
		Account:            6,
		PublicKey:          []byte("abcb1212"),
		Hash:               pt.HashFromString("123456"),
		PrevHash:           pt.HashFromString("654321"),
		DataHash:           pt.HashFromString("345678"),
		Sign:               pt.SignFromString("567890"),
		VerifyTransferSign: true,
	}, res)

	// nil case
	res = pbSettingsToPt(nil)
	assert.Nil(t, res)
}

func TestDup(t *testing.T) {
	val := []byte{1, 2, 3}
	d := dup(val)

	assert.False(t, &val[0] == &d[0])
	assert.Equal(t, val, d)

	n := dup(nil)
	assert.Nil(t, n)
}

func TestFetchErr(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	pdb := mocks.NewMockPlutoDBServiceInterface(mock)

	c := New(pdb)

	testErr := errors.New("test err")

	pdb.EXPECT().Fetch(gomock.Any(), gomock.Any()).Times(1).Return(nil, testErr)

	_, _, err := c.Fetch(context.TODO(), 5, 12)
	assert.EqualError(t, err, "fetch client: "+testErr.Error())
}

func TestFetchOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	pdb := mocks.NewMockPlutoDBServiceInterface(mock)

	c := New(pdb)

	testResp := &plutodbpb.FetchResponse{
		Txns:     []*chainpb.Txn{{ID: 20, Sender: 1000, Balance: 10}},
		Settings: &chainpb.Settings{ID: 10, Account: 23},
	}

	pdb.EXPECT().Fetch(gomock.Any(), gomock.Any()).Times(1).Return(testResp, nil).Do(func(ctx context.Context, req *plutodbpb.FetchRequest) {
		assert.Equal(t, &plutodbpb.FetchRequest{
			Account: 5,
			Limit:   12,
		}, req)
		assert.True(t, ctx == context.TODO())
	})

	txns, sett, err := c.Fetch(context.TODO(), 5, 12)
	assert.NoError(t, err)
	assert.Equal(t, []pt.Txn{
		{ID: 20, Sender: 1000, Balance: 10},
	}, txns)
	assert.Equal(t, &pt.Settings{ID: 10, Account: 23}, sett)
}

func sliceFromString(s string) []byte {
	h := pt.HashFromString(s)
	return h[:]
}
