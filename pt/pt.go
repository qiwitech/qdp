package pt

import (
	"context"

	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/btcsuite/btcutil/base58"

	"github.com/pkg/errors"
)

type (
	// AccID is an unique account id.
	AccID uint64
	// ID is an transaction or settings id. It goes from 1 and up for each account, separate for transactions and settings.
	ID uint64

	// TxnID is an unique ID for each transaction among all accounts.
	TxnID struct {
		AccID
		ID
	}

	// SettingsID is an unique ID for each settings among all accounts.
	SettingsID struct {
		AccID
		ID
	}

	// Txn is a single transaction between two accounts.
	// It's a central structure of the project.
	//
	// See chain package docs for information about how transactions are structured.
	Txn struct {
		// Account incremental id for output transactions.
		ID               ID
		Sender, Receiver AccID
		Amount           int64
		// Balance of the Sender account at the time just after this transaction has been processed.
		Balance int64

		// Account settings id at the moment of this transaction processing.
		SettingsID ID

		// Stores ID of Receiver output transaction when this transaction was taken into account.
		// Set by Receiver. 0 if not spent yet.
		// It's not used for transaction Hash calculation.
		SpentBy ID
		// Previous transaction hash of the same Sender.
		PrevHash Hash
		// Hash is an hash of corrent transaction.
		Hash Hash
		// Transfer Sign.
		// If more that one transactions were in the batch only first contains sign of the whole request
		Sign Sign
	}

	// Settings is an account settings.
	// Settings are form the similar chain as Txns are.
	Settings struct {
		ID                 ID
		Account            AccID
		PublicKey          PublicKey
		PrevHash, Hash     Hash
		VerifyTransferSign bool
		DataHash           Hash
		Sign               Sign
	}

	// TransferItem is an part of Transfer request.
	TransferItem struct {
		Receiver AccID
		Amount   int64
	}

	// Transfer is an request to make transfer from an account to one or many others.
	Transfer struct {
		Sender     AccID
		Batch      []*TransferItem // Receivers and amounts
		Sign       Sign            // Request sign. It will be kept at first transaction of the batch
		PrevHash   Hash            // Hash of last output transaction for Sender account
		SettingsID ID              // Current account settings ID for Sender account
	}

	// TransferResult is an result of transfer.
	TransferResult struct {
		TxnID      TxnID // TxnID of the first transaction in the batch
		Hash       Hash  // Hash of last transaction in the batch
		SettingsId ID    // Current settings ID
	}

	// SettingsResult is an result of settings change.
	SettingsResult struct {
		SettingsID SettingsID // Last Settings ID
		Hash       Hash       // Hash Settings Hash
	}
)

type ( // Services interfaces
	TransferProcessor interface {
		ProcessTransfer(ctx context.Context, t Transfer) (TransferResult, error)
		GetPrevHash(ctx context.Context, acc AccID) (Hash, error)
		GetBalance(ctx context.Context, acc AccID) (int64, error)
		SetPusher(Pusher)
		SetPreloader(Preloader)
		SetSettingsChain(SettingsChain)
	}

	SettingsChain interface {
		GetLastSettings(AccID) *Settings
		GetLastHash(AccID) Hash
		Put(*Settings)
		Reset(AccID)
	}

	SettingsProcessor interface {
		ProcessSettings(ctx context.Context, s *Settings) (SettingsResult, error)
		GetLastSettings(ctx context.Context, acc AccID) (*Settings, error)
	}

	// Chain is an local cache of last account transaftions.
	// It's used to process new transfers fast.
	Chain interface {
		ListUnspentTxns(accID AccID) []Txn
		PutTo(accID AccID, txns []Txn)
		GetLastHash(accID AccID) Hash
		GetBalance(accID AccID) int64
		GetLastTxn(accID AccID) *Txn
		GetLastNTxns(accID AccID, n int) []Txn
		Reset(AccID)
	}

	// BigChain is an global reliable storage for transactions and settings.
	// It's is the only point of truth. In case of possible inconsistency all local account data is erased and refetched from BigChain.
	BigChain interface {
		Fetch(ctx context.Context, accID AccID, limit int) ([]Txn, *Settings, error)
	}

	// Router is an cluster router.
	// It's used to find node which is responsible for processing transactions of given account.
	Router interface {
		// Returns host which is responsible for given key.
		// Key usually is an AccountID
		GetHostByKey(key string) string
		// All nodes hostnames of the cluster
		Nodes() []string
		// Sets cluster nodes
		SetNodes(nodes []string)
		// Checks if given node is our own, so we must process request not redirect it
		IsSelf(node string) bool
	}

	// Pusher is an general transfer pusher mechanism.
	// It's used to push data to global storage, receiver's node, third parties and so on.
	Pusher interface {
		Push(ctx context.Context, txns []Txn) error
	}

	// Settings pusher is the same thing as Pusher but for Settings.
	SettingsPusher interface {
		PushSettings(ctx context.Context, sett *Settings) error
	}

	// Preloader loads the most recent account data from reliable storage.
	// It dosn't fetch data if it was fetched before and wasn't Reset
	Preloader interface {
		Preload(context.Context, AccID) error
		Reset(context.Context, AccID)
	}
)

// Hash "nil" values for compare operations
var (
	ZeroHash Hash
	ZeroSign Sign
)

// Parse errors
var (
	ErrInvalidHash = errors.New("invalid hash string")
	ErrInvalidSign = errors.New("invalid sign string")
)

func (id AccID) String() string {
	return strconv.FormatUint(uint64(id), 10) // faster than fmt.Sprintf
}

