package archiver

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/pt"
)

func TestPush(t *testing.T) {
	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 1, ID: 2, Receiver: 3},
		{Sender: 1, ID: 3, Receiver: 3},
		{Sender: 1, ID: 4, Receiver: 3},
	})
	assert.NoError(t, err)

	m, err := a.CheckTxn(context.TODO(), pt.TxnID{AccID: 1, ID: 1})
	assert.NoError(t, err)
	assert.Equal(t, NOTFOUND, m.Status)

	m, err = a.CheckTxn(context.TODO(), pt.TxnID{AccID: 1, ID: 2})
	assert.NoError(t, err)
	assert.Equal(t, UNPROCESSED, m.Status)

	m, err = a.CheckTxn(context.TODO(), pt.TxnID{AccID: 1, ID: 3})
	assert.NoError(t, err)
	assert.Equal(t, UNPROCESSED, m.Status)
}

func TestRotate(t *testing.T) {
	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 10, ID: 2, Receiver: 3},
		{Sender: 10, ID: 3, Receiver: 3},
		{Sender: 10, ID: 4, Receiver: 3},
	})
	assert.NoError(t, err)

	err = a.Rotate()
	assert.NoError(t, err)

	s, err := a.CheckTxn(context.TODO(), pt.TxnID{AccID: 10, ID: 1})
	assert.NoError(t, err)
	assert.Equal(t, NOTFOUND, s.Status)

	s, err = a.CheckTxn(context.TODO(), pt.TxnID{AccID: 10, ID: 2})
	assert.NoError(t, err)
	assert.Equal(t, PREPROCESSED, s.Status)

	s, err = a.CheckTxn(context.TODO(), pt.TxnID{AccID: 10, ID: 3})
	assert.NoError(t, err)
	assert.Equal(t, PREPROCESSED, s.Status)
}

func TestBlock(t *testing.T) {
	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 10, ID: 2, Receiver: 3},
		{Sender: 10, ID: 3, Receiver: 3},
		{Sender: 10, ID: 4, Receiver: 3},
	})
	assert.NoError(t, err)

	b, err := a.LastBlock(context.TODO())
	assert.NoError(t, err)
	assert.Nil(t, b)

	now = func() int64 {
		return 1501265852621448683
	}

	err = a.Rotate()
	assert.NoError(t, err)

	s, err := a.CheckTxn(context.TODO(), pt.TxnID{AccID: 10, ID: 1})
	assert.NoError(t, err)
	assert.Equal(t, NOTFOUND, s.Status)

	s, err = a.CheckTxn(context.TODO(), pt.TxnID{AccID: 10, ID: 2})
	assert.NoError(t, err)
	assert.Equal(t, PREPROCESSED, s.Status)

	s, err = a.CheckTxn(context.TODO(), pt.TxnID{AccID: 10, ID: 3})
	assert.NoError(t, err)
	assert.Equal(t, PREPROCESSED, s.Status)

	b, err = a.LastBlock(context.TODO())
	assert.NoError(t, err)
	exp := &BlockMeta{
		Hash:     NewBlockHashFromString("b4acba88e2f63f38e53134e1bcb0238ffe4e2670c4359faa5f95c9f4fd31cf0d"),
		Time:     timestr(1501265852621448683),
		Status:   PREPROCESSED,
		HashType: HashSHA256,
	}
	assert.Equal(t, exp, b)
}

func TestBlockPrevHash(t *testing.T) {
	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 10, ID: 2, Receiver: 3},
		{Sender: 10, ID: 3, Receiver: 3},
		{Sender: 10, ID: 4, Receiver: 3},
	})
	assert.NoError(t, err)

	now = func() int64 {
		return 1501111111111111111
	}

	err = a.Rotate()
	assert.NoError(t, err)

	// last block
	b, err := a.LastBlock(context.TODO())
	assert.NoError(t, err)
	exp := &BlockMeta{
		Hash:     NewBlockHashFromString("b4acba88e2f63f38e53134e1bcb0238ffe4e2670c4359faa5f95c9f4fd31cf0d"),
		Time:     timestr(1501111111111111111),
		Status:   PREPROCESSED,
		HashType: HashSHA256,
	}
	assert.Equal(t, exp, b)

	// more transactions
	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 10, ID: 5, Receiver: 3},
		{Sender: 10, ID: 6, Receiver: 3},
	})
	assert.NoError(t, err)

	now = func() int64 {
		return 1502222222222222222
	}

	err = a.Rotate()
	assert.NoError(t, err)

	// new last block
	b, err = a.LastBlock(context.TODO())
	assert.NoError(t, err)
	exp = &BlockMeta{
		Hash:     NewBlockHashFromString("38aa66f7bab3a00ac4332754ff98baa73df0a0e71a1ce69ffa35e026cfffe055"),
		PrevHash: NewBlockHashFromString("b4acba88e2f63f38e53134e1bcb0238ffe4e2670c4359faa5f95c9f4fd31cf0d"),
		Time:     timestr(1502222222222222222),
		Status:   PREPROCESSED,
		HashType: HashSHA256,
	}
	assert.Equal(t, exp, b)

	if t.Failed() {
		return
	}

	// first block
	b, err = a.Block(context.TODO(), b.PrevHash)
	assert.NoError(t, err)
	exp = &BlockMeta{
		Hash:     NewBlockHashFromString("b4acba88e2f63f38e53134e1bcb0238ffe4e2670c4359faa5f95c9f4fd31cf0d"),
		Time:     timestr(1501111111111111111),
		Status:   PREPROCESSED,
		HashType: HashSHA256,
	}
	assert.Equal(t, exp, b)
}

