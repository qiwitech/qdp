package gate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/chain"
	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/processor"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/pt"
)

func TestValidateSettings(t *testing.T) {
	_, err := settingsFromProto(&gatepb.SettingsRequest{
		PrevHash: strings.Repeat("s", 23),
	})
	assert.EqualError(t, err, "validator: prev_hash is too short (23<64)")

	_, err = settingsFromProto(&gatepb.SettingsRequest{
		PrevHash: strings.Repeat("s", 128),
	})
	assert.EqualError(t, err, "validator: prev_hash is too long (128>64)")

	_, err = settingsFromProto(&gatepb.SettingsRequest{
		PrevHash: strings.Repeat("s", 64),
	})
	assert.EqualError(t, err, "validator: invalid hash string")

	_, err = settingsFromProto(&gatepb.SettingsRequest{
		Sign: strings.Repeat("s", 23),
	})
	assert.EqualError(t, err, "validator: sign is too short (23<144)")

	_, err = settingsFromProto(&gatepb.SettingsRequest{
		Sign: strings.Repeat("s", 512),
	})
	assert.EqualError(t, err, "validator: sign is too long (512>144)")

	_, err = settingsFromProto(&gatepb.SettingsRequest{
		Sign: strings.Repeat("s", 144),
	})
	assert.EqualError(t, err, "validator: invalid sign string")
}

func TestSettingsFromProto(t *testing.T) {
	s, err := settingsFromProto(&gatepb.SettingsRequest{
		Account: 0,
	})

	assert.NoError(t, err)
	assert.Equal(t, &pt.Settings{}, s)
}

func TestProcessTransferBadRequest(t *testing.T) {
	g := NewGate(nil, nil)

	res, err := g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &gatepb.TransferResponse{
		Status: &gatepb.Status{
			Code:    gatepb.TransferCode_BAD_REQUEST,
			Message: "gate: validator: empty batch, no receivers",
		},
		TxnId: "",
		Hash:  "",
	}, res)
}

func TestProcessTransfer(t *testing.T) {
	g := NewGate(processor.NewProcessor(chain.NewChain()), nil)

	res, err := g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{
		Sender: 4,
		Batch:  []*gatepb.TransferItem{{Receiver: 10, Amount: 0}},
	})

	assert.NoError(t, err)
	assert.Equal(t, &gatepb.TransferResponse{
		Status:  &gatepb.Status{Code: gatepb.TransferCode_OK},
		TxnId:   "4_1",
		Account: 4,
		Id:      1,
		Hash:    "d7914f58c8e27e7a3b21aa5cf74021ae01dfb4958bbbf255473add1cc4123e28",
	}, res)
}

func TestValidateTransfer(t *testing.T) {
	_, err := transferFromProto(&gatepb.TransferRequest{Batch: nil})
	assert.EqualError(t, err, "validator: empty batch, no receivers")

	_, err = transferFromProto(&gatepb.TransferRequest{
		Batch:    []*gatepb.TransferItem{{}},
		PrevHash: strings.Repeat("s", 1024),
	})
	assert.EqualError(t, err, "validator: prev_hash is too long (1024>64)")

	_, err = transferFromProto(&gatepb.TransferRequest{
		Batch: []*gatepb.TransferItem{{}},
		Sign:  strings.Repeat("s", 1024),
	})
	assert.EqualError(t, err, "validator: sign is too long (1024>144)")

	_, err = transferFromProto(&gatepb.TransferRequest{
		Batch:    []*gatepb.TransferItem{{}},
		PrevHash: strings.Repeat("s", 64),
	})
	assert.EqualError(t, err, "validator: invalid hash string")

	_, err = transferFromProto(&gatepb.TransferRequest{
		Batch: []*gatepb.TransferItem{{}},
		Sign:  strings.Repeat("s", 144),
	})
	assert.EqualError(t, err, "validator: invalid sign string")
}

