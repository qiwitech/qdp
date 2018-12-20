package processor

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/chain"
	"github.com/qiwitech/qdp/mocks"
	"github.com/qiwitech/qdp/pt"
	"github.com/qiwitech/qdp/pusher"
)

func TestInvalidPrevHash(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	transfer := pt.NewSingleTransfer(0, 20, 1000)
	transfer.PrevHash = pt.HashFromString("123456")

	_, err := p.ProcessTransfer(context.TODO(), transfer)
	assert.Equal(t, ErrInvalidPrevHash, err)
}

func TestEmptyPrevHashInFirstTxn(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	transfer := pt.NewSingleTransfer(0, 20, 1000)
	transfer.PrevHash = pt.HashFromString("")

	_, err := p.ProcessTransfer(context.TODO(), transfer)
	assert.NoError(t, err)
}

func TestEmptyBatchNoReceivers(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	_, err := p.ProcessTransfer(context.TODO(), pt.Transfer{})
	assert.Equal(t, ErrNoReceivers, err)
}

func TestTransferNoBalance(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	_, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(10, 20, 1000))
	assert.Equal(t, err, ErrNoBalance)
}

func TestTransferZeroAccoutSkipNoBalance(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	_, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(0, 20, 1000))
	assert.NoError(t, err)
}

func TestBatchReceivers(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	transfer := pt.Transfer{}
	transfer.AddReceiver(20, 2000)
	transfer.AddReceiver(30, 3000)
	transfer.AddReceiver(40, 4000)

	_, err := p.ProcessTransfer(context.TODO(), transfer)
	assert.NoError(t, err)
}

func TestProcessCheckSettingsID(t *testing.T) {
	p := NewProcessor(chain.NewChain())
	sc := chain.NewSettingsChain()
	p.SetSettingsChain(sc)

	prv, err := btcec.NewPrivateKey(pt.SigningCurve)
	assert.NoError(t, err)

	pub := prv.PubKey()

	sc.Put(&pt.Settings{
		ID:        1,
		Account:   5,
		PublicKey: pub.SerializeHybrid(),
	})

	// invalid settingsID
	res, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(5, 4, 0))
	assert.Equal(t, ErrInvalidSettingsID, err)
	assert.Equal(t, pt.TransferResult{
		SettingsId: 1,
	}, res)
}

