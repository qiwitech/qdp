package api

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/proto/apipb"
	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/proto/metadbpb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
)

func TestProcessTransferError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	respErr := errors.New("some test error")
	proc.EXPECT().ProcessTransfer(gomock.Any(), gomock.Any()).Return(nil, respErr)

	res, err := g.ProcessTransfer(context.TODO(), &apipb.TransferRequest{})

	assert.EqualError(t, err, "api: "+respErr.Error())
	assert.Nil(t, res)
}

func TestProcessTransferGateErr(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	proc.EXPECT().ProcessTransfer(gomock.Any(), gomock.Any()).Return(&gatepb.TransferResponse{Status: &gatepb.Status{Code: 3, Message: "message"}, TxnId: "txn_id"}, nil)

	res, err := g.ProcessTransfer(context.TODO(), &apipb.TransferRequest{
		Batch: []*apipb.TransferItem{{}},
	})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.TransferResponse{Status: &apipb.Status{Code: 3, Message: "message"}}, res)
}

func TestProcessTransferOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	proc.EXPECT().ProcessTransfer(gomock.Any(), gomock.Any()).Return(&gatepb.TransferResponse{Status: &gatepb.Status{}, TxnId: "txn_id", Account: 3, Id: 5}, nil)

	res, err := g.ProcessTransfer(context.TODO(), &apipb.TransferRequest{
		Batch: []*apipb.TransferItem{{}},
	})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.TransferResponse{Status: &apipb.Status{}, TxnId: "txn_id"}, res)
}

func TestGetPrevHashError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	respErr := errors.New("some test error")
	proc.EXPECT().GetPrevHash(gomock.Any(), gomock.Any()).Return(nil, respErr)

	res, err := g.GetPrevHash(context.TODO(), &apipb.GetPrevHashRequest{})

	assert.EqualError(t, err, "api: "+respErr.Error())
	assert.Nil(t, res)
}

func TestGetPrevHashOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	proc.EXPECT().GetPrevHash(gomock.Any(), gomock.Any()).Return(&gatepb.GetPrevHashResponse{Status: &gatepb.Status{Code: 4, Message: "message"}, Hash: "prev_hash"}, nil)

	res, err := g.GetPrevHash(context.TODO(), &apipb.GetPrevHashRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.GetPrevHashResponse{Status: &apipb.Status{Code: 4, Message: "message"}, Hash: "prev_hash"}, res)
}

func TestGetBalanceError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	respErr := errors.New("some test error")
	proc.EXPECT().GetBalance(gomock.Any(), gomock.Any()).Return(nil, respErr)

	res, err := g.GetBalance(context.TODO(), &apipb.GetBalanceRequest{})

	assert.EqualError(t, err, "api: "+respErr.Error())
	assert.Nil(t, res)
}

func TestGetBalanceOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	proc.EXPECT().GetBalance(gomock.Any(), gomock.Any()).Return(&gatepb.GetBalanceResponse{Status: &gatepb.Status{Code: 5, Message: "message"}, Balance: 300}, nil)

	res, err := g.GetBalance(context.TODO(), &apipb.GetBalanceRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.GetBalanceResponse{Status: &apipb.Status{Code: 5, Message: "message"}, Balance: 300}, res)
}

func TestUpdateSettingsError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	respErr := errors.New("some test error")
	proc.EXPECT().UpdateSettings(gomock.Any(), gomock.Any()).Return(nil, respErr)

	res, err := g.UpdateSettings(context.TODO(), &apipb.SettingsRequest{})

	assert.EqualError(t, err, "api: "+respErr.Error())
	assert.Nil(t, res)
}

func TestUpdateSettingsOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	proc.EXPECT().UpdateSettings(gomock.Any(), gomock.Any()).Return(&gatepb.SettingsResponse{Status: &gatepb.Status{Code: 5, Message: "message"}, SettingsId: "settings_id"}, nil)

	res, err := g.UpdateSettings(context.TODO(), &apipb.SettingsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.SettingsResponse{Status: &apipb.Status{Code: 5, Message: "message"}, SettingsId: "settings_id"}, res)
}

func TestGetLastSettingsError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	respErr := errors.New("some test error")
	proc.EXPECT().GetLastSettings(gomock.Any(), gomock.Any()).Return(nil, respErr)

	res, err := g.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{})

	assert.EqualError(t, err, "api: "+respErr.Error())
	assert.Nil(t, res)
}

func TestGetLastSettingsOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)

	g := NewService(proc)

	proc.EXPECT().GetLastSettings(gomock.Any(), gomock.Any()).Return(
		&gatepb.GetLastSettingsResponse{
			Status:  &gatepb.Status{Code: 6, Message: "message"},
			Account: 40,
			Id:      10,
			Hash:    "hash",
		}, nil)

	res, err := g.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.GetLastSettingsResponse{
		Status:  &apipb.Status{Code: 6, Message: "message"},
		Account: 40,
		Id:      10,
		Hash:    "hash",
	}, res)
}

