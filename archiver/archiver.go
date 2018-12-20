// archiver takes transactions and settings, collects them to blocks, calculates hash of blocks, creates archives,
// pushes hash to some blockchain(s), uploads archives to external storage
//
//	you can easily verify block hash by command: tar xf archive.tar -O --wildcards "*/settings" "*/txns" | sha256sum
//	you can even get hasher command like this: `tar xf archive.tar -O --wildcards "*/metadata.json" | jq -r .HashType`sum
//	and by the same way you can extract block hash: tar xf archiver.tar -O --wildcards "*/metadata.json" | jq -r .Hash
//
// DB buckets
//	Txns   : <block_hash>/<type>/<txn_id> -> <txn json>
//	Index  : <type>/<txn_id> -> <block_hash>
//	Blocks : <block_hash> -> <block>
//
package archiver

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/proto/archiverpb"
	"github.com/qiwitech/qdp/pt"
)

/*
DB buckets
* Txns   : <block_hash>/<type>/<txn_id> -> <txn json>
* Index  : <type>/<txn_id> -> <block_hash>
* Blocks : <block_hash> -> <block>

*/

type BlockHash [sha256.Size]byte

var ZeroBlockHash BlockHash

const (
	TxnsBucket   = "txns"
	BlocksBucket = "blocks"
	IndexBucket  = "idx"

	TxnsDir     = "txns"
	SettingsDir = "settings"

	LastBlock = "last"
)

type Status int

const (
	NOTFOUND Status = iota
	UNPROCESSED
	PREPROCESSED
	FIXED
	UPLOADED
)

type HashType string

const (
	HashSHA256 = "sha256"
)

var errNoNewData = errors.New("no new data available")

var (
	now func() int64 = func() int64 {
		return time.Now().UnixNano()
	}
	timestr = func(t int64) string {
		return time.Unix(0, t).Format(time.RFC3339)
	}
)

type MinioConfig struct {
	Bucket   string
	Prefix   string
	Location string
	//	ACL          string
	//	StorageClass string

	Client *minio.Client
}

type Archiver struct {
	db         *bolt.DB
	rotateSize int
	blockSize  int

	minio *MinioConfig

	upload func(*BlockMeta, []byte) error

	rotateC chan struct{}
}

type BlockMeta struct {
	Hash     BlockHash `json:"hash"`
	PrevHash BlockHash `json:"prev_hash"`
	Time     string    `json:"time"`

	StorageBucket string `json:"storage_bucket,omitempty"`
	StoragePath   string `json:"storage_path,omitempty"`

	Status Status `json:"status"`

	HashType HashType `json:"hash_type"`
}

func New(db *bolt.DB, rotateSize int) (*Archiver, error) {
	a := &Archiver{db: db,
		rotateSize: rotateSize,
		rotateC:    make(chan struct{}, 1),
	}
	a.upload = a.uploadToMinio

	err := a.recover()
	if err != nil {
		return nil, errors.Wrap(err, "check and recover")
	}

	return a, nil
}

func (a *Archiver) SetMinio(c *MinioConfig) error {
	ok, err := c.Client.BucketExists(c.Bucket)
	if err != nil {
		return errors.Wrap(err, "check bucket existence")
	}
	if !ok {
		err = c.Client.MakeBucket(c.Bucket, c.Location)
		if err != nil {
			return errors.Wrap(err, "create unexisted bucket")
		}
	}

	ok, err = c.Client.BucketExists(c.Bucket)
	if err != nil {
		return errors.Wrap(err, "check bucket existence")
	}
	if !ok {
		return fmt.Errorf("created bucket does not exists: '%v' at '%v'", c.Bucket, c.Location)
	} else {
		log.Printf("bucket '%v' exists", c.Bucket)
	}

	a.minio = c

	return nil
}