func TestProcessCheckSign(t *testing.T) {
	p := NewProcessor(chain.NewChain())
	sc := chain.NewSettingsChain()
	p.SetSettingsChain(sc)

	prv, err := btcec.NewPrivateKey(pt.SigningCurve)
	assert.NoError(t, err)

	pub := prv.PubKey()

	sc.Put(&pt.Settings{
		ID:        1,
		Account:   5,
		PublicKey: pub.SerializeHybrid(),
	})

	// zero sign
	tr := pt.NewSingleTransfer(5, 4, 0)
	tr.SettingsID = 1
	res, err := p.ProcessTransfer(context.TODO(), tr)
	assert.Equal(t, ErrInvalidSign, err)
	assert.Equal(t, pt.TransferResult{
		SettingsId: 1,
	}, res)

	// invalid sign
	tr = pt.NewSingleTransfer(5, 4, 0)
	tr.SettingsID = 1
	copy(tr.Sign[:], "wrong sign")
	res, err = p.ProcessTransfer(context.TODO(), tr)
	assert.Equal(t, ErrInvalidSign, err)
	assert.Equal(t, pt.TransferResult{
		SettingsId: 1,
	}, res)

	// right sign
	tr = pt.NewSingleTransfer(5, 4, 0)
	tr.SettingsID = 1
	hash := pt.GetTransferHashDefault(tr)
	tr.Sign, err = pt.SignTransfer(hash, prv)
	assert.NoError(t, err)
	expect := pt.HashFromString("2d6e6f0639918056d40f97d29678392289bab5d4d0d2f626910f5f21be6a462c")

	res, err = p.ProcessTransfer(context.TODO(), tr)
	assert.NoError(t, err)
	assert.Equal(t, pt.TransferResult{
		TxnID:      pt.NewTxnID(5, 1),
		Hash:       expect,
		SettingsId: 1,
	}, res)

	// right sign without public key
	sc.Put(&pt.Settings{
		ID:      2,
		Account: 5,
	})
	hash, err = p.GetPrevHash(context.TODO(), 5)
	assert.NoError(t, err)
	tr = pt.NewSingleTransfer(5, 4, 0)
	tr.SettingsID = 2
	tr.PrevHash = hash
	hash = pt.GetTransferHashDefault(tr)
	tr.Sign, err = pt.SignTransfer(hash, prv)
	assert.NoError(t, err)

	res, err = p.ProcessTransfer(context.TODO(), tr)
	assert.Equal(t, ErrInvalidSign, err)
	assert.Equal(t, pt.TransferResult{
		Hash:       expect,
		SettingsId: 2,
	}, res)

	// broken public key
	sc.Put(&pt.Settings{
		ID:        3,
		Account:   5,
		PublicKey: []byte("qwer"),
	})
	hash, err = p.GetPrevHash(context.TODO(), 5)
	assert.NoError(t, err)
	tr = pt.NewSingleTransfer(5, 4, 0)
	tr.SettingsID = 3
	tr.PrevHash = hash
	hash = pt.GetTransferHashDefault(tr)
	tr.Sign, err = pt.SignTransfer(hash, prv)
	assert.NoError(t, err)

	res, err = p.ProcessTransfer(context.TODO(), tr)
	assert.Error(t, err, "invalid pub key length 4")
	assert.Equal(t, pt.TransferResult{
		Hash:       expect,
		SettingsId: 3,
	}, res)

	// sign and no settings
	p.SetSettingsChain(chain.NewSettingsChain())
	hash, err = p.GetPrevHash(context.TODO(), 5)
	assert.NoError(t, err)
	tr = pt.NewSingleTransfer(5, 4, 0)
	tr.SettingsID = 2
	tr.PrevHash = hash
	hash = pt.GetTransferHashDefault(tr)
	tr.Sign, err = pt.SignTransfer(hash, prv)
	assert.NoError(t, err)

	res, err = p.ProcessTransfer(context.TODO(), tr)
	assert.Equal(t, ErrInvalidSign, err)
	assert.Equal(t, pt.TransferResult{
		Hash: expect,
	}, res)
}

func BenchmarkProcessSingleTransfer(b *testing.B) {
	p := NewProcessor(chain.NewChain())
	transfer := pt.NewSingleTransfer(0, 20, 1000)
	prevhash := pt.Hash{}
	for i := 0; i < b.N; i++ {
		transfer.PrevHash = prevhash
		res, err := p.ProcessTransfer(context.TODO(), transfer)
		assert.NoError(b, err)
		prevhash = res.Hash
	}
	b.ReportAllocs()
}

func BenchmarkProcessSingleTransferWithLargeUnspentInputs(b *testing.B) {
	p := NewProcessor(chain.NewChain())
	transfer := pt.Transfer{}
	prevhash := map[pt.AccID]pt.Hash{}

	// build N receivers
	num := 10000
	for i := 0; i < num; i++ {
		transfer.AddReceiver(pt.AccID(i), 0)
	}

	for i := 0; i < b.N; i++ {
		transfer.Sender = pt.AccID(i % num)

		transfer.PrevHash = prevhash[transfer.Sender]
		res, err := p.ProcessTransfer(context.TODO(), transfer)
		assert.NoError(b, err)

		prevhash[transfer.Sender] = res.Hash
	}

	b.ReportAllocs()
}

