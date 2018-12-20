package processor

import (
	"context"
	"crypto/sha256"
	"hash"
	"sync"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/pt"
)

var (
	ErrInvalidPrevHash   = errors.New("processor: invalid transfer prev hash")
	ErrNoReceivers       = errors.New("processor: empty batch. no receivers")
	ErrNegativeAmount    = errors.New("processor: invalid receiver amount")
	ErrNoBalance         = errors.New("processor: no balance")
	ErrInvalidSettingsID = errors.New("processor: invalid settings id")
	ErrInvalidSign       = errors.New("processor: invalid sign")

	ErrInvalidSettingsPrevHash = errors.New("settings processor: invalid prev hash")
)

// Processor is an transaction processor.
// It checks if request is correct, creates new transactions, pushes them to Pusher.
type Processor struct {
	mu            sync.Mutex
	hash          hash.Hash
	chain         pt.Chain
	settingsChain pt.SettingsChain
	pusher        pt.Pusher
	preloader     pt.Preloader
}

func NewProcessor(chain pt.Chain) *Processor {
	return &Processor{
		hash:  sha256.New(),
		chain: chain,
	}
}

func (p *Processor) SetSettingsChain(c pt.SettingsChain) {
	p.settingsChain = c
}

func (p *Processor) SetPreloader(pl pt.Preloader) {
	p.preloader = pl
}

func (p *Processor) SetPusher(pusher pt.Pusher) {
	p.pusher = pusher
}

func (p *Processor) GetPrevHash(ctx context.Context, acc pt.AccID) (pt.Hash, error) {
	defer p.mu.Unlock()
	p.mu.Lock()

	if err := p.preloadAccount(ctx, acc); err != nil {
		return pt.ZeroHash, errors.Wrap(err, "account preloading")
	}

	return p.chain.GetLastHash(acc), nil
}

func (p *Processor) GetBalance(ctx context.Context, acc pt.AccID) (int64, error) {
	defer p.mu.Unlock()
	p.mu.Lock()

	if err := p.preloadAccount(ctx, acc); err != nil {
		return 0, errors.Wrap(err, "account preloading")
	}

	return p.chain.GetBalance(acc), nil
}

func (p *Processor) ProcessTransfer(ctx context.Context, t pt.Transfer) (pt.TransferResult, error) {
	var res pt.TransferResult

	// check receivers size
	if len(t.Batch) == 0 {
		return res, ErrNoReceivers
	}

	defer p.mu.Unlock()
	p.mu.Lock()

	if err := p.preloadAccount(ctx, t.Sender); err != nil {
		return res, errors.Wrap(err, "account preloading")
	}

	// fetch last txn
	last := p.chain.GetLastTxn(t.Sender)
	lastHash := pt.Hash{}
	if last != nil {
		if last.Hash != pt.ZeroHash {
			pt.GetHashDefault(last)
		}

		lastHash = last.Hash

		res.Hash = last.Hash
	}

	if last != nil { // idempotence check
		if len(t.Batch) == 1 {
			if t.PrevHash == last.PrevHash &&
				t.SettingsID == last.SettingsID && t.Batch[0].Receiver == last.Receiver && t.Batch[0].Amount == last.Amount {
				// TODO(nik): check other fields
				res.TxnID = pt.NewTxnID(last.Sender, last.ID)
				res.Hash = last.Hash
				return res, nil
			}
		} else { // batch case
			prev := p.chain.GetLastNTxns(t.Sender, len(t.Batch))
			if len(prev) == len(t.Batch) {
				l := len(prev)
				f := prev[l-1]
				first := t.PrevHash == f.PrevHash && t.SettingsID == f.SettingsID
				rest := true
				l--
				for i := range prev {
					if prev[i].Receiver != t.Batch[l-i].Receiver || prev[i].Amount != t.Batch[l-i].Amount {
						rest = false
						break
					}
				}
				if first && rest {
					res.TxnID = pt.NewTxnID(f.Sender, f.ID)
					res.Hash = f.Hash
					return res, nil
				}
			}
		}
	}

	if p.settingsChain != nil {
		sett := p.settingsChain.GetLastSettings(t.Sender)
		if sett != nil {
			res.SettingsId = sett.ID

			if t.SettingsID != sett.ID {
				return res, ErrInvalidSettingsID
			}
			if sett.PublicKey != nil {
				key, err := sett.PublicKey.BTCECKey()
				if err != nil {
					return res, err
				}
				hash := pt.GetTransferHashDefault(t)
				err = pt.VerifyTransferHash(t.Sign, hash, key)
				if err != nil {
					return res, ErrInvalidSign
				}
			} else if t.Sign != pt.ZeroSign {
				return res, ErrInvalidSign
			}
		} else if t.Sign != pt.ZeroSign {
			return res, ErrInvalidSign
		}
	}

	// check prev txn hash
	if lastHash != t.PrevHash {
		return res, ErrInvalidPrevHash
	}

	// fetch balance
	balance := p.chain.GetBalance(t.Sender)

	// TODO(outself): write inputs hash
	// batch alloc objects, memory optimization routine
	txns := make([]pt.Txn, len(t.Batch))
	for i, r := range t.Batch {
		//	if r.Amount < 0 {
		//		return res, ErrNegativeAmount
		//	}

		balance -= r.Amount

		// zero account can have negative balance
		if t.Sender != 0 {
			if balance < 0 {
				return res, ErrNoBalance
			}
		}

		txns[i].Sender = t.Sender
		txns[i].Receiver = r.Receiver
		txns[i].Amount = r.Amount
		txns[i].Balance = balance
	}

	// TODO(outself): check txns
	//if err := p.chain.CheckTxns(txns); err != nil {
	//	return nil, errors.Wrap(err, "process transfer failed")
	//}

	var id pt.ID
	if last != nil {
		id = last.ID
	}

	// link output transaction with used input transactions
	inputsTxns := p.chain.ListUnspentTxns(t.Sender)
	for i := range inputsTxns {
		inputsTxns[i].SpentBy = id + 1
	}

	// calc hashes and assign txn ids
	for i := range txns {
		id++

		if i == 0 {
			txns[i].PrevHash = t.PrevHash
		} else {
			txns[i].PrevHash = txns[i-1].Hash
		}

		txns[i].ID = id
		txns[i].Hash = pt.GetHash(p.hash, &txns[i])
	}
	txns[0].Sign = t.Sign

	// return last txn hash and combined external txn_id as sender_id+txn_num
	res.Hash = txns[len(txns)-1].Hash
	// TODO(outself): add test for id equal
	res.TxnID = pt.NewTxnID(txns[0].Sender, txns[0].ID)

	// merge new txns and changed inputs (with SpentBy == first current output txn id)
	txns = append(txns, inputsTxns...)

	// push txns to another processors/db/external services
	if p.pusher != nil {
		if err := p.pusher.Push(ctx, txns); err != nil {
			// TODO(nik)[Done. Verification could be needed]: We need to reload state for account since we don't know write status for sure.
			if p.preloader != nil {
				p.preloader.Reset(ctx, t.Sender)
			}
			return res, errors.Wrap(err, "push")
		}
	}

	// commit to chain
	p.chain.PutTo(t.Sender, txns)
	return res, nil
}

func (p *Processor) preloadAccount(ctx context.Context, acc pt.AccID) error {
	if p.preloader == nil {
		// we can work without preloader
		return nil
	}

	return p.preloader.Preload(ctx, acc)
}
