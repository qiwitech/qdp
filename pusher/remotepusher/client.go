package remotepusher

import (
	"context"

	"github.com/pkg/errors"
	"github.com/qiwitech/tcprpc"

	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
)

func txnsToProto(in []pt.Txn) []*chainpb.Txn {
	txns := make([]*chainpb.Txn, len(in))
	for i := range in {
		t := in[i]
		txns[i] = &chainpb.Txn{
			ID:         uint64(t.ID),
			Sender:     uint64(t.Sender),
			Receiver:   uint64(t.Receiver),
			Amount:     t.Amount,
			Balance:    t.Balance,
			SpentBy:    uint64(t.SpentBy),
			SettingsId: uint64(t.SettingsID),
			PrevHash:   t.PrevHash[:],
		}
		if t.Hash != pt.ZeroHash {
			txns[i].Hash = t.Hash[:]
		}
		if t.Sign != pt.ZeroSign {
			txns[i].Sign = t.Sign[:]
		}
	}
	return txns
}

func settToProto(in *pt.Settings) *chainpb.Settings {
	sett := &chainpb.Settings{
		ID:        uint64(in.ID),
		Account:   uint64(in.Account),
		Hash:      in.Hash[:],
		PrevHash:  in.PrevHash[:],
		PublicKey: in.PublicKey[:],
		Sign:      in.Sign[:],
		DataHash:  in.DataHash[:],
	}
	return sett
}

type PusherClient struct {
	txns     pusherpb.PusherServiceInterface
	settings pusherpb.SettingsPusherServiceInterface
}

func NewClient(txns pusherpb.PusherServiceInterface, settings pusherpb.SettingsPusherServiceInterface) *PusherClient {
	return &PusherClient{txns: txns, settings: settings}
}

func NewHTTPClient(baseurl string) pt.Pusher {
	g := tcprpc.NewClient(baseurl)
	txns := pusherpb.NewTCPRPCPusherServiceClient(g, "v1/")
	return NewClient(txns, nil)
}

func NewDBPusher(baseurl string) *PusherClient {
	g := tcprpc.NewClient(baseurl)
	txns := pusherpb.NewTCPRPCPusherServiceClient(g, "v1/")
	settings := pusherpb.NewTCPRPCSettingsPusherServiceClient(g, "v1/")
	return NewClient(txns, settings)
}

func (c *PusherClient) Push(ctx context.Context, txns []pt.Txn) error {
	if len(txns) == 0 {
		return nil
	}

	txnsProto := txnsToProto(txns)

	resp, err := c.txns.Push(ctx, &pusherpb.PushRequest{Txns: txnsProto})
	if err != nil {
		return errors.Wrap(err, "remote push failed")
	}

	if resp == nil {
		return errors.New("remote push failed: empty response")
	}

	switch pusherpb.PushCode(resp.Status.Code) {
	case pusherpb.PushCode_INTERNAL_ERROR:
		return errors.Errorf("remote push failed: %+v", resp.Status.Message)

	}

	return nil
}

func (c *PusherClient) PushSettings(ctx context.Context, sett *pt.Settings) error {
	settProto := settToProto(sett)

	resp, err := c.settings.PushSettings(ctx, &pusherpb.PushSettingsRequest{Settings: []*chainpb.Settings{settProto}})
	if err != nil {
		return errors.Wrap(err, "remote push failed")
	}

	if resp == nil {
		return errors.New("remote push failed: empty response")
	}

	switch pusherpb.PushCode(resp.Status.Code) {
	case pusherpb.PushCode_INTERNAL_ERROR:
		return errors.Errorf("remote push failed: %+v", resp.Status.Message)

	}

	return nil
}
