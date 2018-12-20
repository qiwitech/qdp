package multipusher

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/qiwitech/qdp/pt"
)

// Multipusher pushes the same transactions to subsequent pushers in parallel
type Multipusher struct {
	list []pt.Pusher
}

func New(sub ...pt.Pusher) *Multipusher {
	return &Multipusher{list: sub}
}

func (p *Multipusher) Push(ctx context.Context, txns []pt.Txn) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, s := range p.list {
		s := s
		g.Go(func() error {
			return s.Push(ctx, txns)
		})
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "multipush failed")
	}

	return nil
}

// SettingsMultipusher pushes the same settings to subsequent pushers in parallel
type SettingsMultipusher struct {
	list []pt.SettingsPusher
}

func NewSettings(sub ...pt.SettingsPusher) *SettingsMultipusher {
	return &SettingsMultipusher{list: sub}
}

func (p *SettingsMultipusher) PushSettings(ctx context.Context, sett *pt.Settings) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, s := range p.list {
		s := s
		g.Go(func() error {
			return s.PushSettings(ctx, sett)
		})
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "multipush failed")
	}

	return nil
}
