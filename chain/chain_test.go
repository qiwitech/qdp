package chain

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/qiwitech/qdp/pt"
)

func TestPutTxn(t *testing.T) {
	c := NewChain()

	c.PutTo(20, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 0}})
	assert.Len(t, c.list, 1)
	assert.Contains(t, c.unspent, pt.AccID(20))
}

func TestSpentTxn(t *testing.T) {
	c := NewChain()

	c.PutTo(20, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 2}})
	assert.Len(t, c.list, 1)
	assert.Len(t, c.unspent, 0)
}

func TestAddInputs(t *testing.T) {
	c := NewChain()

	txn1 := pt.Txn{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 2}
	txn2 := pt.Txn{ID: 5, Sender: 30, Receiver: 20, Amount: 1000, SpentBy: 2}
	c.PutTo(10, []pt.Txn{txn1})
	c.PutTo(30, []pt.Txn{txn2})

	assert.Len(t, c.list, 2)
	assert.Len(t, c.unspent, 0)

	res1 := c.GetLastTxn(10)
	res2 := c.GetLastTxn(30)

	assert.Equal(t, txn1, *res1)
	assert.Equal(t, txn2, *res2)
}

func TestUpdateInputs(t *testing.T) {
	c := NewChain()

	c.PutTo(10, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 2}})
	c.PutTo(20, []pt.Txn{{ID: 2, Sender: 20, Receiver: 20, Amount: 1000, SpentBy: 2}})
	assert.Len(t, c.list, 2)
	assert.Len(t, c.unspent, 0)
}

func TestSpendTxnReceiverChain(t *testing.T) {
	c := NewChain()

	c.PutTo(20, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 0}})
	assert.Len(t, c.list, 1)
	assert.Contains(t, c.unspent, pt.AccID(20))

	c.PutTo(20, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 3}})
	assert.Len(t, c.list, 1)
	if assert.Contains(t, c.unspent, pt.AccID(20)) {
		assert.Empty(t, c.unspent[pt.AccID(20)])
	}
}

func TestSpendTxnSenderChain(t *testing.T) {
	c := NewChain()

	c.PutTo(10, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 0}})
	assert.Len(t, c.list, 1)
	assert.Empty(t, c.unspent)

	c.PutTo(10, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 3}})
	assert.Len(t, c.list, 1)
	assert.Empty(t, c.unspent)
}

func TestGetLastHash(t *testing.T) {
	c := NewChain()

	h := c.GetLastHash(10)
	assert.Equal(t, h, pt.ZeroHash)

	c.PutTo(10, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20, Amount: 1000, SpentBy: 0, Hash: pt.HashFromString("01020304")}})
	h = c.GetLastHash(10)
	assert.Equal(t, pt.HashFromString("01020304"), h)
}

func TestPutInvalidTxns(t *testing.T) {
	c := NewChain()

	assert.Panics(t, func() {
		c.PutTo(10, []pt.Txn{{ID: 0}})
	})
}

func TestPutToWrongChain(t *testing.T) {
	c := NewChain()

	assert.Panics(t, func() {
		c.PutTo(30, []pt.Txn{{ID: 1, Sender: 10, Receiver: 20}})
	})
}

func TestGetBalance(t *testing.T) {
	c := NewChain()

	c.PutTo(10, []pt.Txn{
		{ID: 1, Sender: 0, Receiver: 10, Amount: 1000, Balance: -1000, SpentBy: 1},
		{ID: 1, Sender: 10, Receiver: 20, Amount: 500, Balance: 500}, // 500
	})

	assert.Equal(t, int64(500), c.GetBalance(10))
}

func TestGetBalanceWithUnspentInputs(t *testing.T) {
	c := NewChain()

	c.PutTo(10, []pt.Txn{
		{ID: 1, Sender: 0, Receiver: 10, Amount: 1000, SpentBy: 1},
		{ID: 1, Sender: 1, Receiver: 10, Amount: 100},                // 100
		{ID: 1, Sender: 10, Receiver: 20, Amount: 500, Balance: 500}, // + 500
		{ID: 1, Sender: 5, Receiver: 10, Amount: 300},                // + 300 = 900
	})

	assert.Equal(t, int64(900), c.GetBalance(10))
}

