package preloader

import (
	"context"
	"strconv"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/qiwitech/qdp/pt"
)

var (
	AccountsCached = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "plutos",
		Subsystem: "preloader",
		Name:      "accounts_cached",
		Help:      "number of accounts in memory at the moment",
	})
)

func init() {
	prometheus.MustRegister(AccountsCached)
}

var ErrLoading = errors.New("account is fetching from bigchain. retry again")

// Preloader preloads account last transactions when account is used for the first time of after reset
type Preloader struct {
	sync.Mutex
	// as a set
	preloaded       map[pt.AccID]struct{}
	preloadGroup    singleflight.Group
	bigchain        pt.BigChain
	chain           pt.Chain
	settingsChain   pt.SettingsChain
	MaxTransactions int
}

func New(chain pt.Chain, settingsChain pt.SettingsChain, bigchain pt.BigChain) *Preloader {
	return &Preloader{
		bigchain:        bigchain,
		chain:           chain,
		preloaded:       make(map[pt.AccID]struct{}),
		settingsChain:   settingsChain,
		MaxTransactions: 5,
	}
}

func (p *Preloader) load(ctx context.Context, accID pt.AccID) (err error) {
	txns, settings, err := p.bigchain.Fetch(ctx, accID, p.MaxTransactions)
	if err != nil {
		// TODO(outself): log warning
		return errors.Wrap(err, "chain preloader")
	}

	if len(txns) != 0 {
		p.chain.PutTo(accID, txns)
	}

	if settings != nil {
		p.settingsChain.Put(settings)
	}

	p.Lock()
	p.preloaded[accID] = struct{}{}

	l := len(p.preloaded)
	p.Unlock()

	AccountsCached.Set(float64(l))

	return nil
}

var testCallback chan error

func (p *Preloader) Preload(ctx context.Context, accID pt.AccID) error {
	p.Lock()
	_, exist := p.preloaded[accID]
	p.Unlock()

	if exist {
		return nil
	}

	d, ok := ctx.Deadline()

	_, err, _ := p.preloadGroup.Do(strconv.FormatUint(uint64(accID), 10), func() (interface{}, error) {
		// TODO(outself): inherit parent ctx with essential values, but without timeout/cancel
		ctx := context.Background()
		var cancel func()
		if ok {
			ctx, cancel = context.WithDeadline(ctx, d)
			defer cancel()
		}
		err := p.load(ctx, accID)
		if testCallback != nil {
			testCallback <- err
		}
		return nil, err
	})
	return err

	//	return ErrLoading
}

func (p *Preloader) Reset(ctx context.Context, accID pt.AccID) {
	p.Lock()
	delete(p.preloaded, accID)
	p.Unlock()

	p.chain.Reset(accID)
	p.settingsChain.Reset(accID)
}
