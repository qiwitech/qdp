/*
Package chain is an local accounts chains cache.
It's intended to process new transfers fast not fetching data from database.

General structure is a set of independent accounts.
Each account is a list of output transactions and their inputs.

Example of chain for account 100. Transactions goes from last to previous. We have 3 output transactions and 2 unspent inputs. Current balance is 650.
	    <- id:4 from:102 to:100 amount:20 spent_by:0
	    <- id:3 from:107 to:100 amount:30 spent_by:0
	-> id:3 from:100 to:103 amount:200 balance:600
	-> id:2 from:100 to:102 amount:530 balance:800
	    <- id:14 from:10 to:100 amount:30 spent_by:2
	    <- id:8 from:105 to:100 amount:300 spent_by:2
	-> id:1 from:100 to:101 amount:100 balance:900
	    <- id:12 from:10 to:100 amount:1000 spent_by:1
Input transactions are indented to highlight that they are:
	1. Belongs to another chain by ids increment and prev_hash links
	2. Taken into account in the next output transaction of current account.
*/
package chain

import (
	"sync"

	"github.com/qiwitech/qdp/pt"
)

// Chain is an inmemory cache of accounts chains.
type Chain struct {
	mu      sync.Mutex
	list    map[pt.AccID]*skiplist
	unspent map[pt.AccID]map[pt.TxnID]*pt.Txn
}

type chainElement struct {
	Txn    *pt.Txn              // output
	Inputs map[pt.TxnID]*pt.Txn // inputs
}

func NewChain() *Chain {
	return &Chain{
		list:    make(map[pt.AccID]*skiplist),
		unspent: make(map[pt.AccID]map[pt.TxnID]*pt.Txn),
	}
}

func (c *Chain) GetBalance(accID pt.AccID) int64 {
	defer c.mu.Unlock()
	c.mu.Lock()

	var balance int64

	last := c.getLastTxn(accID)
	if last != nil {
		balance += last.Balance
	}

	txns, ok := c.unspent[accID]
	if !ok {
		return balance
	}

	for _, txn := range txns {
		balance += txn.Amount
	}

	return balance
}

func (c *Chain) ListUnspentTxns(accID pt.AccID) []pt.Txn {
	defer c.mu.Unlock()
	c.mu.Lock()

	tmap, ok := c.unspent[accID]
	if !ok || len(tmap) == 0 {
		return nil
	}

	txns := make([]pt.Txn, len(tmap))
	i := 0
	for _, txn := range tmap {
		// copy value
		txns[i] = *txn
		i++
	}
	return txns
}

// PutTo puts transactions to account chain.
func (c *Chain) PutTo(accID pt.AccID, txns []pt.Txn) {
	defer c.mu.Unlock()
	c.mu.Lock()

	list, ok := c.list[accID]
	// create chain, if not exists
	if !ok {
		list = newSkipList()
		c.list[accID] = list
	}

	// cut chain
	if e := list.Front(); e != nil {
		var id pt.ID
		if e.Value.Txn != nil {
			id = e.Value.Txn.ID
		} else {
			for _, txn := range e.Value.Inputs {
				id = txn.SpentBy
				break
			}
		}

		if id > 3 {
			list.RemoveAfter(id - 3)
		}
	}

	for i, txn := range txns {
		if txn.ID == 0 {
			panic("chain put: zero txn id")
		}
		if txn.Sender != accID && txn.Receiver != accID {
			panic("put to wrong chain")
		}

		if accID == txn.Sender { // it's output txn
			e := list.GetOrInsert(txn.ID)
			if e == nil {
				panic("can't insert element")
			}
			if e.Value.Txn == nil {
				e.Value.Txn = &txns[i]
			} else if e.Value.Txn.SpentBy == 0 && txn.SpentBy != 0 {
				e.Value.Txn.SpentBy = txn.SpentBy
			}
		}

		if accID != txn.Receiver {
			continue
		}
		// it's input txn

		receiverTxnID := pt.NewTxnID(txn.Sender, txn.ID)

		if txn.SpentBy == 0 { // it's unspent
			_, ok := c.unspent[txn.Receiver]
			if !ok {
				unspent := make(map[pt.TxnID]*pt.Txn)
				c.unspent[txn.Receiver] = unspent
			}
			c.unspent[txn.Receiver][receiverTxnID] = &txns[i]
			continue
		}

		// it's spent input
		e := list.GetOrInsert(txn.SpentBy)
		if e == nil {
			panic("can't insert element")
		}
		if e.Value.Inputs == nil {
			e.Value.Inputs = make(map[pt.TxnID]*pt.Txn)
		}
		e.Value.Inputs[receiverTxnID] = &txn

		// delete txn from unspent list
		delete(c.unspent[txn.Receiver], receiverTxnID)
	}
}

func (c *Chain) GetLastTxn(accID pt.AccID) *pt.Txn {
	defer c.mu.Unlock()
	c.mu.Lock()

	return c.getLastTxn(accID)
}

func (c *Chain) getLastTxn(accID pt.AccID) *pt.Txn {
	if list, ok := c.list[accID]; ok {
		if e := list.Front(); e != nil {
			return e.Value.Txn
		}
	}

	return nil
}

func (c *Chain) GetLastHash(accID pt.AccID) pt.Hash {
	if txn := c.GetLastTxn(accID); txn != nil {
		if txn.Hash == pt.ZeroHash {
			pt.GetHashDefault(txn)
		}
		return txn.Hash
	}
	return pt.ZeroHash
}

// GetLastNTxns returns at most n transactions (as many as we have) from last one to previous
func (c *Chain) GetLastNTxns(accID pt.AccID, n int) []pt.Txn {
	defer c.mu.Unlock()
	c.mu.Lock()

	list, ok := c.list[accID]
	if !ok {
		return nil
	}

	res := make([]pt.Txn, n)

	i := 0
	for e := list.Front(); e != nil && i < n; e = e.Next() {
		if e.Value.Txn == nil {
			break
		}
		res[i] = *e.Value.Txn
		i++
	}

	res = res[:i]

	return res
}

func (c *Chain) Reset(accID pt.AccID) {
	defer c.mu.Unlock()
	c.mu.Lock()

	delete(c.list, accID)
	delete(c.unspent, accID)
}
