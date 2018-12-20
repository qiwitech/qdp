package processor

import (
	"context"

	"github.com/qiwitech/qdp/pt"
)

// SettingsMultiprocessor allows to use many settings processors in parallel safe.
type SettingsMultiprocessor struct {
	sub []pt.SettingsProcessor
}

// NewSettingsMultiprocessor creates n parallel SettingsProcessors on top of chain
func NewSettingsMultiprocessor(chain pt.SettingsChain, n int) *SettingsMultiprocessor {
	m := &SettingsMultiprocessor{
		sub: make([]pt.SettingsProcessor, n),
	}
	for i := range m.sub {
		m.sub[i] = NewSettingsProcessor(chain)
	}
	return m
}

func (p *SettingsMultiprocessor) ProcessSettings(ctx context.Context, s *pt.Settings) (pt.SettingsResult, error) {
	sub := p.sub[s.Account%pt.AccID(len(p.sub))]
	return sub.ProcessSettings(ctx, s)
}

func (p *SettingsMultiprocessor) GetLastSettings(ctx context.Context, acc pt.AccID) (*pt.Settings, error) {
	sub := p.sub[acc%pt.AccID(len(p.sub))]
	return sub.GetLastSettings(ctx, acc)
}
