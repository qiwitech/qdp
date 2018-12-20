package pusher

import (
	"context"

	"github.com/qiwitech/qdp/pt"
)

func NewChainReceiversPusher(chain pt.Chain) *ChainPusher {
	return &ChainPusher{chain}
}

// ChainPusher saves transactions to receivers chains
type ChainPusher struct {
	chain pt.Chain
}

func (p *ChainPusher) Push(ctx context.Context, txns []pt.Txn) error {
	for i, txn := range txns {
		p.chain.PutTo(txn.Receiver, txns[i:i+1])
	}
	return nil
}