func BenchmarkProcessTransfer1000Receivers(b *testing.B) {
	p := NewProcessor(chain.NewChain())
	transfer := pt.NewSingleTransfer(0, 20, 1000)
	for i := 0; i < 1000; i++ {
		// don't use same receiver (0 -> 0),
		// because inputs processing complexity became O(n^2)
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

func BenchmarkProcessTransfer32Receivers(b *testing.B) {
	p := NewProcessor(chain.NewChain())
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

func TestProcessTransfer(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	res, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(0, 20, 1000))
	if assert.NoError(t, err) {
		assert.Equal(t, pt.TransferResult{
			TxnID: pt.TxnID{AccID: 0, ID: 1},
			Hash:  pt.HashFromString("2d2be2c4259955feff587e567aee7ae3e99b5022289813cbacbfaaeb48df4d16"),
		}, res)
	}

	transfer := pt.NewSingleTransfer(0, 30, 1000)
	transfer.PrevHash = res.Hash

	res, err = p.ProcessTransfer(context.TODO(), transfer)
	if assert.NoError(t, err) {
		assert.Equal(t, pt.TransferResult{
			TxnID: pt.TxnID{AccID: 0, ID: 2},
			Hash:  pt.HashFromString("660d2d296c5d8b18261a84a2e3b77cb9cf1b3c700733ea3fe7b1931dc61d4ace"),
		}, res)
	}
}

func TestProcessTransferSetLastHash(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	res, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(0, 20, 1000))
	if assert.NoError(t, err) {
		assert.Equal(t, pt.TransferResult{
			TxnID: pt.TxnID{AccID: 0, ID: 1},
			Hash:  pt.HashFromString("2d2be2c4259955feff587e567aee7ae3e99b5022289813cbacbfaaeb48df4d16"),
		}, res)
	}

	transfer := pt.NewSingleTransfer(0, 30, 1000)
	transfer.PrevHash = pt.HashFromString("2d2be2c4259955feff587e567aee7ae3e99b5022289813cbacbfaaeb48df4d16")

	res, err = p.ProcessTransfer(context.TODO(), transfer)
	if assert.NoError(t, err) {
		assert.Equal(t, pt.TransferResult{
			TxnID: pt.TxnID{AccID: 0, ID: 2},
			Hash:  pt.HashFromString("660d2d296c5d8b18261a84a2e3b77cb9cf1b3c700733ea3fe7b1931dc61d4ace"),
		}, res)
	}
}

func TestGetPrevHashPreloaderError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	prel := mocks.NewMockPreloader(mock)

	p := NewProcessor(chain.NewChain())
	p.SetPreloader(prel)

	testErr := errors.New("test error")

	prel.EXPECT().Preload(gomock.Any(), gomock.Any()).Return(testErr)

	_, err := p.GetPrevHash(context.TODO(), 4)
	assert.EqualError(t, err, "account preloading: "+testErr.Error())
}

func TestGetBalancePreloaderError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	prel := mocks.NewMockPreloader(mock)

	p := NewProcessor(chain.NewChain())
	p.SetPreloader(prel)

	testErr := errors.New("test error")

	prel.EXPECT().Preload(gomock.Any(), gomock.Any()).Return(testErr)

	_, err := p.GetBalance(context.TODO(), 4)
	assert.EqualError(t, err, "account preloading: "+testErr.Error())
}

func TestTransferPreloaderError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	prel := mocks.NewMockPreloader(mock)

	p := NewProcessor(chain.NewChain())
	p.SetPreloader(prel)

	testErr := errors.New("test error")

	prel.EXPECT().Preload(gomock.Any(), gomock.Any()).Return(testErr)

	_, err := p.ProcessTransfer(context.TODO(), pt.Transfer{Sender: 4, Batch: []*pt.TransferItem{{Receiver: 3, Amount: 1}}})
	assert.EqualError(t, err, "account preloading: "+testErr.Error())
}

func TestSettingsGetLastPreloaderError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	prel := mocks.NewMockPreloader(mock)

	p := NewSettingsProcessor(chain.NewSettingsChain())
	p.SetPreloader(prel)

	testErr := errors.New("test error")

	prel.EXPECT().Preload(gomock.Any(), gomock.Any()).Return(testErr)

	_, err := p.GetLastSettings(context.TODO(), 4)
	assert.EqualError(t, err, "account preloading: "+testErr.Error())
}