func TestTransferFromProto(t *testing.T) {
	res, err := transferFromProto(&gatepb.TransferRequest{
		Sender:   1000,
		Batch:    []*gatepb.TransferItem{{Receiver: 10, Amount: 100}},
		Sign:     "",
		PrevHash: "",
	})
	assert.NoError(t, err)
	assert.Equal(t, pt.NewSingleTransfer(1000, 10, 100), *res)
}

func TestUpdateSettings(t *testing.T) {
	g := NewGate(nil, processor.NewSettingsProcessor(chain.NewSettingsChain()))

	res, err := g.UpdateSettings(context.TODO(), &gatepb.SettingsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &gatepb.SettingsResponse{
		Status:     &gatepb.Status{Code: gatepb.TransferCode_OK},
		SettingsId: "0_1",
		Hash:       "3d31945c4761c707e118a725421f324af313207fcb9db862c8e952ade42c9ee6",
	}, res)
}

func TestUpdateSettingsBadRequest(t *testing.T) {
	g := NewGate(nil, processor.NewSettingsProcessor(chain.NewSettingsChain()))

	res, err := g.UpdateSettings(context.TODO(), &gatepb.SettingsRequest{PrevHash: "qwe"})

	assert.NoError(t, err)
	assert.Equal(t, &gatepb.SettingsResponse{
		Status: &gatepb.Status{
			Code:    gatepb.TransferCode_BAD_REQUEST,
			Message: "gate: validator: prev_hash is too short (3<64)",
		},
	}, res)
}

func TestUpdateSettingsRoutingError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	g := NewGate(nil, processor.NewSettingsProcessor(chain.NewSettingsChain()))

	r := mocks.NewMockRouter(mock)
	g.SetRouter(r)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("another-txn-host")
	r.EXPECT().IsSelf(gomock.Any()).Return(false)
	r.EXPECT().Nodes().Return([]string{"another-txn-host"})

	res, err := g.UpdateSettings(context.TODO(), &gatepb.SettingsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, "route error: see other node another-txn-host", res.Status.Message)
}

func TestUpdateSettingsErrors(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockSettingsProcessor(mock)

	g := NewGate(nil, proc)

	// invalid prev hash
	proc.EXPECT().ProcessSettings(context.TODO(), gomock.Any()).Return(pt.SettingsResult{}, processor.ErrInvalidSettingsPrevHash)

	res, err := g.UpdateSettings(context.TODO(), &gatepb.SettingsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &gatepb.SettingsResponse{
		Status: &gatepb.Status{
			Code:    gatepb.TransferCode_INVALID_PREV_HASH,
			Message: "gate: settings processor: invalid prev hash",
		},
	}, res)

	// internal error
	respErr := errors.New("some test error")
	proc.EXPECT().ProcessSettings(context.TODO(), gomock.Any()).Return(pt.SettingsResult{}, respErr)

	res, err = g.UpdateSettings(context.TODO(), &gatepb.SettingsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &gatepb.SettingsResponse{
		Status: &gatepb.Status{
			Code:    gatepb.TransferCode_INTERNAL_ERROR,
			Message: "gate: " + respErr.Error(),
		},
	}, res)
}