func (a *Archiver) recover() error {
	return a.db.Update(func(tx *bolt.Tx) error {
		tb, err := tx.CreateBucketIfNotExists([]byte(TxnsBucket))
		if err != nil {
			return err
		}
		bb, err := tx.CreateBucketIfNotExists([]byte(BlocksBucket))
		if err != nil {
			return err
		}
		ib, err := tx.CreateBucketIfNotExists([]byte(IndexBucket))
		if err != nil {
			return err
		}
		_ = ib

		checked := make(map[string]struct{})
		c := tb.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			bh, _, _ := parseTxnKey(k)
			if _, ok := checked[bh.String()]; ok {
				continue
			}
			bdata := bb.Get([]byte(bh.String()))
			var meta BlockMeta
			err := json.Unmarshal(bdata, &meta)
			if err != nil {
				return errors.Wrap(err, "unmarshal block")
			}

			if meta.Status != UPLOADED {
				return fmt.Errorf("unuploaded block %v", bh)
			}

			checked[bh.String()] = struct{}{}
		}
		return nil
	})
}

func (a *Archiver) Start(d time.Duration) error {
	t := time.NewTicker(d)
	defer t.Stop()

	var err error
	for {
		select {
		case <-t.C:
			err = a.Rotate()
		case <-a.rotateC:
			err = a.Rotate()
		}

		if err != nil {
			return err
		}
	}
}

func (a *Archiver) Rotate() error {
	var buf bytes.Buffer
	var meta BlockMeta

	err := a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TxnsBucket))
		c := b.Cursor()

		h := sha256.New()
		meta.HashType = HashSHA256

		cnt := 0
		zpref := []byte(ZeroBlockHash.String())
		// calc hash
		for k, v := c.Seek(zpref); bytes.HasPrefix(k, zpref); k, v = c.Next() {
			cnt++
			_, _ = h.Write(v)
		}

		if cnt == 0 {
			return errNoNewData
		}

		meta.Hash = NewBlockHash(h.Sum(nil))
		meta.PrevHash = NewBlockHashFromString(string(tx.Bucket([]byte(BlocksBucket)).Get([]byte(LastBlock))))
		meta.Time = timestr(now())

		metadata, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		jw := tar.NewWriter(&buf)
		err = jw.WriteHeader(&tar.Header{
			Name: meta.Hash.String() + "/metadata.json",
			Mode: 0660,
			Size: int64(len(metadata)),
		})
		if err != nil {
			return err
		}
		_, err = jw.Write(metadata)
		if err != nil {
			return err
		}

		for k, v := c.Seek(zpref); bytes.HasPrefix(k, zpref); k, v = c.Next() {
			_, dir, tid := parseTxnKey(k)

			err = jw.WriteHeader(&tar.Header{
				Name: meta.Hash.String() + "/" + dir + "/" + tid.String() + ".json",
				Mode: 0660,
				Size: int64(len(v)),
			})
			if err != nil {
				return err
			}
			_, err = jw.Write(v)
			if err != nil {
				return err
			}

			err = b.Delete(k)
			if err != nil {
				return err
			}

			err = b.Put(txnKey(meta.Hash, dir, tid), v)
			if err != nil {
				return err
			}

			err = tx.Bucket([]byte(IndexBucket)).Put(idxKey(dir, tid), []byte(meta.Hash.String()))
			if err != nil {
				return err
			}
		}

		err = jw.Close()
		if err != nil {
			return err
		}

		err = tx.Bucket([]byte(BlocksBucket)).Put([]byte(LastBlock), []byte(meta.Hash.String()))
		if err != nil {
			return err
		}

		a.blockSize = 0

		return nil
	})
	if err == errNoNewData {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "first stage")
	}

	meta.Status = PREPROCESSED

	// TODO: push to masterchain
	// meta.Status = FIXED

	// upload here
	err = a.upload(&meta, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "upload")
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrap(err, "marshal meta")
	}

	err = a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		err := b.Put([]byte(meta.Hash.String()), data)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "last stage")
	}

	return nil
}

