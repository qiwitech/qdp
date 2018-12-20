package remotepusher

import (
	"context"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
)

func NewService(pusher pt.Pusher) *Service {
	return &Service{pusher}
}

type Service struct {
	pusher pt.Pusher
}

func txnsFromProto(in []*chainpb.Txn) ([]pt.Txn, error) {
	txns := make([]pt.Txn, len(in))
	for i, t := range in {
		txns[i].ID = pt.ID(t.ID)
		txns[i].Sender = pt.AccID(t.Sender)
		txns[i].Receiver = pt.AccID(t.Receiver)
		txns[i].Amount = t.Amount
		txns[i].Balance = t.Balance
		txns[i].SpentBy = pt.ID(t.SpentBy)

		if len(t.PrevHash) != 0 && len(t.PrevHash) != len(pt.ZeroHash) {
			return nil, errors.Errorf("invalid prev_hash size %d for txn_id=%d, sender_id=%d", len(t.PrevHash), t.ID, t.Sender)
		}

		copy(txns[i].PrevHash[:], t.PrevHash)

		if len(t.Hash) != 0 && len(t.Hash) != len(pt.ZeroHash) {
			return nil, errors.Errorf("invalid hash size %d for txn_id=%d, sender_id=%d", len(t.Hash), t.ID, t.Sender)
		}

		copy(txns[i].Hash[:], t.Hash)

		if len(t.Sign) != 0 && len(t.Sign) != len(pt.ZeroSign) {
			return nil, errors.Errorf("invalid sign size %d for txn_id=%d, sender_id=%d", len(t.Sign), t.ID, t.Sender)
		}

		copy(txns[i].Sign[:], t.Sign)
	}
	return txns, nil
}

func (s *Service) Push(ctx context.Context, req *pusherpb.PushRequest) (*pusherpb.PushResponse, error) {
	res := &pusherpb.PushResponse{
		Status: &pusherpb.Status{Code: int32(pusherpb.PushCode_OK)},
	}

	txns, err := txnsFromProto(req.Txns)
	if err != nil {
		res.Status.Message = errors.Wrap(err, "validator").Error()
		res.Status.Code = int32(pusherpb.PushCode_INTERNAL_ERROR)
		return res, nil
	}

	if err := s.pusher.Push(ctx, txns); err != nil {
		res.Status.Message = errors.Wrap(err, "pusher").Error()
		res.Status.Code = int32(pusherpb.PushCode_INTERNAL_ERROR)
		return res, nil
	}

	return res, nil
}
