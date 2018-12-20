package remotepusher

import (
	"context"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
)

type SettingsService struct {
	pusher pt.SettingsPusher
}

func NewSettingsService(pusher pt.SettingsPusher) *SettingsService {
	return &SettingsService{pusher}
}

func settFromProto(in []*chainpb.Settings) ([]pt.Settings, error) {
	sett := make([]pt.Settings, len(in))
	for i, s := range in {
		sett[i] = pt.Settings{
			ID:        pt.ID(s.ID),
			Account:   pt.AccID(s.Account),
			PublicKey: make([]byte, len(s.PublicKey)),
		}
		copy(sett[i].Hash[:], s.Hash)
		copy(sett[i].PrevHash[:], s.PrevHash)
		copy(sett[i].Sign[:], s.Sign)
		copy(sett[i].DataHash[:], s.DataHash)
		copy(sett[i].PublicKey[:], s.PublicKey)
	}
	return sett, nil
}

func (s *SettingsService) PushSettings(ctx context.Context, req *pusherpb.PushSettingsRequest) (*pusherpb.PushSettingsResponse, error) {
	res := &pusherpb.PushSettingsResponse{
		Status: &pusherpb.Status{Code: int32(pusherpb.PushCode_OK)},
	}

	sett, err := settFromProto(req.Settings)
	if err != nil {
		res.Status.Message = errors.Wrap(err, "validator").Error()
		res.Status.Code = int32(pusherpb.PushCode_INTERNAL_ERROR)
		return res, nil
	}

	for i := range sett {
		if err := s.pusher.PushSettings(ctx, &sett[i]); err != nil {
			res.Status.Message = errors.Wrap(err, "pusher").Error()
			res.Status.Code = int32(pusherpb.PushCode_INTERNAL_ERROR)
			return res, nil
		}
	}

	return res, nil
}