func (a *Archiver) uploadToMinio(meta *BlockMeta, buf []byte) error {
	if a.minio == nil {
		return nil
	}

	k := a.minio.Prefix + meta.Hash.String() + ".tar"

	minioMeta := map[string]string{
		"block_hash":      meta.Hash.String(),
		"block_prev_hash": meta.PrevHash.String(),
	}

	tries := 0
retry:
	tries++
	// TODO(nik): what should we do with ACL and StorageClass?
	_, err := a.minio.Client.PutObject(a.minio.Bucket, k, bytes.NewReader(buf), int64(len(buf)), minio.PutObjectOptions{
		UserMetadata: minioMeta,
	})
	if err != nil {
		log.Printf("upload error: %v", err)
		if tries < 3 {
			goto retry
		}
		return err
	}
	// TODO(nik): copy to last.tar
	dst, err := minio.NewDestinationInfo(a.minio.Bucket, a.minio.Prefix+"last.tar", nil, map[string]string{
		"block_hash":      meta.Hash.String(),
		"block_prev_hash": meta.PrevHash.String(),
	})
	if err != nil {
		return errors.Wrap(err, "make last link")
	}
	src := minio.NewSourceInfo(a.minio.Bucket, k, nil)

	err = a.minio.Client.CopyObject(dst, src)
	if err != nil {
		return errors.Wrap(err, "make last link")
	}

	meta.StorageBucket = a.minio.Bucket
	meta.StoragePath = k
	meta.Status = UPLOADED

	return nil
}

