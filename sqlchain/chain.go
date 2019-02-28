package sqlchain

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
	"github.com/qiwitech/qdp/pt"
)

type DB struct {
	c *sql.DB
}

func New(c *sql.DB) (*DB, error) {
	_, err := c.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS txns (
		id          BIGINT UNSIGNED,
		sender      BIGINT UNSIGNED,
		receiver    BIGINT UNSIGNED,
		amount      BIGINT,
		balance     BIGINT,
		settings_id BIGINT UNSIGNED,
		spent_by    BIGINT UNSIGNED,
		prev_hash   VARCHAR(64),
		hash        VARCHAR(64),
		sign        VARCHAR(250),
		UNIQUE KEY (sender, id)
	)`))
	if err != nil {
		return nil, err
	}
	_, err = c.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS sett (
		id          BIGINT UNSIGNED,
		account     BIGINT UNSIGNED,
		verify_transfer_sign BOOL,
		prev_hash   VARCHAR(64),
		data_hash   VARCHAR(64),
		hash        VARCHAR(64),
		sign        VARCHAR(250),
		public_key  VARCHAR(250),
		UNIQUE KEY (account, id)
	)`))
	if err != nil {
		return nil, err
	}
	return &DB{c: c}, nil
}

func (d *DB) Push(ctx context.Context, txns []pt.Txn) (err error) {
	if len(txns) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO txns (id, sender, receiver, amount, balance, settings_id, spent_by, prev_hash, sign, hash) VALUES `)
	for i, txn := range txns {
		if i != 0 {
			b.WriteString(", ")
		}
		if txn.Hash == pt.ZeroHash {
			txn.Hash = pt.GetHashDefault(&txn)
		}
		b.WriteString(fmt.Sprintf("(%d, %d, %d, %d, %d, %d, %d, %q, %q, %q)", txn.ID, txn.Sender, txn.Receiver, txn.Amount, txn.Balance, txn.SettingsID, txn.SpentBy,
			hex.EncodeToString(txn.PrevHash[:]), hex.EncodeToString(txn.Sign[:]), hex.EncodeToString(txn.Hash[:])))
	}

	b.WriteString(` ON DUPLICATE KEY UPDATE spent_by = VALUES(spent_by)`)

	_, err = d.c.Exec(b.String())

	return err
}

func (d *DB) PushSettings(ctx context.Context, sett *pt.Settings) (err error) {
	if sett == nil {
		return nil
	}
	if sett.Hash == pt.ZeroHash {
		sett.Hash = pt.GetSettingsHashDefault(sett)
	}
	_, err = d.c.Exec(fmt.Sprintf(`INSERT INTO sett (id, account, verify_transfer_sign, prev_hash, data_hash, sign, public_key, hash)
						VALUES (%d, %d, %v, %q, %q, %q, %q, %q)`, sett.ID, sett.Account, sett.VerifyTransferSign,
		hex.EncodeToString(sett.PrevHash[:]),
		hex.EncodeToString(sett.DataHash[:]),
		hex.EncodeToString(sett.Sign[:]),
		hex.EncodeToString(sett.PublicKey[:]),
		hex.EncodeToString(sett.Hash[:]),
	))
	return err
}

func (d *DB) Fetch(ctx context.Context, req *plutodbpb.FetchRequest) (resp *plutodbpb.FetchResponse, err error) {
	var txns []*chainpb.Txn
	add := func(rows *sql.Rows) error {
		for rows.Next() {
			var txn chainpb.Txn
			var ph, sign string
			err := rows.Scan(&txn.ID, &txn.Sender, &txn.Receiver, &txn.Amount, &txn.Balance, &txn.SettingsId, &txn.SpentBy, &ph, &sign)
			if err != nil {
				return err
			}
			if err = decodeHex(&txn.PrevHash, ph); err != nil {
				return err
			}
			if err = decodeHex(&txn.Sign, sign); err != nil {
				return err
			}
			txns = append(txns, &txn)
		}
		return rows.Close()
	}

	q := fmt.Sprintf(`SELECT id, sender, receiver, amount, balance, settings_id, spent_by, prev_hash, sign FROM txns WHERE sender = %d ORDER BY id DESC LIMIT %d`, req.Account, req.Limit)
	rows, err := d.c.Query(q)
	if err != nil {
		return nil, err
	}
	if err = add(rows); err != nil {
		return nil, err
	}

	q = fmt.Sprintf(`SELECT id, sender, receiver, amount, balance, settings_id, spent_by, prev_hash, sign FROM txns WHERE receiver = %d AND id = 0`, req.Account)
	rows, err = d.c.Query(q)
	if err != nil {
		return nil, err
	}
	if err = add(rows); err != nil {
		return nil, err
	}

	var sett *chainpb.Settings
	rows, err = d.c.Query(`SELECT id, account, verify_transfer_sign, prev_hash, data_hash, sign, public_key FROM sett WHERE account = ? ORDER BY id DESC LIMIT 1`, req.Account)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		sett = new(chainpb.Settings)
		var ph, dh, sign, key string
		err = rows.Scan(&sett.ID, &sett.Account, &sett.VerifyTransferSign, &ph, &dh, &sign, &key)
		if err != nil {
			return nil, err
		}
		if err = decodeHex(&sett.PrevHash, ph); err != nil {
			return nil, err
		}
		if err = decodeHex(&sett.DataHash, dh); err != nil {
			return nil, err
		}
		if err = decodeHex(&sett.Sign, sign); err != nil {
			return nil, err
		}
		if err = decodeHex(&sett.PublicKey, key); err != nil {
			return nil, err
		}
	}

	return &plutodbpb.FetchResponse{Status: &plutodbpb.Status{}, Txns: txns, Settings: sett}, nil
}

func (d *DB) GetHistory(ctx context.Context, req *plutodbpb.GetHistoryRequest) (resp *plutodbpb.GetHistoryResponse, _ error) {
	resp = &plutodbpb.GetHistoryResponse{
		Status: &plutodbpb.Status{Code: 0},
	}
	var txns []*chainpb.Txn

	out := false
	id := uint64(0)
loop:
	for {
		var err error
		var rows *sql.Rows
		if out {
			if id == 0 {
				id--
			}
			rows, err = d.c.Query(`SELECT id, sender, receiver, amount, balance, settings_id, spent_by, prev_hash, sign FROM txns WHERE sender = ? AND id < ? ORDER BY id DESC LIMIT 1`, req.Account, id)
		} else {
			rows, err = d.c.Query(`SELECT id, sender, receiver, amount, balance, settings_id, spent_by, prev_hash, sign FROM txns WHERE receiver = ? AND spent_by = ?`, req.Account, id)
		}
		if err != nil {
			return nil, err
		}
		var added bool
		for rows.Next() {
			var txn chainpb.Txn
			var ph, sign string
			err = rows.Scan(&txn.ID, &txn.Sender, &txn.Receiver, &txn.Amount, &txn.Balance, &txn.SettingsId, &txn.SpentBy, &ph, &sign)
			if err != nil {
				return nil, err
			}
			if err = decodeHex(&txn.PrevHash, ph); err != nil {
				return nil, err
			}
			if err = decodeHex(&txn.Sign, sign); err != nil {
				return nil, err
			}
			txns = append(txns, &txn)
			if len(txns) == int(req.Limit) {
				rows.Close()
				break loop
			}
			added = true
			if out {
				id = txn.ID
			}
		}
		if out && !added {
			break loop
		}
		out = !out
	}
	resp.Txns = txns
	return resp, nil
}

func (d *DB) GetTxnMulti(ctx context.Context, req *plutodbpb.GetTxnMultiRequest) (*plutodbpb.GetTxnMultiResponse, error) {
	resp := &plutodbpb.GetTxnMultiResponse{
		Status: &plutodbpb.Status{},
	}

	txns := make([]*chainpb.Txn, len(req.IDs))
	for i, id := range req.IDs {
		row := d.c.QueryRow(`SELECT id, sender, receiver, amount, balance, settings_id, spent_by, prev_hash, sign FROM txns WHERE sender = ? AND id < ? ORDER BY id DESC LIMIT 1`, id.Account, id.ID)
		var txn chainpb.Txn
		var ph, sign string
		err := row.Scan(&txn.ID, &txn.Sender, &txn.Receiver, &txn.Amount, &txn.Balance, &txn.SettingsId, &txn.SpentBy, &ph, &sign)
		if err != nil {
			return nil, err
		}
		if err = decodeHex(&txn.PrevHash, ph); err != nil {
			return nil, err
		}
		if err = decodeHex(&txn.Sign, sign); err != nil {
			return nil, err
		}
		txns[i] = &txn
	}

	resp.Txns = txns

	return resp, nil
}

func decodeHex(dst *[]byte, s string) error {
	n := hex.DecodedLen(len(s))
	*dst = make([]byte, n)
	_, err := hex.Decode(*dst, []byte(s))
	return err
}
