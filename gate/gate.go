// Package gate is an plutos node entry point for user requests
package gate

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/preloader"
	"github.com/qiwitech/qdp/processor"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/pt"
)

// Gate is an plutos node entry point for user requests
type Gate struct {
	processor         pt.TransferProcessor
	settingsProcessor pt.SettingsProcessor
	router            pt.Router
}

func NewGate(processor pt.TransferProcessor, settingsProcessor pt.SettingsProcessor) *Gate {
	return &Gate{processor: processor, settingsProcessor: settingsProcessor}
}

func (g *Gate) SetRouter(router pt.Router) {
	g.router = router
}

func transferFromProto(req *gatepb.TransferRequest) (*pt.Transfer, error) {
	if len(req.Batch) == 0 {
		return nil, errors.New("validator: empty batch, no receivers")
	}

	t := &pt.Transfer{
		Sender:     pt.AccID(req.Sender),
		Batch:      make([]*pt.TransferItem, len(req.Batch)),
		SettingsID: pt.ID(req.SettingsId),
	}

	if err := validateHexLen(req.PrevHash, len(pt.ZeroHash), "prev_hash"); err != nil {
		return nil, err
	}

	if err := validateHexLen(req.Sign, len(pt.ZeroSign), "sign"); err != nil {
		return nil, err
	}

	// TODO(outself): add too short validators

	var err error
	if t.PrevHash, err = pt.GetHashFromString(req.PrevHash); err != nil {
		return nil, errors.Wrap(err, "validator")
	}

	if t.Sign, err = pt.GetSignFromString(req.Sign); err != nil {
		return nil, errors.Wrap(err, "validator")
	}

	for i, r := range req.Batch {
		t.Batch[i] = &pt.TransferItem{Receiver: pt.AccID(r.Receiver), Amount: r.Amount}
	}

	return t, nil
}

func validateHexLen(s string, defaultLen int, fieldName string) error {
	defaultLen *= 2
	if len(s) > defaultLen {
		return errors.Errorf("validator: %s is too long (%d>%d)", fieldName, len(s), defaultLen)
	}

	if len(s) > 0 && len(s) < defaultLen {
		return errors.Errorf("validator: %s is too short (%d<%d)", fieldName, len(s), defaultLen)
	}
	return nil
}

func settingsFromProto(req *gatepb.SettingsRequest) (*pt.Settings, error) {
	s := &pt.Settings{
		Account: pt.AccID(req.Account),
		//PublicKey:          pt.PublicKey(req.PublicKey),
		VerifyTransferSign: req.VerifyTransferSign,
	}

	if err := validateHexLen(req.PrevHash, len(pt.ZeroHash), "prev_hash"); err != nil {
		return nil, err
	}

	if err := validateHexLen(req.Sign, len(pt.ZeroSign), "sign"); err != nil {
		return nil, err
	}

	var err error
	if s.PrevHash, err = pt.GetHashFromString(req.PrevHash); err != nil {
		return nil, errors.Wrap(err, "validator")
	}

	if s.Sign, err = pt.GetSignFromString(req.Sign); err != nil {
		return nil, errors.Wrap(err, "validator")
	}

	if s.PublicKey, err = pt.ParsePubKey(req.PublicKey); err != nil {
		return nil, errors.Wrap(err, "validator")
	}

	return s, nil
}

// ProcessTransfer checks if it is responsible for Sender account and if so processes requests
func (g *Gate) ProcessTransfer(ctx context.Context, req *gatepb.TransferRequest) (*gatepb.TransferResponse, error) {
	res := &gatepb.TransferResponse{
		Status: &gatepb.Status{Code: gatepb.TransferCode_OK},
	}

	if !g.checkRouting(res.Status, req.Sender) {
		return res, nil
	}

	t, err := transferFromProto(req)
	if err != nil {
		res.Status.Code = gatepb.TransferCode_BAD_REQUEST
		res.Status.Message = errors.Wrap(err, "gate").Error()
		return res, nil
	}

	tres, err := g.processor.ProcessTransfer(ctx, *t)

	// helper fields. send them any way
	res.Hash = tres.Hash.String()
	res.SettingsId = uint64(tres.SettingsId)

	if err != nil {
		res.Status.Message = errors.Wrap(err, "gate").Error()
		cause := errors.Cause(err)

		switch cause {
		case processor.ErrNoBalance:
			res.Status.Code = gatepb.TransferCode_NO_BALANCE

		case processor.ErrNegativeAmount, processor.ErrNoReceivers:
			res.Status.Code = gatepb.TransferCode_BAD_REQUEST

		case processor.ErrInvalidPrevHash:
			res.Status.Code = gatepb.TransferCode_INVALID_PREV_HASH

		case preloader.ErrLoading:
			res.Status.Code = gatepb.TransferCode_RETRY

		default:
			res.Status.Code = gatepb.TransferCode_INTERNAL_ERROR
		}

		return res, nil
	}

	res.TxnId = tres.TxnID.String()
	res.Account = uint64(tres.TxnID.AccID)
	res.Id = uint64(tres.TxnID.ID)

	return res, nil
}

