package processor

import (
	"context"
	"crypto/sha256"
	"hash"
	"sync"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/pt"
)

// SettingsProcessor is an accounts settings processor.
type SettingsProcessor struct {
	mu        sync.Mutex
	hash      hash.Hash
	chain     pt.SettingsChain
	pusher    pt.SettingsPusher
	preloader pt.Preloader
}

func NewSettingsProcessor(chain pt.SettingsChain) *SettingsProcessor {
	return &SettingsProcessor{
		hash:  sha256.New(),
		chain: chain,
	}
}

func (p *SettingsProcessor) SetPreloader(pl pt.Preloader) {
	p.preloader = pl
}

func (p *SettingsProcessor) SetPusher(pusher pt.SettingsPusher) {
	p.pusher = pusher
}

func (p *SettingsProcessor) ProcessSettings(ctx context.Context, s *pt.Settings) (pt.SettingsResult, error) {
	var res pt.SettingsResult

	defer p.mu.Unlock()
	p.mu.Lock()

	if err := p.preloadAccount(ctx, s.Account); err != nil {
		return res, errors.Wrap(err, "account preloading")
	}

	// fetch last settings
	last := p.chain.GetLastSettings(s.Account)
	lastHash := pt.Hash{}
	if last != nil {
		lastHash = last.Hash
		if last.PublicKey != nil {
			key, err := last.PublicKey.BTCECKey()
			if err != nil {
				return res, err
			}
			hash := pt.GetSettingsRequestHashDefault(s)
			err = pt.VerifyTransferHash(s.Sign, hash, key)
			if err != nil {
				return res, ErrInvalidSign
			}
		} else if s.Sign != pt.ZeroSign {
			return res, ErrInvalidSign
		}
	}

	// check prev settings hash
	if lastHash != s.PrevHash {
		return res, ErrInvalidSettingsPrevHash
	}

	// increment last id
	var id pt.ID
	if last != nil {
		id = last.ID
	}

	s.ID = id + 1

	// TODO(outself): rename
	pt.GetSettingsHash(p.hash, s)

	// push settings to another processors/db/external services
	if p.pusher != nil {
		if err := p.pusher.PushSettings(ctx, s); err != nil {
			return res, errors.Wrap(err, "push")
		}
	}

	// commit to chain
	p.chain.Put(s)

	res.SettingsID = pt.NewSettingsID(s.Account, s.ID)
	res.Hash = s.Hash
	return res, nil
}

func (p *SettingsProcessor) GetLastSettings(ctx context.Context, acc pt.AccID) (*pt.Settings, error) {
	defer p.mu.Unlock()
	p.mu.Lock()

	if err := p.preloadAccount(ctx, acc); err != nil {
		return nil, errors.Wrap(err, "account preloading")
	}

	return p.chain.GetLastSettings(acc), nil
}

func (p *SettingsProcessor) preloadAccount(ctx context.Context, acc pt.AccID) error {
	if p.preloader == nil {
		// now we could work without bigchain
		return nil
	}

	return p.preloader.Preload(ctx, acc)
}