func (t TxnID) String() string {
	return strconv.FormatUint(uint64(t.AccID), 10) + "_" + strconv.FormatUint(uint64(t.ID), 10)
}

func (t SettingsID) String() string {
	return strconv.FormatUint(uint64(t.AccID), 10) + "_" + strconv.FormatUint(uint64(t.ID), 10)
}

func (s Sign) String() string {
	return hex.EncodeToString(s[:])
}

func (th Hash) String() string {
	return hex.EncodeToString(th[:])
}

func GetSignFromString(s string) (Sign, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return ZeroSign, ErrInvalidSign
	}

	var h Sign
	copy(h[:], b)

	return h, nil
}

func SignFromString(s string) Sign {
	h, err := GetSignFromString(s)
	if err != nil {
		panic("invalid sign from string: " + s)
	}
	return h
}

func GetHashFromString(s string) (Hash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return ZeroHash, ErrInvalidHash
	}

	var h Hash
	copy(h[:], b)

	return h, nil
}

func HashFromString(s string) Hash {
	h, err := GetHashFromString(s)
	if err != nil {
		panic("invalid txn hash from string: " + s)
	}
	return h
}

func GetHashDefault(txn *Txn) Hash {
	h := HashNew()
	return GetHash(h, txn)
}

func GetHash(h hash.Hash, txn *Txn) Hash {
	if h.Size() != len(txn.Hash) {
		panic("hash size differs")
	}

	h.Reset()

	order := binary.BigEndian
	buf := txn.Hash[:]

	// TODO(outself): disable error check lint. write errors unnecessary
	order.PutUint64(buf, uint64(txn.ID))
	h.Write(buf[:8])
	order.PutUint64(buf, uint64(txn.Sender))
	h.Write(buf[:8])
	order.PutUint64(buf, uint64(txn.Receiver))
	h.Write(buf[:8])
	order.PutUint64(buf, uint64(txn.Amount))
	h.Write(buf[:8])
	order.PutUint64(buf, uint64(txn.Balance))
	h.Write(buf[:8])
	order.PutUint64(buf, uint64(txn.SettingsID))
	h.Write(buf[:8])

	h.Write(txn.PrevHash[:])

	_ = h.Sum(buf[:0])
	return txn.Hash
}

func GetSettingsHashDefault(s *Settings) Hash {
	h := HashNew()
	return GetSettingsHash(h, s)
}

func GetSettingsHash(h hash.Hash, s *Settings) Hash {
	if h.Size() != len(s.Hash) {
		panic("hash size differs")
	}

	h.Reset()

	order := binary.BigEndian
	buf := s.Hash[:]

	// TODO(outself): disable error check lint. write errors unnecessary
	order.PutUint64(buf, uint64(s.ID))
	h.Write(buf[:8])
	order.PutUint64(buf, uint64(s.Account))
	h.Write(buf[:8])

	if s.VerifyTransferSign {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	h.Write(buf[:1])

	h.Write(s.PrevHash[:])
	h.Write(s.PublicKey[:])
	h.Write(s.DataHash[:])

	_ = h.Sum(buf[:0])
	return s.Hash
}

func GetTransferHashDefault(t Transfer) Hash {
	h := HashNew()
	return GetTransferHash(h, t)
}

func GetTransferHash(h hash.Hash, t Transfer) Hash {
	var hbuf Hash
	if h.Size() != len(hbuf) {
		panic("hash size differs")
	}

	h.Reset()
	order := binary.BigEndian

	buf := hbuf[:]

	order.PutUint64(buf, uint64(t.Sender))
	h.Write(buf[:8])

	for _, ti := range t.Batch {
		order.PutUint64(buf, uint64(ti.Receiver))
		h.Write(buf[:8])

		order.PutUint64(buf, uint64(ti.Amount))
		h.Write(buf[:8])
	}

	h.Write(t.PrevHash[:])

	order.PutUint64(buf, uint64(t.SettingsID))
	h.Write(buf[:8])

	h.Sum(buf[:0])
	return hbuf
}

func GetSettingsRequestHashDefault(s *Settings) Hash {
	h := HashNew()
	return GetSettingsRequestHash(h, s)
}

func GetSettingsRequestHash(h hash.Hash, s *Settings) Hash {
	if h.Size() != len(s.Hash) {
		panic("hash size differs")
	}

	h.Reset()

	order := binary.BigEndian
	buf := s.Hash[:]

	// TODO(outself): disable error check lint. write errors unnecessary
	order.PutUint64(buf, uint64(s.Account))
	h.Write(buf[:8])

	if s.VerifyTransferSign {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	h.Write(buf[:1])

	h.Write(s.PrevHash[:])
	h.Write(s.PublicKey[:])
	h.Write(s.DataHash[:])

	_ = h.Sum(buf[:0])
	return s.Hash
}

func NewSingleTransfer(sender, receiver AccID, amount int64) Transfer {
	transfer := Transfer{Sender: sender}
	transfer.AddReceiver(receiver, amount)
	return transfer
}

func (t *Transfer) AddReceiver(receiver AccID, amount int64) {
	t.Batch = append(t.Batch, &TransferItem{Receiver: receiver, Amount: amount})
}

func ParsePubKey(s string) (PublicKey, error) {
	if s == "" {
		return nil, nil
	}
	return PublicKey(base58.Decode(s)), nil
}

func (k PublicKey) String() string {
	return base58.Encode(k)
}

func NewTxnID(acc AccID, id ID) TxnID {
	return TxnID{AccID: acc, ID: id}
}

func NewSettingsID(acc AccID, id ID) SettingsID {
	return SettingsID{AccID: acc, ID: id}
}