func TestGetPrevHash(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockTransferProcessor(mock)

	g := NewGate(proc, nil)

	req := &gatepb.GetPrevHashRequest{Account: 0}
	resp := &gatepb.GetPrevHashResponse{
		Status: &gatepb.Status{},
		Hash:   "0000000000000000000000000000000000000000000000000000000000000000",
	}

	proc.EXPECT().GetPrevHash(context.TODO(), pt.AccID(req.Account)).Return(pt.ZeroHash, nil)

	res, err := g.GetPrevHash(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	respErr := errors.New("some error")

	proc.EXPECT().GetPrevHash(context.TODO(), pt.AccID(req.Account)).Return(pt.ZeroHash, respErr)

	res, err = g.GetPrevHash(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.GetPrevHashResponse{Status: &gatepb.Status{
		Code:    gatepb.TransferCode_INTERNAL_ERROR,
		Message: "gate: " + respErr.Error(),
	}}, res)
}

func TestGetBalance(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockTransferProcessor(mock)

	g := NewGate(proc, nil)

	ctx := context.TODO()
	req := &gatepb.GetBalanceRequest{Account: 0}
	resp := &gatepb.GetBalanceResponse{
		Status:  &gatepb.Status{},
		Balance: 103,
	}

	proc.EXPECT().GetBalance(ctx, pt.AccID(req.Account)).Return(int64(103), nil)

	res, err := g.GetBalance(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	respErr := errors.New("some error")

	proc.EXPECT().GetBalance(ctx, pt.AccID(req.Account)).Return(int64(0), respErr)

	res, err = g.GetBalance(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.GetBalanceResponse{Status: &gatepb.Status{
		Code:    gatepb.TransferCode_INTERNAL_ERROR,
		Message: "gate: " + respErr.Error(),
	}}, res)

	// routing error
	r := mocks.NewMockRouter(mock)
	g.SetRouter(r)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("another-txn-host")
	r.EXPECT().IsSelf(gomock.Any()).Return(false)
	r.EXPECT().Nodes().Return([]string{"another-txn-host"})

	res, err = g.GetBalance(context.TODO(), &gatepb.GetBalanceRequest{Account: 10})
	// TODO(outself): add details route-map matching
	assert.NoError(t, err)
	assert.Equal(t, gatepb.TransferCode_SEE_OTHER, res.Status.Code)
	assert.Equal(t, "route error: see other node another-txn-host", res.Status.Message)
}

func TestGetLastSettings(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockSettingsProcessor(mock)

	g := NewGate(nil, proc)

	ctx := context.TODO()
	req := &gatepb.GetLastSettingsRequest{Account: 0}
	resp := &gatepb.GetLastSettingsResponse{
		Status:   &gatepb.Status{},
		Id:       1,
		Account:  3,
		Hash:     "0000000000000000000000000000000000000000000000000000000000000000",
		PrevHash: "0000000000000000000000000000000000000000000000000000000000000000",
		DataHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Sign:     "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	}

	proc.EXPECT().GetLastSettings(ctx, pt.AccID(req.Account)).Return(&pt.Settings{ID: 1, Account: 3}, nil)

	res, err := g.GetLastSettings(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	proc.EXPECT().GetLastSettings(ctx, pt.AccID(req.Account)).Return(nil, nil)

	res, err = g.GetLastSettings(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.GetLastSettingsResponse{Status: &gatepb.Status{}}, res)

	respErr := errors.New("some error")

	proc.EXPECT().GetLastSettings(ctx, pt.AccID(req.Account)).Return(nil, respErr)

	res, err = g.GetLastSettings(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.GetLastSettingsResponse{Status: &gatepb.Status{
		Code:    gatepb.TransferCode_INTERNAL_ERROR,
		Message: "gate: " + respErr.Error(),
	}}, res)

	// routing error
	r := mocks.NewMockRouter(mock)
	g.SetRouter(r)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("another-txn-host")
	r.EXPECT().IsSelf(gomock.Any()).Return(false)
	r.EXPECT().Nodes().Return([]string{"another-txn-host"})

	res, err = g.GetLastSettings(context.TODO(), &gatepb.GetLastSettingsRequest{Account: 10})
	// TODO(outself): add details route-map matching
	assert.NoError(t, err)
	assert.Equal(t, gatepb.TransferCode_SEE_OTHER, res.Status.Code)
	assert.Equal(t, "route error: see other node another-txn-host", res.Status.Message)
}

func TestTransferErrors(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockTransferProcessor(mock)

	g := NewGate(proc, nil)

	ctx := context.TODO()
	req := &gatepb.TransferRequest{Sender: 0, Batch: []*gatepb.TransferItem{{}}}
	resp := &gatepb.TransferResponse{Hash: "0000000000000000000000000000000000000000000000000000000000000000"}
	respErr := errors.New("some test error")

	// check ErrNoBalance
	proc.EXPECT().ProcessTransfer(ctx, gomock.Any()).Return(pt.TransferResult{}, processor.ErrNoBalance)

	resp.Status = &gatepb.Status{Code: gatepb.TransferCode_NO_BALANCE, Message: "gate: processor: no balance"}
	res, err := g.ProcessTransfer(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	// check ErrNegativeAmount
	proc.EXPECT().ProcessTransfer(ctx, gomock.Any()).Return(pt.TransferResult{}, processor.ErrNegativeAmount)

	resp.Status = &gatepb.Status{Code: gatepb.TransferCode_BAD_REQUEST, Message: "gate: processor: invalid receiver amount"}
	res, err = g.ProcessTransfer(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	// check ErrNoReceivers
	proc.EXPECT().ProcessTransfer(ctx, gomock.Any()).Return(pt.TransferResult{}, processor.ErrNoReceivers)

	resp.Status = &gatepb.Status{Code: gatepb.TransferCode_BAD_REQUEST, Message: "gate: processor: empty batch. no receivers"}
	res, err = g.ProcessTransfer(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	// check ErrInvalidPrevHash
	proc.EXPECT().ProcessTransfer(ctx, gomock.Any()).Return(pt.TransferResult{}, processor.ErrInvalidPrevHash)

	resp.Status = &gatepb.Status{Code: gatepb.TransferCode_INVALID_PREV_HASH, Message: "gate: processor: invalid transfer prev hash"}
	res, err = g.ProcessTransfer(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)

	// check default error case
	proc.EXPECT().ProcessTransfer(ctx, gomock.Any()).Return(pt.TransferResult{}, respErr)

	resp.Status = &gatepb.Status{Code: gatepb.TransferCode_INTERNAL_ERROR, Message: "gate: " + respErr.Error()}
	res, err = g.ProcessTransfer(context.TODO(), req)
	assert.NoError(t, err)
	assert.Equal(t, resp, res)
}

func TestRouting(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	proc := mocks.NewMockTransferProcessor(mock)
	r := mocks.NewMockRouter(mock)

	g := NewGate(proc, nil)
	g.SetRouter(r)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("another-host")
	r.EXPECT().IsSelf(gomock.Any()).Return(false)
	r.EXPECT().Nodes().Return([]string{"another-host"})

	res, err := g.GetPrevHash(context.TODO(), &gatepb.GetPrevHashRequest{Account: 10})
	// TODO(outself): add details route-map matching
	assert.NoError(t, err)
	assert.Equal(t, gatepb.TransferCode_SEE_OTHER, res.Status.Code)
	assert.Equal(t, "route error: see other node another-host", res.Status.Message)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("another-txn-host")
	r.EXPECT().IsSelf(gomock.Any()).Return(false)
	r.EXPECT().Nodes().Return([]string{"another-txn-host"})

	// TODO(outself): prev_hash without routing case

	res2, err := g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{Sender: 0, Batch: []*gatepb.TransferItem{{}}})
	// TODO(outself): add details route-map matching
	assert.NoError(t, err)
	assert.Equal(t, gatepb.TransferCode_SEE_OTHER, res2.Status.Code)
	assert.Equal(t, "route error: see other node another-txn-host", res2.Status.Message)

	r.EXPECT().GetHostByKey(gomock.Any()).Return("another-txn-host")
	r.EXPECT().IsSelf(gomock.Any()).Return(true)
	proc.EXPECT().ProcessTransfer(context.TODO(), gomock.Any()).Return(pt.TransferResult{}, nil)

	res2, err = g.ProcessTransfer(context.TODO(), &gatepb.TransferRequest{Sender: 0, Batch: []*gatepb.TransferItem{{}}})
	assert.NoError(t, err)
	assert.Equal(t, &gatepb.TransferResponse{
		Status: &gatepb.Status{},
		TxnId:  "0_0",
		Hash:   "0000000000000000000000000000000000000000000000000000000000000000",
	}, res2)
}
