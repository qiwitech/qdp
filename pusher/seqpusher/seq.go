package seqpusher

import (
	"context"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/pt"
)

// Seqpusher pushes the same transactions to subsequent pushers sequentially.
// If some push in the chain failed next are not executed
type Seqpusher struct {
	list []pt.Pusher
}

// New creates new Seqpusher with chain of sub pushers
func New(sub ...pt.Pusher) Seqpusher {
	return Seqpusher{list: sub}
}

// Push pushes the same transactions to subsequent pushers sequentially.
// If some push in the chain failed next are not executed
func (p Seqpusher) Push(ctx context.Context, txns []pt.Txn) error {
	for _, s := range p.list {
		if err := s.Push(ctx, txns); err != nil {
			return errors.Wrap(err, "seqpusher")
		}
	}

	return nil
}

// Seqpusher pushes the same settings to subsequent pushers sequentially.
// If some push in the chain failed next are not executed
type SettingsSeqpusher struct {
	list []pt.SettingsPusher
}

// NewSettings creates new SettingsSeqpusher with chain of sub pushers
func NewSettings(sub ...pt.SettingsPusher) SettingsSeqpusher {
	return SettingsSeqpusher{list: sub}
}

// PushSettings pushes the same settings to subsequent pushers sequentially.
// If some push in the chain failed next are not executed
func (p SettingsSeqpusher) PushSettings(ctx context.Context, sett *pt.Settings) error {
	for _, s := range p.list {
		if err := s.PushSettings(ctx, sett); err != nil {
			return errors.Wrap(err, "seqpusher")
		}
	}

	return nil
}
