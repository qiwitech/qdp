package archiver

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qiwitech/qdp/pt"
)

func Handler(a *Archiver) http.Handler {
	e := gin.Default()

	e.GET("/check/txn/:id", func(c *gin.Context) {
		id := c.Param("id")
		txnid, err := parseTxnID(id)
		if !checkError(c, err, "txnid parse") {
			return
		}

		b, err := a.CheckTxn(context.TODO(), txnid)
		if !checkError(c, err, "check") {
			return
		}

		responseBlock(c, "block", b)
	})
	e.GET("/check/settings/:id", func(c *gin.Context) {
		id := c.Param("id")
		txnid, err := parseTxnID(id)
		if !checkError(c, err, "txnid parse") {
			return
		}

		b, err := a.CheckSettings(context.TODO(), pt.SettingsID(txnid))
		if !checkError(c, err, "check") {
			return
		}

		responseBlock(c, "block", b)
	})
	e.GET("/blocks/last", func(c *gin.Context) {
		meta, err := a.LastBlock(context.TODO())
		if !checkError(c, err, "check") {
			return
		}

		responseBlock(c, "block", meta)
	})
	e.GET("/blocks/hash/:hash", func(c *gin.Context) {
		hs := c.Param("hash")
		h := NewBlockHashFromString(hs)

		meta, err := a.Block(context.TODO(), h)
		if !checkError(c, err, "check") {
			return
		}

		responseBlock(c, "block", meta)
	})
	e.GET("/rotate", func(c *gin.Context) {
		err := a.Rotate()
		if !checkError(c, err, "rotate") {
			return
		}

		response(c, "status", "ok")
	})

	return e
}

func parseTxnID(s string) (pt.TxnID, error) {
	var t pt.TxnID
	n, err := fmt.Sscanf(s, "%d_%d", &t.AccID, &t.ID)
	if err != nil {
		log.Printf("parse (%s) error: %v %v", s, n, err)
	}
	return t, err
}

func checkError(c *gin.Context, err error, msg string) bool {
	if err == nil {
		return true
	}
	c.JSON(http.StatusOK, gin.H{"error": msg + ": " + err.Error()})
	return false
}

func response(c *gin.Context, key string, v interface{}) {
	c.JSON(http.StatusOK, gin.H{key: v})
}

func responseBlock(c *gin.Context, key string, meta *BlockMeta) {
	type aux struct {
		*BlockMeta
		Status string
	}

	var x aux
	x.BlockMeta = meta
	if meta != nil {
		x.Status = meta.Status.String()
	} else {
		x.Status = NOTFOUND.String()
	}

	response(c, key, x)
}
