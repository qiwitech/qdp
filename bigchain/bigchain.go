// Package bigchain is an reliable storage interface implementation for plutos to use
package bigchain

import (
	"context"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
	"github.com/qiwitech/qdp/pt"
)

type BigChain struct {
	cl plutodbpb.PlutoDBServiceInterface
}

func New(cl plutodbpb.PlutoDBServiceInterface) *BigChain {
	return &BigChain{cl: cl}
}

func (b *BigChain) Fetch(ctx context.Context, acc pt.AccID, limit int) ([]pt.Txn, *pt.Settings, error) {
	res, err := b.cl.Fetch(ctx, &plutodbpb.FetchRequest{
		Account: uint64(acc),
		Limit:   uint32(limit),
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetch client")
	}

	txns := pbTxnsToPt(res.Txns)
	sett := pbSettingsToPt(res.Settings)

	return txns, sett, nil
}

func pbTxnsToPt(v []*chainpb.Txn) []pt.Txn {
	if v == nil {
		return nil
	}

	res := make([]pt.Txn, len(v))
	for i, t := range v {
		res[i] = pt.Txn{
			ID:       pt.ID(t.ID),
			Sender:   pt.AccID(t.Sender),
			Receiver: pt.AccID(t.Receiver),
			Amount:   t.Amount,
			Balance:  t.Balance,
			SpentBy:  pt.ID(t.SpentBy),
		}
		copy(res[i].Hash[:], t.Hash)
		copy(res[i].PrevHash[:], t.PrevHash)
	}
	return res
}

func pbSettingsToPt(v *chainpb.Settings) *pt.Settings {
	if v == nil {
		return nil
	}

	r := &pt.Settings{
		ID:                 pt.ID(v.ID),
		Account:            pt.AccID(v.Account),
		VerifyTransferSign: v.VerifyTransferSign,
		PublicKey:          dup(v.PublicKey),
	}
	copy(r.Hash[:], v.Hash)
	copy(r.PrevHash[:], v.PrevHash)
	copy(r.DataHash[:], v.DataHash)
	copy(r.Sign[:], v.Sign)
	return r
}

// dup duplicates memory
func dup(v []byte) []byte {
	if v == nil {
		return nil
	}
	r := make([]byte, len(v))
	copy(r, v)
	return r
}
