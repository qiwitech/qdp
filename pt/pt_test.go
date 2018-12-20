package pt

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTxnID(t *testing.T) {
	txnid := &TxnID{AccID: 1, ID: 10}
	assert.Equal(t, "1_10", txnid.String())
}

func TestSettingsID(t *testing.T) {
	settid := &SettingsID{AccID: 1, ID: 10}
	assert.Equal(t, "1_10", settid.String())
}

func TestAccountID(t *testing.T) {
	id := AccID(4)
	assert.Equal(t, "4", id.String())
}

func BenchmarkTxnIdToString(b *testing.B) {
	txnid := &TxnID{AccID: 1e4, ID: 1e6}
	for i := 0; i < b.N; i++ {
		_ = txnid.String()
	}
}

func TestNewSingleTransfer(t *testing.T) {
	assert.Equal(t, Transfer{Sender: 10, Batch: []*TransferItem{{Receiver: 20, Amount: 30}}}, NewSingleTransfer(10, 20, 30))
}

func TestNewTxnID(t *testing.T) {
	assert.Equal(t, TxnID{AccID: 10, ID: 20}, NewTxnID(10, 20))
}

func TestNewSettingsID(t *testing.T) {
	assert.Equal(t, SettingsID{AccID: 10, ID: 20}, NewSettingsID(10, 20))
}

func TestTransferAddReceiver(t *testing.T) {
	ts := &Transfer{}

	ts.AddReceiver(10, 20)

	assert.NotEmpty(t, ts.Batch)

	ts.AddReceiver(10, 20)
	assert.Len(t, ts.Batch, 2)
}

func TestHashToString(t *testing.T) {
	str := "0102030405060708"
	th := HashFromString(str)

	assert.Equal(t, str+strings.Repeat("0", 2*len(th)-len(str)), th.String())
}

func TestHashFromString(t *testing.T) {
	th := HashFromString("10")
	assert.Equal(t, Hash{0x10}, th)
}

func TestSignFromString(t *testing.T) {
	th := SignFromString("10")
	assert.Equal(t, Sign{0x10}, th)
}

func TestInvalidHashFromString(t *testing.T) {
	assert.Panics(t, func() {
		HashFromString("zz")
	})

	assert.Panics(t, func() {
		HashFromString("1")
	})

	assert.Panics(t, func() {
		HashFromString(" ")
	})

}

func TestHash(t *testing.T) {
	txn := &Txn{
		ID:       1,
		Sender:   10,
		Receiver: 20,
		Amount:   2000,
		Balance:  3000,
		SpentBy:  100,
		PrevHash: HashFromString("123123"),
	}

	assert.Equal(t, HashFromString("fb175d2e658883ed5adae1e15602a70fc7d132b38d03cc42af86350319b766ab"), GetHashDefault(txn))
}

func BenchmarkGetHash(b *testing.B) {
	txn := &Txn{
		ID:       1,
		Sender:   10,
		Receiver: 20,
		Amount:   2000,
		Balance:  3000,
		SpentBy:  100,
		PrevHash: HashFromString("1337"),
	}

	for i := 0; i < b.N; i++ {
		_ = GetHashDefault(txn)
	}

	b.ReportAllocs()
}

func BenchmarkCalcHash(b *testing.B) {
	txn := &Txn{
		ID:       1,
		Sender:   10,
		Receiver: 20,
		Amount:   2000,
		Balance:  3000,
		SpentBy:  100,
		PrevHash: HashFromString("1337"),
	}
	h := sha256.New()

	for i := 0; i < b.N; i++ {
		_ = GetHash(h, txn)
	}

	b.ReportAllocs()
}

func TestGetSettingsHash(t *testing.T) {
	s := &Settings{ID: 1, Account: 20, VerifyTransferSign: true, PublicKey: []byte("public_key"), PrevHash: HashFromString("123123")}
	assert.Equal(t, HashFromString("c85a0e427c75f05bafd0f873b9b98ee4bc2f3e6a9f71388ec0f7391fb509fc68"), GetSettingsHashDefault(s))
	s.VerifyTransferSign = false
	assert.Equal(t, HashFromString("437937ce2e5b49b0f9e4a18b034491c2ceec1d051b325ea9c122fb8ef5fca57c"), GetSettingsHashDefault(s))
}

func TestGetTransferHash(t *testing.T) {
	transfer := NewSingleTransfer(0, 10, 20)
	transfer.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")
	transfer.SettingsID = 100

	h := GetTransferHashDefault(transfer)
	assert.Equal(t, HashFromString("f4e6b88b76c9dd326b2fff91a0848ece9a6e20c1a44ffdba724dddccf507df42"), h)
}

func TestSignTransfer(t *testing.T) {
	transfer := NewSingleTransfer(0, 10, 20)
	transfer.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")
	transfer.SettingsID = 100

	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}

	h := GetTransferHashDefault(transfer)

	sign, err := SignTransfer(h, priv)
	assert.NoError(t, err)
	assert.NotEqual(t, Sign{}, sign)
}