// ProcessTransfer checks if it is responsible for Account and if so processes requests
func (g *Gate) UpdateSettings(ctx context.Context, req *gatepb.SettingsRequest) (*gatepb.SettingsResponse, error) {
	res := &gatepb.SettingsResponse{
		Status: &gatepb.Status{Code: gatepb.TransferCode_OK},
	}

	if !g.checkRouting(res.Status, req.Account) {
		return res, nil
	}

	s, err := settingsFromProto(req)
	if err != nil {
		res.Status.Code = gatepb.TransferCode_BAD_REQUEST
		res.Status.Message = errors.Wrap(err, "gate").Error()
		return res, nil
	}

	sres, err := g.settingsProcessor.ProcessSettings(ctx, s)
	if err != nil {
		res.Status.Message = errors.Wrap(err, "gate").Error()
		cause := errors.Cause(err)

		switch cause {
		case processor.ErrInvalidSettingsPrevHash:
			res.Status.Code = gatepb.TransferCode_INVALID_PREV_HASH

		case preloader.ErrLoading:
			res.Status.Code = gatepb.TransferCode_RETRY

		default:
			res.Status.Code = gatepb.TransferCode_INTERNAL_ERROR
		}

		return res, nil
	}

	res.SettingsId = sres.SettingsID.String()
	res.Hash = sres.Hash.String()

	return res, nil
}

func (g *Gate) GetPrevHash(ctx context.Context, req *gatepb.GetPrevHashRequest) (*gatepb.GetPrevHashResponse, error) {
	res := &gatepb.GetPrevHashResponse{
		Status: &gatepb.Status{Code: gatepb.TransferCode_OK},
	}

	// TODO: draft
	if !g.checkRouting(res.Status, req.Account) {
		return res, nil
	}

	h, err := g.processor.GetPrevHash(ctx, pt.AccID(req.Account))
	if err != nil {
		res.Status.Message = errors.Wrap(err, "gate").Error()
		cause := errors.Cause(err)
		switch cause {
		case preloader.ErrLoading:
			res.Status.Code = gatepb.TransferCode_RETRY
		default:
			res.Status.Code = gatepb.TransferCode_INTERNAL_ERROR
		}
		return res, nil
	}

	res.Hash = h.String()
	return res, nil
}

func (g *Gate) GetBalance(ctx context.Context, req *gatepb.GetBalanceRequest) (*gatepb.GetBalanceResponse, error) {
	res := &gatepb.GetBalanceResponse{
		Status: &gatepb.Status{Code: gatepb.TransferCode_OK},
	}

	if !g.checkRouting(res.Status, req.Account) {
		return res, nil
	}

	b, err := g.processor.GetBalance(ctx, pt.AccID(req.Account))
	if err != nil {
		res.Status.Message = errors.Wrap(err, "gate").Error()
		cause := errors.Cause(err)
		switch cause {
		case preloader.ErrLoading:
			res.Status.Code = gatepb.TransferCode_RETRY
		default:
			res.Status.Code = gatepb.TransferCode_INTERNAL_ERROR
		}
		return res, nil
	}

	res.Balance = b
	return res, nil
}

func (g *Gate) GetLastSettings(ctx context.Context, req *gatepb.GetLastSettingsRequest) (*gatepb.GetLastSettingsResponse, error) {
	res := &gatepb.GetLastSettingsResponse{
		Status: &gatepb.Status{Code: gatepb.TransferCode_OK},
	}

	// TODO: draft
	if !g.checkRouting(res.Status, req.Account) {
		return res, nil
	}

	s, err := g.settingsProcessor.GetLastSettings(ctx, pt.AccID(req.Account))
	if err != nil {
		res.Status.Message = errors.Wrap(err, "gate").Error()
		cause := errors.Cause(err)
		switch cause {
		case preloader.ErrLoading:
			res.Status.Code = gatepb.TransferCode_RETRY
		default:
			res.Status.Code = gatepb.TransferCode_INTERNAL_ERROR
		}
		return res, nil
	}
	if s == nil {
		return res, nil
	}

	res.Id = uint64(s.ID)
	res.Hash = s.Hash.String()
	res.Account = uint64(s.Account)
	res.PrevHash = s.PrevHash.String()
	res.DataHash = s.DataHash.String()
	res.VerifyTransferSign = s.VerifyTransferSign
	res.Sign = s.Sign.String()
	res.PublicKey = s.PublicKey.String()

	return res, nil
}

func (g *Gate) checkRouting(st *gatepb.Status, acc uint64) bool {
	if g.router == nil {
		return true
	}
	node := g.router.GetHostByKey(fmt.Sprintf("%d", pt.AccID(acc)))
	if !g.router.IsSelf(node) {
		st.Code = gatepb.TransferCode_SEE_OTHER
		st.Message = errors.Errorf("route error: see other node %s", node).Error()
		rt := &gatepb.RouteMap{Nodes: g.router.Nodes(), Target: node}
		m, _ := proto.Marshal(rt)
		st.Details = []*any.Any{{Value: m, TypeUrl: proto.MessageName(rt)}}
		return false
	}
	return true
}