func TestGetByMetaKeyOk(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)
	pdb := mocks.NewMockPlutoDBServiceInterface(mock)
	meta := mocks.NewMockMetaDBServiceInterface(mock)

	g := NewService(proc)

	g.SetPlutoDBClient(pdb)
	g.SetMetaDBClient(meta)

	meta.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Do(func(_ context.Context, req *metadbpb.GetMultiRequest) {
		assert.Equal(t, &metadbpb.GetMultiRequest{
			Prefix: TxnsMetaPrefix,
			Keys:   [][]byte{[]byte("meta_key")},
		}, req)
	}).Return(&metadbpb.GetMultiResponse{
		Status: &metadbpb.Status{},
		Results: []*metadbpb.Data{
			{
				Key: []byte("meta_key"),
				Index: []*metadbpb.Pair{
					{Key: []byte("sender"), Value: tobytes(uint64(4))},
					{Key: []byte("id"), Value: tobytes(uint64(6))},
					{Key: []byte("prevhash"), Value: []byte("5f7d1337b3f6e5f6f2e999906c8167f80990c994fa52151396638e6b75742824")},
					{Key: []byte("moon"), Value: []byte("some_moon_phase")},
				},
				Fields: []*metadbpb.Pair{
					{Key: []byte("field1"), Value: []byte("val1")},
				},
			},
		},
	}, nil)

	pdb.EXPECT().GetTxnMulti(gomock.Any(), gomock.Any()).Do(func(_ context.Context, req *plutodbpb.GetTxnMultiRequest) {
		assert.Equal(t, &plutodbpb.GetTxnMultiRequest{
			IDs: []*chainpb.TxnID{{Account: 4, ID: 6}},
		}, req)
	}).Return(&plutodbpb.GetTxnMultiResponse{
		Status: &plutodbpb.Status{},
		Txns: []*chainpb.Txn{
			{Sender: 4, ID: 6},
		},
	}, nil)

	resp, err := g.GetByMetaKey(context.TODO(), &apipb.GetByMetaKeyRequest{
		Keys: [][]byte{[]byte("meta_key")},
	})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.GetByMetaKeyResponse{
		Status: &apipb.Status{},
		Txns: []*apipb.Txn{
			{
				Sender: "4", Id: "6",
				Receiver: "0", Amount: "0", Balance: "0", SpentBy: "0",
				Meta: &apipb.Meta{
					Key: []byte("meta_key"),
					Index: map[string][]byte{
						"sender":   tobytes(uint64(4)),
						"id":       tobytes(uint64(6)),
						"prevhash": []byte("5f7d1337b3f6e5f6f2e999906c8167f80990c994fa52151396638e6b75742824"),
						"moon":     []byte("some_moon_phase"),
					},
					Data: map[string][]byte{
						"field1": []byte("val1"),
					},
				},
			},
		},
	}, resp)
}

func TestGetByMetaKeyOkWithoutPlutodb(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockProcessorServiceInterface(mock)
	meta := mocks.NewMockMetaDBServiceInterface(mock)

	g := NewService(proc)

	g.SetMetaDBClient(meta)

	meta.EXPECT().GetMulti(gomock.Any(), gomock.Any()).Do(func(_ context.Context, req *metadbpb.GetMultiRequest) {
		assert.Equal(t, &metadbpb.GetMultiRequest{
			Prefix: TxnsMetaPrefix,
			Keys:   [][]byte{[]byte("meta_key")},
		}, req)
	}).Return(&metadbpb.GetMultiResponse{
		Status: &metadbpb.Status{},
		Results: []*metadbpb.Data{
			{
				Key: []byte("meta_key"),
				Index: []*metadbpb.Pair{
					{Key: []byte("sender"), Value: tobytes(uint64(4))},
					{Key: []byte("id"), Value: tobytes(uint64(6))},
					{Key: []byte("prevhash"), Value: []byte("5f7d1337b3f6e5f6f2e999906c8167f80990c994fa52151396638e6b75742824")},
					{Key: []byte("moon"), Value: []byte("some_moon_phase")},
				},
				Fields: []*metadbpb.Pair{
					{Key: []byte("field1"), Value: []byte("val1")},
				},
			},
		},
	}, nil)

	resp, err := g.GetByMetaKey(context.TODO(), &apipb.GetByMetaKeyRequest{
		Keys: [][]byte{[]byte("meta_key")},
	})

	assert.NoError(t, err)
	assert.Equal(t, &apipb.GetByMetaKeyResponse{
		Status: &apipb.Status{},
		Txns: []*apipb.Txn{
			{
				Meta: &apipb.Meta{
					Key: []byte("meta_key"),
					Index: map[string][]byte{
						"sender":   tobytes(uint64(4)),
						"id":       tobytes(uint64(6)),
						"prevhash": []byte("5f7d1337b3f6e5f6f2e999906c8167f80990c994fa52151396638e6b75742824"),
						"moon":     []byte("some_moon_phase"),
					},
					Data: map[string][]byte{
						"field1": []byte("val1"),
					},
				},
			},
		},
	}, resp)
}

func TestRemoveZeros(t *testing.T) {
	assert.Equal(t, []*apipb.Txn(nil), removeZeros(nil))
	assert.Equal(t, []*apipb.Txn{}, removeZeros([]*apipb.Txn{}))
	assert.Equal(t, []*apipb.Txn{{Id: "1"}}, removeZeros([]*apipb.Txn{{Id: "1"}}))
	assert.Equal(t, []*apipb.Txn{{Id: "1"}, {Id: "2"}}, removeZeros([]*apipb.Txn{{Id: "1"}, {Id: "2"}}))
	assert.Equal(t, []*apipb.Txn{{Id: "1"}, {Id: "2"}, {Id: "3"}}, removeZeros([]*apipb.Txn{{Id: "1"}, {Id: "2"}, {Id: "3"}}))
	assert.Equal(t, []*apipb.Txn{{Id: "1"}, {Id: "3"}}, removeZeros([]*apipb.Txn{{Id: "1"}, {Id: "0"}, {Id: "3"}}))
	assert.Equal(t, []*apipb.Txn{}, removeZeros([]*apipb.Txn{{Id: ""}, {Id: "0"}, {Id: "0"}}))
}