func TestBlockHash(t *testing.T) {
	h := NewBlockHash([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	data, err := h.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, []byte(`"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"`), data)

	var dec BlockHash
	err = dec.UnmarshalJSON(data)
	assert.NoError(t, err)

	assert.Equal(t, h, dec)
}

func TestBlockMeta(t *testing.T) {
	m := BlockMeta{
		Hash:     NewBlockHashFromString("0102030405060708090a"),
		PrevHash: NewBlockHashFromString("1112131415161718191a"),
		Time:     timestr(1000),
		Status:   UPLOADED,
	}

	data, err := json.Marshal(&m)
	assert.NoError(t, err)

	t.Logf("block: %s", data)

	var dec BlockMeta
	err = json.Unmarshal(data, &dec)
	assert.NoError(t, err)

	assert.Equal(t, m, dec)
}

func TestSettingsBlock(t *testing.T) {
	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.PushSettings(context.TODO(), &pt.Settings{
		Account:  16,
		ID:       1,
		DataHash: pt.HashFromString("1f00e696bd81ab78fe529fdf9bb597147b2b951b536c64a9d73e4c88d828df54"),
	})
	assert.NoError(t, err)

	now = func() int64 {
		return 1501265853000008683
	}

	err = a.Rotate()
	assert.NoError(t, err)

	s, err := a.CheckSettings(context.TODO(), pt.SettingsID{AccID: 16, ID: 1})
	assert.NoError(t, err)
	assert.Equal(t, PREPROCESSED, s.Status)

	b, err := a.LastBlock(context.TODO())
	assert.NoError(t, err)
	exp := &BlockMeta{
		Hash:     NewBlockHashFromString("637bd6d95c84e0d62721e085043d7ede560f6cb8a829d8a726d870bed894a777"),
		Time:     timestr(1501265853000008683),
		Status:   PREPROCESSED,
		HashType: HashSHA256,
	}
	assert.Equal(t, exp, b)
}

func TestTar(t *testing.T) {
	t.Skip("debug")

	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 10, ID: 2, Receiver: 3},
		{Sender: 10, ID: 3, Receiver: 3},
		{Sender: 10, ID: 4, Receiver: 3},
	})
	assert.NoError(t, err)

	err = a.PushSettings(context.TODO(), &pt.Settings{
		Account:  16,
		ID:       1,
		DataHash: pt.HashFromString("1f00e696bd81ab78fe529fdf9bb597147b2b951b536c64a9d73e4c88d828df54"),
	})
	assert.NoError(t, err)

	if t.Failed() {
		return
	}

	now = func() int64 {
		return 1501265852621448683
	}

	// check tar file
	a.upload = func(m *BlockMeta, data []byte) error {
		tr := tar.NewReader(bytes.NewReader(data))
		buf := make([]byte, 1000)
		hdr, err := tr.Next()
		if err != nil {
			return err
		}
		n, err := tr.Read(buf)
		if err != nil {
			return err
		}
		assert.Equal(t, m.Hash.String()+"/metadata.json", hdr.Name)
		assert.Equal(t, []byte(`{"hash":"989c3605d835a170c38b058533ac6f2aa36fe26acd448c48e028d1c0d1437b8c","prev_hash":"0000000000000000000000000000000000000000000000000000000000000000","time":"2017-07-28T21:17:32+03:00","status":0,"hash_type":"sha256"}`), buf[:n])
		h := sha256.New()
		for {
			_, err = tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			n, err = tr.Read(buf)
			if err != nil {
				return err
			}

			_, _ = h.Write(buf[:n])
		}

		sum := h.Sum(nil)
		bh := NewBlockHash(sum)

		assert.Equal(t, m.Hash, bh)

		return nil
	}

	err = a.Rotate()
	assert.NoError(t, err)
}

func TestSaveTar(t *testing.T) {
	t.Skip()

	db, del := createBolt(t)
	defer del()

	a, err := New(db, 100)
	assert.NoError(t, err)

	err = a.Push(context.TODO(), []pt.Txn{
		{Sender: 10, ID: 2, Receiver: 3},
		{Sender: 10, ID: 3, Receiver: 3},
		{Sender: 10, ID: 4, Receiver: 3},
	})
	assert.NoError(t, err)

	err = a.PushSettings(context.TODO(), &pt.Settings{
		Account:  16,
		ID:       1,
		DataHash: pt.HashFromString("1f00e696bd81ab78fe529fdf9bb597147b2b951b536c64a9d73e4c88d828df54"),
	})
	assert.NoError(t, err)

	if t.Failed() {
		return
	}

	now = func() int64 {
		return 1501265852621448683
	}

	// write tar to disk
	a.upload = func(m *BlockMeta, data []byte) error {
		dir := os.TempDir()
		stat, err := os.Stat(path.Join(dir, "ram"))
		if err == nil && stat.IsDir() {
			dir = path.Join(dir, "ram")
		}

		name := path.Join(dir, "archiver_test.tar")

		err = ioutil.WriteFile(name, data, 0660)

		t.Logf("tar archive: %v", name)

		return err
	}

	err = a.Rotate()
	assert.NoError(t, err)
}

func createBolt(t *testing.T) (*bolt.DB, func()) {
	file, err := ioutil.TempFile(os.TempDir(), "arch_test_")
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}

	db, err := bolt.Open(file.Name(), 0666, nil)
	if err != nil {
		panic(err)
	}

	return db, func() {
		if t.Failed() {
			t.Logf("db path: %v", file.Name())
			return
		}
		if err := os.Remove(file.Name()); err != nil {
			panic(err)
		}
	}
}