func TestProcessSettingsGetLastPreloaderError(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	prel := mocks.NewMockPreloader(mock)

	p := NewSettingsProcessor(chain.NewSettingsChain())
	p.SetPreloader(prel)

	testErr := errors.New("test error")

	prel.EXPECT().Preload(gomock.Any(), gomock.Any()).Return(testErr)

	_, err := p.ProcessSettings(context.TODO(), &pt.Settings{Account: 4})
	assert.EqualError(t, err, "account preloading: "+testErr.Error())
}

type fakeFailingPusher struct{}

func (p *fakeFailingPusher) Push(ctx context.Context, txns []pt.Txn) error {
	return errors.New("fake internal error")
}

func TestProcessTransferWithPusher(t *testing.T) {
	p := NewProcessor(chain.NewChain())

	p.SetPusher(&fakeFailingPusher{})

	_, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(0, 20, 1000))
	assert.EqualError(t, errors.Cause(err), "fake internal error")
}

func TestProcessMustPushTxns(t *testing.T) {
	c := chain.NewChain()
	c2 := chain.NewChain()

	p := NewProcessor(c)
	p.SetPusher(pusher.NewChainReceiversPusher(c2))

	p2 := NewProcessor(c2)
	p2.SetPusher(pusher.NewChainReceiversPusher(c))

	_, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(20, 30, 0))
	assert.NoError(t, err)
	assert.Nil(t, c2.GetLastTxn(20))
	assert.Equal(t, []pt.Txn{*c.GetLastTxn(20)}, c2.ListUnspentTxns(30))

	_, err = p2.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(30, 20, 0))
	assert.NoError(t, err)
	assert.Nil(t, c.GetLastTxn(30))
	assert.Equal(t, []pt.Txn{*c2.GetLastTxn(30)}, c.ListUnspentTxns(20))

}

func TestBalanceAfterSpendInputs(t *testing.T) {
	c := chain.NewChain()
	p := NewProcessor(c)

	p.SetPusher(pusher.NewChainReceiversPusher(c))

	_, err := p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(0, 20, 1000))
	assert.NoError(t, err)

	_, err = p.ProcessTransfer(context.TODO(), pt.NewSingleTransfer(20, 30, 1000))
	assert.NoError(t, err)

	assert.Equal(t, int64(-1000), c.GetBalance(0))
	assert.Equal(t, int64(0), c.GetBalance(20))
	assert.Equal(t, int64(1000), c.GetBalance(30))
}

func TestProcessSettings(t *testing.T) {
	p := NewSettingsProcessor(chain.NewSettingsChain())

	s := &pt.Settings{Account: 10}
	res, err := p.ProcessSettings(context.TODO(), s)
	assert.NoError(t, err)

	assert.Equal(t, pt.SettingsResult{
		SettingsID: pt.NewSettingsID(10, 1),
		Hash:       pt.HashFromString("74280ac6f6059268431efe2a9b8903957a08f7685e651c6cbbedcd83c101bcc5"),
	}, res)

	s = &pt.Settings{Account: 10, PrevHash: s.Hash}
	res, err = p.ProcessSettings(context.TODO(), s)
	assert.NoError(t, err)

	assert.Equal(t, pt.SettingsResult{
		SettingsID: pt.NewSettingsID(10, 2),
		Hash:       pt.HashFromString("0d4e1907ce02854ea140eaf16f09d97d218f5d79e710867777fb152cfc3f1362"),
	}, res)
}

