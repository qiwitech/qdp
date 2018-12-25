package client

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	bolt "github.com/coreos/bbolt"

	"github.com/qiwitech/qdp/proto/apipb"
	"github.com/qiwitech/qdp/pt"
)

var (
	keys  *bolt.DB
	curve elliptic.Curve = btcec.S256()

	buf [8]byte

	KeysDBFlag string

	keysMu sync.Mutex
)

func SignTransfer(req *apipb.TransferRequest) error {
	priv, err := LoadPrivateKey(req.Sender)
	if err != nil {
		return err
	}

	if priv == nil {
		return nil
	}

	hash := TransferRequestHash(req)
	sign, err := pt.SignTransfer(hash, priv)
	if err != nil {
		return err
	}

	req.Sign = sign.String()

	return nil
}

func SavePrivateKey(account uint64, priv *btcec.PrivateKey) error {
	defer keysMu.Unlock()
	keysMu.Lock()

	if err := openKeysDB(); err != nil {
		return err
	}
	if keys == nil {
		return nil
	}

	binary.BigEndian.PutUint64(buf[:8], account)
	pkb := priv.Serialize()

	err := keys.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("keys"))
		if err != nil {
			return err
		}

		return b.Put(buf[:8], pkb)
	})
	if err != nil {
		return err
	}

	return nil
}

func LoadPrivateKey(account uint64) (*btcec.PrivateKey, error) {
	defer keysMu.Unlock()
	keysMu.Lock()

	if err := openKeysDB(); err != nil {
		return nil, err
	}
	if keys == nil {
		return nil, nil
	}

	binary.BigEndian.PutUint64(buf[:8], account)

	var pkb []byte
	err := keys.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("keys"))
		if b == nil {
			return nil
		}

		v := b.Get(buf[:8])
		pkb = dup(v)

		return nil
	})
	if pkb == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	priv, _ := btcec.PrivKeyFromBytes(curve, pkb)

	return priv, nil
}

func openKeysDB() error {
	if keys != nil {
		return nil
	}
	if KeysDBFlag == "" {
		return nil
	}

	f := KeysDBFlag

	db, err := bolt.Open(f, 0644, &bolt.Options{
		NoFreelistSync: true,
	})
	if err != nil {
		return err
	}

	keys = db

	return nil
}

func TransferRequestHash(t *apipb.TransferRequest) pt.Hash {
	h := sha256.New()
	order := binary.BigEndian

	var hbuf pt.Hash
	buf := hbuf[:]

	order.PutUint64(buf, uint64(t.Sender))
	h.Write(buf[:8])

	for _, ti := range t.Batch {
		order.PutUint64(buf, uint64(ti.Receiver))
		h.Write(buf[:8])

		order.PutUint64(buf, uint64(ti.Amount))
		h.Write(buf[:8])
	}

	ph, err := hex.DecodeString(t.PrevHash)
	if err != nil {
		panic("prev hash format")
	}
	h.Write(ph)

	order.PutUint64(buf, uint64(t.SettingsId))
	h.Write(buf[:8])

	h.Sum(buf[:0])
	return hbuf
}

func SettingsRequestHash(s *apipb.SettingsRequest) pt.Hash {
	h := sha256.New()

	order := binary.BigEndian
	var hbuf pt.Hash
	buf := hbuf[:]

	// TODO(outself): disable error check lint. write errors unnecessary
	order.PutUint64(buf, uint64(s.Account))
	h.Write(buf[:8])

	if s.VerifyTransferSign {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	h.Write(buf[:1])

	hash, err := hex.DecodeString(s.PrevHash)
	if err != nil {
		panic(err)
	}
	h.Write(hash)

	pk, err := pt.ParsePubKey(s.PublicKey)
	if err != nil {
		panic(err)
	}
	h.Write(pk)

	hash, err = hex.DecodeString(s.DataHash)
	if err != nil {
		panic(err)
	}
	h.Write(hash)

	_ = h.Sum(buf[:0])
	return hbuf
}

func dup(v []byte) []byte {
	if len(v) == 0 {
		return nil
	}
	r := make([]byte, len(v))
	copy(r, v)
	return r
}