func (a *Archiver) Push(ctx context.Context, txns []pt.Txn) error {
	for i := range txns {
		pt.GetHashDefault(&txns[i])
	}

	ts := txnsToProto(txns)
	err := a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TxnsBucket))
		for i, txn := range txns {
			k := txnKey(ZeroBlockHash, TxnsDir, pt.NewTxnID(txn.Sender, txn.ID))
			v, err := json.Marshal(ts[i])
			if err != nil {
				return err
			}
			err = b.Put(k, v)
			if err != nil {
				return err
			}
		}
		a.blockSize += len(txns)
		if a.blockSize > a.rotateSize {
			a.triggerRotation()
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) PushSettings(ctx context.Context, sett *pt.Settings) error {
	pt.GetSettingsHashDefault(sett)

	err := a.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TxnsBucket))
		k := txnKey(ZeroBlockHash, SettingsDir, pt.NewTxnID(sett.Account, sett.ID))
		v, err := json.Marshal(sett)
		if err != nil {
			return err
		}
		err = b.Put(k, v)
		if err != nil {
			return err
		}
		a.blockSize++
		if a.blockSize > a.rotateSize {
			a.triggerRotation()
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) CheckTxn(ctx context.Context, tid pt.TxnID) (*BlockMeta, error) {
	var meta BlockMeta

	err := a.db.View(func(tx *bolt.Tx) error {
		block := tx.Bucket([]byte(IndexBucket)).Get(idxKey(TxnsDir, tid))
		if block == nil {
			data := tx.Bucket([]byte(TxnsBucket)).Get(txnKey(ZeroBlockHash, TxnsDir, tid))
			if data != nil {
				meta.Status = UNPROCESSED
			} else {
				meta.Status = NOTFOUND
			}
			return nil
		}
		data := tx.Bucket([]byte(BlocksBucket)).Get(block)
		err := json.Unmarshal(data, &meta)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (a *Archiver) CheckSettings(ctx context.Context, tid pt.SettingsID) (*BlockMeta, error) {
	var meta BlockMeta

	sid := pt.TxnID(tid)

	err := a.db.View(func(tx *bolt.Tx) error {
		block := tx.Bucket([]byte(IndexBucket)).Get(idxKey(SettingsDir, sid))
		if block == nil {
			data := tx.Bucket([]byte(TxnsBucket)).Get(txnKey(ZeroBlockHash, SettingsDir, sid))
			if data != nil {
				meta.Status = UNPROCESSED
			} else {
				meta.Status = NOTFOUND
			}
			return nil
		}
		data := tx.Bucket([]byte(BlocksBucket)).Get(block)
		err := json.Unmarshal(data, &meta)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (a *Archiver) LastBlock(ctx context.Context) (*BlockMeta, error) {
	var meta *BlockMeta
	err := a.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		hash := b.Get([]byte(LastBlock))
		if hash == nil {
			return nil
		}
		data := b.Get(hash)
		if data == nil {
			return nil
		}

		meta = &BlockMeta{}
		err := json.Unmarshal(data, meta)
		return err
	})
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (a *Archiver) Block(ctx context.Context, hash BlockHash) (*BlockMeta, error) {
	var meta *BlockMeta
	err := a.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		data := b.Get([]byte(hash.String()))
		if data == nil {
			return nil
		}

		meta = &BlockMeta{}
		err := json.Unmarshal(data, meta)
		return err
	})
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (a *Archiver) triggerRotation() {
	select {
	case a.rotateC <- struct{}{}:
	default:
	}
}

func idxKey(dir string, tid pt.TxnID) []byte {
	k := dir + "/" + fmt.Sprintf("%016x_%08x", uint64(tid.AccID), uint64(tid.ID))
	return []byte(k)
}

func txnKey(h BlockHash, dir string, tid pt.TxnID) []byte {
	k := h.String() + "/" + dir + "/" + fmt.Sprintf("%016x_%08x", uint64(tid.AccID), uint64(tid.ID))
	return []byte(k)
}

func parseTxnKey(k []byte) (bh BlockHash, tp string, tid pt.TxnID) {
	s := strings.Split(string(k), "/")
	_, _ = hex.Decode(bh[:], []byte(s[0]))
	tp = s[1]
	_, _ = fmt.Sscanf(s[2], "%x_%x", &tid.AccID, &tid.ID)
	return
}

func NewBlockHash(h []byte) BlockHash {
	var r BlockHash
	copy(r[:], h)
	return r
}

// NewBlockHashFromString is not safe, it can be used only for tests or constants
func NewBlockHashFromString(s string) BlockHash {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return NewBlockHash(data)
}

func (h BlockHash) String() string {
	return hex.EncodeToString(h[:])
}

func (s Status) Bytes() []byte {
	return []byte{byte(s)}
}

func (h BlockHash) MarshalJSON() ([]byte, error) {
	str := hex.EncodeToString(h[:])
	return []byte(fmt.Sprintf("%q", str)), nil
}

func (h *BlockHash) UnmarshalJSON(data []byte) error {
	if data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("format error: %s", data)
	}
	_, err := hex.Decode((*h)[:], data[1:len(data)-1])
	return err
}

func txnsToProto(in []pt.Txn) []*archiverpb.Txn {
	txns := make([]*archiverpb.Txn, len(in))
	for i := range in {
		t := in[i]
		txns[i] = &archiverpb.Txn{
			Id:         fmt.Sprintf("%d", t.ID),
			Sender:     fmt.Sprintf("%d", t.Sender),
			Receiver:   fmt.Sprintf("%d", t.Receiver),
			Amount:     fmt.Sprintf("%d", t.Amount),
			Balance:    fmt.Sprintf("%d", t.Balance),
			SpentBy:    fmt.Sprintf("%d", t.SpentBy),
			SettingsId: fmt.Sprintf("%d", t.SettingsID),
			PrevHash:   hex.EncodeToString(t.PrevHash[:]),
			Hash:       hex.EncodeToString(t.Hash[:]),
			Sign:       hex.EncodeToString(t.Sign[:]),
		}
	}
	return txns
}

func settingsToProto(t *pt.Settings) *archiverpb.Settings {
	return &archiverpb.Settings{
		Id:                 fmt.Sprintf("%d", t.ID),
		Account:            fmt.Sprintf("%d", t.Account),
		PrevHash:           hex.EncodeToString(t.PrevHash[:]),
		Hash:               hex.EncodeToString(t.Hash[:]),
		DataHash:           hex.EncodeToString(t.DataHash[:]),
		PublicKey:          hex.EncodeToString(t.PublicKey[:]),
		Sign:               hex.EncodeToString(t.Sign[:]),
		VerifyTransferSign: t.VerifyTransferSign,
	}
}
