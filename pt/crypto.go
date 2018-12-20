// +build !gost

package pt

import (
	"crypto/sha256"
	"log"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

type (
	Hash [32]byte
	Sign [72]byte

	PublicKey []byte
)

var (
	HashNew      = sha256.New
	SigningCurve = btcec.S256()
)

func (k PublicKey) BTCECKey() (*btcec.PublicKey, error) {
	return btcec.ParsePubKey(k, SigningCurve)
}

func VerifyTransferHash(sign Sign, transferHash Hash, publicKey *btcec.PublicKey) error {
	signature, err := btcec.ParseSignature(sign[:], SigningCurve)
	if err != nil {
		return errors.Wrap(err, "parse transfer signature")
	}

	if signature.Verify(transferHash[:], publicKey) {
		return nil
	}

	return errors.New("invalid transfer sign")
}

func SignTransfer(transferHash Hash, privKey *btcec.PrivateKey) (Sign, error) {
	sig, err := privKey.Sign(transferHash[:])
	if err != nil {
		return Sign{}, errors.Wrap(err, "sign failed")
	}
	buf := sig.Serialize()
	var s Sign
	if len(s) < len(buf) {
		log.Fatalf("sign size differs: type(%d), sign(%d)", len(s), len(buf))
	}
	copy(s[:], buf)
	return s, nil
}