func TestGetLastTxn(t *testing.T) {
	c := NewChain()

	txn := &pt.Txn{ID: 1, Sender: 10, Receiver: 20, Amount: 500, Balance: 500}
	c.PutTo(10, []pt.Txn{*txn})
	assert.Equal(t, txn, c.GetLastTxn(10))

	txn = &pt.Txn{ID: 2, Sender: 10, Receiver: 20, Amount: 500, Balance: 0}
	c.PutTo(10, []pt.Txn{*txn})
	assert.Equal(t, txn, c.GetLastTxn(10))
}

func TestGetLastNTxns(t *testing.T) {
	c := NewChain()

	assert.Nil(t, c.GetLastNTxns(10, 3))

	txns := []pt.Txn{
		{ID: 3, Sender: 10, Receiver: 20, Amount: 100, Balance: 400},
		{ID: 2, Sender: 10, Receiver: 20, Amount: 500, Balance: 500},
		// ID: 1, Sender: 10 isn't here
		{ID: 100, Sender: 5, Receiver: 10, Amount: 500, Balance: 99500, SpentBy: 1}, // input to ID: 1
	}

	c.PutTo(10, txns)
	assert.Equal(t, txns[0:2], c.GetLastNTxns(10, 3))
}

func TestMustGetLastOutputTxn(t *testing.T) {
	c := NewChain()

	// put output
	txn := &pt.Txn{ID: 2, Sender: 10, Receiver: 20, Amount: 500, Balance: 0}
	c.PutTo(10, []pt.Txn{*txn})
	assert.Equal(t, txn, c.GetLastTxn(10))

	// put input
	txn = &pt.Txn{ID: 1, Sender: 0, Receiver: 10, Amount: 500, Balance: -500}
	c.PutTo(10, []pt.Txn{*txn})
	assert.NotEqual(t, txn, c.GetLastTxn(10))
}

func BenchmarkPutMulti(b *testing.B) {
	c := NewChain()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	val, ok := quick.Value(reflect.TypeOf(pt.Txn{}), r)
	if !ok {
		panic("quick.Value failed")
	}
	txn := val.Interface().(pt.Txn)

	b.Logf("txn = %+v\n", txn)

	for i := 0; i < b.N; i++ {
		c.PutTo(txn.Receiver, []pt.Txn{txn})
	}
}

func BenchmarkGetBalance(b *testing.B) {
	c := NewChain()

	txns := []pt.Txn{}
	balance := int64(0)
	for i := 0; i < 100; i++ {
		balance -= 1000
		txns = append(txns, pt.Txn{ID: 1 + pt.ID(i), Sender: 0, Receiver: 10 + pt.AccID(i), Amount: 1000, Balance: balance})
	}
	c.PutTo(0, txns)

	for i := 0; i < b.N; i++ {
		c.GetBalance(0)
	}
}

func TestListUnspentTxns(t *testing.T) {
	c := NewChain()

	assert.Nil(t, c.ListUnspentTxns(0))

	c.PutTo(10, []pt.Txn{
		{ID: 1, Sender: 0, Receiver: 10, Amount: 1000, SpentBy: 1},
		{ID: 1, Sender: 1, Receiver: 10, Amount: 100},
		{ID: 1, Sender: 10, Receiver: 20, Amount: 500, Balance: 500},
		{ID: 1, Sender: 5, Receiver: 10, Amount: 300},
	})

	txns := c.ListUnspentTxns(10)

	assert.Len(t, txns, 2)
	assert.Contains(t, txns, pt.Txn{ID: 1, Sender: 5, Receiver: 10, Amount: 300})
	assert.Contains(t, txns, pt.Txn{ID: 1, Sender: 1, Receiver: 10, Amount: 100})
}
