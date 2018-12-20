// +build ignore

package chain

import (
	"bytes"
	"fmt"
)

func (c *Chain) Dump() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "`list` = %+v\n", `list`)
	for accID, list := range c.list {
		fmt.Fprintf(&buf, "accID = %+v\n", accID)
		for el := list.Front(); el != nil; el = el.Next() {
			fmt.Fprintf(&buf, "    el.Value() = %+v\n", el.Value)
		}
	}

	fmt.Fprintf(&buf, "`unspent` = %+v\n", `unspent`)
	for accID, tmap := range c.unspent {
		fmt.Fprintf(&buf, "  accID = %+v\n", accID)
		for txnid, txn := range tmap {
			fmt.Fprintf(&buf, "    txnid = %+v\n", txnid)
			fmt.Fprintf(&buf, "      txn = %+v\n", txn)
		}
	}

	return buf.String()
}