func TestProcessSettingsSign(t *testing.T) {
	c := chain.NewSettingsChain()
	p := NewSettingsProcessor(c)

	key, err := hex.DecodeString("ba857b4da5b94a9c71d7365a00f8dfe04fd0dc860cbe6cb3e3cb592102dacb2c")
	assert.NoError(t, err)

	prv, pub := btcec.PrivKeyFromBytes(pt.SigningCurve, key)
	assert.NoError(t, err)

	_ = prv

	// set public key
	s := &pt.Settings{ID: 1, Account: 10, PublicKey: pub.SerializeHybrid()}
	c.Put(s)

	// no sign
	s = &pt.Settings{Account: 10, PublicKey: pub.SerializeHybrid()}
	_, err = p.ProcessSettings(context.TODO(), s)
	assert.EqualError(t, err, ErrInvalidSign.Error())

	// signed
	s = &pt.Settings{Account: 10, PublicKey: pub.SerializeHybrid()}
	hash := pt.GetSettingsRequestHashDefault(s)
	s.Sign, err = pt.SignTransfer(hash, prv)
	assert.NoError(t, err)
	res, err := p.ProcessSettings(context.TODO(), s)
	assert.NoError(t, err)
	assert.Equal(t, pt.SettingsResult{SettingsID: pt.NewSettingsID(10, 2), Hash: pt.HashFromString("2b7742f16fc17283936ee227d0808ad3496edcd3a8d51ebb058d0d1fa3145583")}, res)

	// no public key, but signed
	s = &pt.Settings{ID: 3, Account: 10}
	c.Put(s)

	s = &pt.Settings{Account: 10, PublicKey: pub.SerializeHybrid()}
	hash = pt.GetSettingsRequestHashDefault(s)
	s.Sign, err = pt.SignTransfer(hash, prv)
	assert.NoError(t, err)
	_, err = p.ProcessSettings(context.TODO(), s)
	assert.EqualError(t, err, "processor: invalid sign")

	// broken pubkey
	s = &pt.Settings{ID: 4, Account: 10, PublicKey: []byte("qwer")}
	c.Put(s)
	_, err = p.ProcessSettings(context.TODO(), s)
	assert.EqualError(t, err, "invalid pub key length 4")
}

func TestProcessSettingsWithLast(t *testing.T) {
	p := NewSettingsProcessor(chain.NewSettingsChain())
	res, err := p.ProcessSettings(context.TODO(), &pt.Settings{Account: 10})
	assert.NoError(t, err)

	res, err = p.ProcessSettings(context.TODO(), &pt.Settings{Account: 10, PrevHash: res.Hash})
	assert.NoError(t, err)
	assert.Equal(t, pt.SettingsResult{
		SettingsID: pt.NewSettingsID(10, 2),
		Hash:       pt.HashFromString("0d4e1907ce02854ea140eaf16f09d97d218f5d79e710867777fb152cfc3f1362"),
	}, res)
}

func TestProcessSettingsWithInvalidPrevHash(t *testing.T) {
	p := NewSettingsProcessor(chain.NewSettingsChain())
	s := &pt.Settings{Account: 10, PrevHash: pt.HashFromString("10")}

	res, err := p.ProcessSettings(context.TODO(), s)
	assert.Equal(t, pt.SettingsResult{}, res)
	// TODO(outself): errors.Cause
	assert.EqualError(t, err, "settings processor: invalid prev hash")
}

func TestGetPrevHash(t *testing.T) {
	c := chain.NewChain()
	p := NewProcessor(c)

	h, err := p.GetPrevHash(context.TODO(), pt.AccID(10))
	assert.NoError(t, err)
	assert.Equal(t, pt.ZeroHash, h)
}

func TestGetBalance(t *testing.T) {
	c := chain.NewChain()
	p := NewProcessor(c)

	b, err := p.GetBalance(context.TODO(), pt.AccID(10))
	assert.NoError(t, err)
	assert.Equal(t, int64(0), b)
}

func TestGetLastSettings(t *testing.T) {
	c := chain.NewSettingsChain()
	p := NewSettingsProcessor(c)

	b, err := p.GetLastSettings(context.TODO(), pt.AccID(10))
	assert.NoError(t, err)
	assert.Equal(t, (*pt.Settings)(nil), b)
}
