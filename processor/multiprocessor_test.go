package processor

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/chain"
	"github.com/qiwitech/qdp/pt"
)

func TestMultiProcessTransferWithPusher(t *testing.T) {
	p := NewMultiprocessor(chain.NewChain(), 3)

	p.SetPusher(&fakeFailingPusher{})

	_, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(0, 20, 1000))
	assert.EqualError(t, errors.Cause(err), "fake internal error")
}

func TestMultiGetPrevHash(t *testing.T) {
	c := chain.NewChain()
	p := NewMultiprocessor(c, 3)

	h, err := p.GetPrevHash(context.TODO(), pt.AccID(10))
	assert.NoError(t, err)
	assert.Equal(t, pt.ZeroHash, h)
}

func TestMultiGetBalance(t *testing.T) {
	c := chain.NewChain()
	p := NewMultiprocessor(c, 3)

	b, err := p.GetBalance(context.TODO(), pt.AccID(10))
	assert.NoError(t, err)
	assert.Equal(t, int64(0), b)
}

func TestMultiCoverSetSettingsChain(t *testing.T) {
	c := chain.NewChain()
	p := NewMultiprocessor(c, 3)

	p.SetSettingsChain(nil)
}

func BenchmarkMultiProcessTransfer32Receivers(b *testing.B) {
	p := NewMultiprocessor(chain.NewChain(), 997)
	transfer := pt.NewSingleTransfer(0, 20, 1000)
	for i := 0; i < 32; i++ {
		transfer.AddReceiver(pt.AccID(i+1), int64(i))
	}

	prevhash := pt.Hash{}
	for i := 0; i < b.N; i++ {
		transfer.PrevHash = prevhash
		res, err := p.ProcessTransfer(context.TODO(), transfer)
		assert.NoError(b, err)
		prevhash = res.Hash
	}
	b.ReportAllocs()
}
