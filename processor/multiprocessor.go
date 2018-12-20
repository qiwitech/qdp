package processor

import (
	"context"

	"github.com/qiwitech/qdp/pt"
)

// Multiprocessor allows to use many processors in parallel safe.
type Multiprocessor struct {
	sub []pt.TransferProcessor
}

// NewMultiprocessor creates n parallel Processors on top of chain
func NewMultiprocessor(chain pt.Chain, n int) *Multiprocessor {
	m := &Multiprocessor{
		sub: make([]pt.TransferProcessor, n),
	}
	for i := range m.sub {
		m.sub[i] = NewProcessor(chain)
	}
	return m
}

func (p *Multiprocessor) SetPusher(pusher pt.Pusher) {
	for _, s := range p.sub {
		s.SetPusher(pusher)
	}
}

func (p *Multiprocessor) SetPreloader(preloader pt.Preloader) {
	for _, s := range p.sub {
		s.SetPreloader(preloader)
	}
}

func (p *Multiprocessor) SetSettingsChain(sc pt.SettingsChain) {
	for _, s := range p.sub {
		s.SetSettingsChain(sc)
	}
}

func (p *Multiprocessor) ProcessTransfer(ctx context.Context, t pt.Transfer) (pt.TransferResult, error) {
	sub := p.sub[t.Sender%pt.AccID(len(p.sub))]
	return sub.ProcessTransfer(ctx, t)
}

func (p *Multiprocessor) GetPrevHash(ctx context.Context, acc pt.AccID) (pt.Hash, error) {
	sub := p.sub[acc%pt.AccID(len(p.sub))]
	return sub.GetPrevHash(ctx, acc)
}

func (p *Multiprocessor) GetBalance(ctx context.Context, acc pt.AccID) (int64, error) {
	sub := p.sub[acc%pt.AccID(len(p.sub))]
	return sub.GetBalance(ctx, acc)
}