func TestGetSettingsRequestHash(t *testing.T) {
	sett := &Settings{
		ID:                 4,
		Account:            10,
		VerifyTransferSign: true,
	}
	sett.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")

	h := GetSettingsRequestHashDefault(sett)
	assert.Equal(t, HashFromString("dc337b2f39737299b170bf4b8f62fa803bf2e1ea52fab9d067ad7d95e0bf25e8"), h)
}

func TestGetSettingsRequestHashNotVerify(t *testing.T) {
	sett := &Settings{
		ID:                 4,
		Account:            10,
		VerifyTransferSign: false,
	}
	sett.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")

	h := GetSettingsRequestHashDefault(sett)
	assert.Equal(t, HashFromString("32882373cbb34f2fdb25f5dd60e1fc1df97c6656ba34b4f6d60c4d25ee1c60c9"), h)
}

func TestPubKey(t *testing.T) {
	pk, err := ParsePubKey("")
	assert.NoError(t, err)
	assert.Nil(t, pk)

	priv, err := btcec.NewPrivateKey(btcec.S256())
	assert.NoError(t, err)

	pub := priv.PubKey()

	ks := base58.Encode(pub.SerializeHybrid())
	pk, err = ParsePubKey(ks)
	assert.NoError(t, err)

	assert.Equal(t, ks, pk.String())

	k, err := pk.BTCECKey()
	assert.NoError(t, err)

	assert.Equal(t, pub, k)
}

/*
func TestSignTransferInvalidPrivKey(t *testing.T) {
	transfer := NewSingleTransfer(0, 10, 20)
	transfer.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")
	transfer.SettingsID = 100

	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}

	//h := GetTransferHashDefault(transfer)

	sign, err := SignTransfer(Hash{}, priv)
	assert.NoError(t, err)
	assert.NotEqual(t, Sign{}, sign)
}*/

func TestVerifyTransferSign(t *testing.T) {
	transfer := NewSingleTransfer(0, 10, 20)
	transfer.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")
	transfer.SettingsID = 100

	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}

	h := GetTransferHashDefault(transfer)

	sign, err := SignTransfer(h, priv)
	assert.NoError(t, err)
	assert.NotEqual(t, Sign{}, sign)

	err = VerifyTransferHash(sign, h, priv.PubKey())
	assert.NoError(t, err)

	err = VerifyTransferHash(Sign{0, 0, 0, 0}, h, priv.PubKey())
	assert.EqualError(t, errors.Cause(err), "malformed signature: no header magic")

	err = VerifyTransferHash(Sign{48, 69, 2, 33, 0, 153, 88, 73, 140, 165, 24, 39, 11, 104, 216, 168, 24, 56, 187, 126, 252, 7, 33, 11, 184, 205, 175, 55, 246, 55, 219, 172, 1, 151, 242, 123, 78, 2, 32, 107, 87, 248, 235, 157, 92, 157, 117, 219, 29, 247, 178, 222, 31, 127, 212, 175, 8, 50, 215, 212, 240, 112, 114, 173, 103, 0, 178, 121, 161, 96, 125, 0}, h, priv.PubKey())
	assert.EqualError(t, err, "invalid transfer sign")
}

func BenchmarkSignTransfer(b *testing.B) {
	transfer := NewSingleTransfer(0, 10, 20)
	transfer.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")
	transfer.SettingsID = 100

	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}

	h := GetTransferHashDefault(transfer)

	for i := 0; i < b.N; i++ {
		_, err := SignTransfer(h, priv)
		assert.NoError(b, err)
	}

	b.ReportAllocs()
}

func BenchmarkVerifyTransferSign(b *testing.B) {
	transfer := NewSingleTransfer(0, 10, 20)
	transfer.PrevHash = HashFromString("d1365234717958d8489b700f900bfaa0ecf0db5b137c25a5b43058de75f118a1")
	transfer.SettingsID = 100

	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}

	h := GetTransferHashDefault(transfer)
	pub := priv.PubKey()

	sign, err := SignTransfer(h, priv)
	assert.NoError(b, err)

	for i := 0; i < b.N; i++ {
		err = VerifyTransferHash(sign, h, pub)
		assert.NoError(b, err)
	}

	b.ReportAllocs()
}

func TestSignStringify(t *testing.T) {
	s := Sign{}
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", s.String())
}

func TestGetSignFromString(t *testing.T) {
	s, err := GetSignFromString("100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)
	assert.Equal(t, Sign{0x10}, s)

	s, err = GetSignFromString("abc")
	assert.EqualError(t, err, "invalid sign string")
	assert.Equal(t, ZeroSign, s)
}

func TestGetHashFromString(t *testing.T) {
	h, err := GetHashFromString("10")
	assert.NoError(t, err)
	assert.Equal(t, Hash{0x10}, h)

	h, err = GetHashFromString("abc")
	assert.EqualError(t, err, "invalid hash string")
	assert.Equal(t, ZeroHash, h)
}

/*
func TestCoverStrings(t *testing.T) {
	txn := &Txn{Sender: 10, Receiver: 30, ID: 4}
	txn.String()
}*/
