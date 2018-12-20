package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"

	"github.com/qiwitech/qdp/proto/apipb"
	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/proto/metadbpb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
)

var (
	TxnsMetaPrefix = []byte("m")
)

var ErrMetaIsNotAvailable = errors.New("metadb is not available")

type Service struct {
	gate    gatepb.ProcessorServiceInterface
	plutodb plutodbpb.PlutoDBServiceInterface
	metadb  metadbpb.MetaDBServiceInterface
}

func NewService(gate gatepb.ProcessorServiceInterface) *Service {
	return &Service{
		gate: gate,
	}
}

func (s *Service) SetMetaDBClient(c metadbpb.MetaDBServiceInterface) {
	s.metadb = c
}

func (s *Service) SetPlutoDBClient(c plutodbpb.PlutoDBServiceInterface) {
	s.plutodb = c
}

func (s *Service) ProcessTransfer(ctx context.Context, req *apipb.TransferRequest) (*apipb.TransferResponse, error) {
	//ctx, cancel := context.WithTimeout(ctx, time.Second)
	//defer cancel()

	res := &apipb.TransferResponse{Status: &apipb.Status{}}

	if m := req.Metadata; m != nil {
		if s.metadb == nil {
			return nil, ErrMetaIsNotAvailable
		}
		if len(m.Key) == 0 {
			res.Status.Code = apipb.TransferCode_METADATA_ERROR
			res.Status.Message = "key must be specified"
			return res, nil
		}
	}

	t := &gatepb.TransferRequest{
		Sender:     req.Sender,
		Batch:      make([]*gatepb.TransferItem, len(req.Batch)),
		SettingsId: req.SettingsId,
		PrevHash:   req.PrevHash,
		Sign:       req.Sign,
	}

	for i, r := range req.Batch {
		t.Batch[i] = &gatepb.TransferItem{Receiver: r.Receiver, Amount: r.Amount}
	}

	gateres, err := s.gate.ProcessTransfer(ctx, t)
	if err != nil {
		return nil, errors.Wrap(err, "api")
	}

	res.Status.Code = apipb.TransferCode(gateres.Status.Code)
	res.Status.Message = gateres.Status.Message

	// helper fields. send them any way
	res.Hash = gateres.Hash
	res.SettingsId = gateres.SettingsId

	if res.Status.Code != 0 {
		return res, nil
	}

	if m := req.Metadata; m != nil {
		d := &metadbpb.Data{
			Key: m.Key,
			Index: []*metadbpb.Pair{
				{Key: []byte("prevhash"), Value: []byte(req.PrevHash)},
				{Key: []byte("sender"), Value: tobytes(req.Sender)},
				{Key: []byte("id"), Value: tobytes(gateres.Id)},
			},
		}
		if len(m.Index) != 0 {
			for k, v := range m.Index {
				d.Index = append(d.Index, &metadbpb.Pair{Key: []byte(k), Value: v})
			}
		}
		if len(m.Data) != 0 {
			d.Fields = make([]*metadbpb.Pair, len(m.Data))
			i := 0
			for k, v := range m.Data {
				d.Fields[i] = &metadbpb.Pair{Key: []byte(k), Value: v}
				i++
			}
		}
		resp, err := s.metadb.Put(ctx, &metadbpb.PutRequest{Prefix: TxnsMetaPrefix, Data: d})
		if err != nil {
			return nil, errors.Wrap(err, "meta")
		}
		if resp.Status.Code != metadbpb.DBStatusCode_OK {
			res.Status.Code = apipb.TransferCode_METADATA_ERROR
			res.Status.Message = resp.Status.Message
			return res, nil
		}
	}

	res.TxnId = gateres.TxnId

	return res, nil
}

func (s *Service) GetPrevHash(ctx context.Context, req *apipb.GetPrevHashRequest) (*apipb.GetPrevHashResponse, error) {
	//ctx, cancel := context.WithTimeout(ctx, time.Second)
	//defer cancel()

	gateres, err := s.gate.GetPrevHash(ctx, &gatepb.GetPrevHashRequest{Account: req.Account})
	if err != nil {
		return nil, errors.Wrap(err, "api")
	}

	res := &apipb.GetPrevHashResponse{
		Status: &apipb.Status{
			Code:    apipb.TransferCode(gateres.Status.Code),
			Message: gateres.Status.Message,
		},
		Hash: gateres.Hash,
	}

	return res, nil
}

func (s *Service) GetBalance(ctx context.Context, req *apipb.GetBalanceRequest) (*apipb.GetBalanceResponse, error) {
	//ctx, cancel := context.WithTimeout(ctx, time.Second)
	//defer cancel()

	gatereq := gatepb.GetBalanceRequest(*req)
	gateres, err := s.gate.GetBalance(ctx, &gatereq)
	if err != nil {
		return nil, errors.Wrap(err, "api")
	}

	res := &apipb.GetBalanceResponse{
		Status: &apipb.Status{
			Code:    apipb.TransferCode(gateres.Status.Code),
			Message: gateres.Status.Message,
		},
		Balance: gateres.Balance,
	}

	return res, nil
}

func (s *Service) UpdateSettings(ctx context.Context, req *apipb.SettingsRequest) (*apipb.SettingsResponse, error) {
	gatereq := gatepb.SettingsRequest(*req)
	gateres, err := s.gate.UpdateSettings(ctx, &gatereq)
	if err != nil {
		return nil, errors.Wrap(err, "api")
	}

	res := &apipb.SettingsResponse{
		Status: &apipb.Status{
			Code:    apipb.TransferCode(gateres.Status.Code),
			Message: gateres.Status.Message,
		},
		Hash:       gateres.Hash,
		SettingsId: gateres.SettingsId,
	}

	return res, nil
}

func (s *Service) GetLastSettings(ctx context.Context, req *apipb.GetLastSettingsRequest) (*apipb.GetLastSettingsResponse, error) {
	//ctx, cancel := context.WithTimeout(ctx, time.Second)
	//defer cancel()

	gatereq := gatepb.GetLastSettingsRequest(*req)
	gateres, err := s.gate.GetLastSettings(ctx, &gatereq)
	if err != nil {
		return nil, errors.Wrap(err, "api")
	}

	res := &apipb.GetLastSettingsResponse{
		Status: &apipb.Status{
			Code:    apipb.TransferCode(gateres.Status.Code),
			Message: gateres.Status.Message,
		},
		Id:                 gateres.Id,
		Hash:               gateres.Hash,
		Account:            gateres.Account,
		PublicKey:          gateres.PublicKey,
		PrevHash:           gateres.PrevHash,
		DataHash:           gateres.DataHash,
		Sign:               gateres.Sign,
		VerifyTransferSign: gateres.VerifyTransferSign,
	}

	return res, nil
}

func (s *Service) GetHistory(ctx context.Context, req *apipb.GetHistoryRequest) (*apipb.GetHistoryResponse, error) {
	//ctx, cancel := context.WithTimeout(ctx, time.Second)
	//defer cancel()
	if s.plutodb == nil {
		return nil, errors.New("plutodb is not available")
	}

	pdbreq := plutodbpb.GetHistoryRequest(*req)
	pdbresp, err := s.plutodb.GetHistory(ctx, &pdbreq)
	if err != nil {
		return nil, errors.Wrap(err, "api")
	}

	res := &apipb.GetHistoryResponse{
		Status: &apipb.Status{
			Code:    apipb.TransferCode(pdbresp.Status.Code),
			Message: pdbresp.Status.Message,
		},
		Token: pdbresp.Token,
	}
	res.Txns = txnsToApi(pdbresp.Txns)

	return res, nil
}

func (s *Service) GetByMetaKey(ctx context.Context, req *apipb.GetByMetaKeyRequest) (*apipb.GetByMetaKeyResponse, error) {
	if s.metadb == nil {
		return nil, ErrMetaIsNotAvailable
	}
	//	if s.plutodb == nil {
	//		return nil, errors.New("plutodb is not available")
	//	}

	res := &apipb.GetByMetaKeyResponse{
		Status: &apipb.Status{},
	}

	metareq := &metadbpb.GetMultiRequest{
		Prefix: TxnsMetaPrefix,
		Keys:   req.Keys,
	}
	metaresp, err := s.metadb.GetMulti(ctx, metareq)
	if err != nil {
		return nil, errors.Wrap(err, "metadb")
	}
	if metaresp.Status.Code != 0 {
		res.Status = &apipb.Status{
			Code:    apipb.TransferCode(apipb.TransferCode_METADATA_ERROR),
			Message: metaresp.Status.Message,
		}
		return res, nil
	}

	if s.plutodb != nil {
		txnsreq := &plutodbpb.GetTxnMultiRequest{IDs: make([]*chainpb.TxnID, len(metaresp.Results))}
		for i := range metaresp.Results {
			txnid := &chainpb.TxnID{}
			idx := metaresp.Results[i].Index
			for _, p := range idx {
				if bytes.Equal([]byte("sender"), p.Key) {
					unbytes(p.Value, &txnid.Account)
				}
				if bytes.Equal([]byte("id"), p.Key) {
					unbytes(p.Value, &txnid.ID)
				}
			}
			txnsreq.IDs[i] = txnid
		}
		txnsres, err := s.plutodb.GetTxnMulti(ctx, txnsreq)
		if err != nil {
			return nil, errors.Wrap(err, "plutodb")
		}

		if len(txnsres.Txns) != len(txnsreq.IDs) {
			panic(fmt.Sprintf("got %d txns on %d request length", len(txnsres.Txns), len(txnsreq.IDs)))
		}

		res.Txns = txnsToApi(txnsres.Txns)
	} else {
		res.Txns = make([]*apipb.Txn, len(metaresp.Results))
		for i := range metaresp.Results {
			res.Txns[i] = &apipb.Txn{}
		}
	}

	for i, it := range metaresp.Results {
		res.Txns[i].Meta = MetaFromData(it)
	}

	if s.plutodb != nil {
		res.Txns = removeZeros(res.Txns)
	}

	return res, nil
}

func (s *Service) SearchMeta(ctx context.Context, req *apipb.SearchMetaRequest) (*apipb.SearchMetaResponse, error) {
	if s.metadb == nil {
		return nil, ErrMetaIsNotAvailable
	}

	res := &apipb.SearchMetaResponse{
		Status: &apipb.Status{},
	}

	var fs []*metadbpb.Pair
	if idx := req.Index; idx != nil {
		fs = make([]*metadbpb.Pair, len(req.Index))
		i := 0
		for k, v := range req.Index {
			fs[i] = &metadbpb.Pair{Key: []byte(k), Value: v}
			i++
		}
	}

	mreq := &metadbpb.SearchRequest{Prefix: TxnsMetaPrefix, Filters: fs, Token: req.Token, Limit: req.Limit}

	mresp, err := s.metadb.Search(ctx, mreq)
	if err != nil {
		res.Status.Code = apipb.TransferCode_METADATA_ERROR
		res.Status.Message = err.Error()
		return res, nil
	}
	if mresp.Status == nil {
		res.Status.Code = apipb.TransferCode_METADATA_ERROR
		res.Status.Message = "no status received"
		return res, nil
	}
	if mresp.Status.Code != 0 {
		res.Status.Code = apipb.TransferCode_METADATA_ERROR
		res.Status.Message = fmt.Sprintf("code %d: %v", mresp.Status.Code, mresp.Status.Message)
		return res, nil
	}

	res.Items = make([]*apipb.Meta, len(mresp.Items))
	for i, it := range mresp.Items {
		res.Items[i] = MetaFromData(it)
	}

	res.NextToken = mresp.NextToken

	return res, nil
}

func (s *Service) PutMeta(ctx context.Context, req *apipb.PutMetaRequest) (*apipb.PutMetaResponse, error) {
	if s.metadb == nil {
		return nil, ErrMetaIsNotAvailable
	}

	res := &apipb.PutMetaResponse{Status: &apipb.Status{}}

	m := req.Meta
	if m == nil || len(m.Key) == 0 {
		res.Status.Code = apipb.TransferCode_METADATA_ERROR
		res.Status.Message = "key must be specified"
		return res, nil
	}

	d := &metadbpb.Data{
		Key: m.Key,
	}
	if len(m.Index) != 0 {
		for k, v := range m.Index {
			d.Index = append(d.Index, &metadbpb.Pair{Key: []byte(k), Value: v})
		}
	}
	if len(m.Data) != 0 {
		d.Fields = make([]*metadbpb.Pair, len(m.Data))
		i := 0
		for k, v := range m.Data {
			d.Fields[i] = &metadbpb.Pair{Key: []byte(k), Value: v}
			i++
		}
	}
	resp, err := s.metadb.Put(ctx, &metadbpb.PutRequest{Prefix: TxnsMetaPrefix, Data: d})
	if err != nil {
		return nil, errors.Wrap(err, "meta")
	}
	if resp.Status.Code != metadbpb.DBStatusCode_OK {
		res.Status.Code = apipb.TransferCode_METADATA_ERROR
		res.Status.Message = resp.Status.Message
		return res, nil
	}

	return res, nil
}

func MetaFromData(d *metadbpb.Data) *apipb.Meta {
	m := &apipb.Meta{
		Key: d.Key,
	}
	if fs := d.Index; len(fs) != 0 {
		data := make(map[string][]byte, len(fs))
		for _, p := range fs {
			data[string(p.Key)] = p.Value
		}
		m.Index = data
	}
	if fs := d.Fields; len(fs) != 0 {
		data := make(map[string][]byte, len(fs))
		for _, p := range fs {
			data[string(p.Key)] = p.Value
		}
		m.Data = data
	}
	return m
}

func removeZeros(in []*apipb.Txn) []*apipb.Txn {
	s, d := 0, 0
	for s < len(in) {
		id := in[s].Id
		if id == "" || id == "0" {
			s++
			continue
		}

		in[d] = in[s]
		s++
		d++
	}

	in = in[:d]

	return in
}

func txnsToApi(in []*chainpb.Txn) []*apipb.Txn {
	txns := make([]*apipb.Txn, len(in))
	for i := range in {
		t := in[i]
		txns[i] = &apipb.Txn{
			Id:       fmtID(t.ID),
			Sender:   fmtAccID(t.Sender),
			Receiver: fmtAccID(t.Receiver),
			Amount:   fmtAmount(t.Amount),
			Balance:  fmtAmount(t.Balance),
			SpentBy:  fmtID(t.SpentBy),
			PrevHash: fmtHash(t.PrevHash),
			Hash:     fmtHash(t.Hash),
			Sign:     fmtSign(t.Sign),
		}
	}
	return txns
}

func fmtID(v uint64) string {
	return fmt.Sprintf("%d", v)
}

func fmtAccID(v uint64) string {
	return fmt.Sprintf("%d", v)
}

func fmtAmount(v int64) string {
	return fmt.Sprintf("%d", v)
}

func fmtHash(v []byte) string {
	return fmt.Sprintf("%x", v)
}

func fmtSign(v []byte) string {
	return fmt.Sprintf("%x", v)
}

func tobytes(k ...interface{}) []byte {
	i := 0
	for _, k := range k {
		switch k := k.(type) {
		case rune:
			i++
		case uint64:
			i += 8
		case string:
			i += len(k)
		case []byte:
			i += len(k)
		default:
			panic(k)
		}
	}
	r := make([]byte, i)
	i = 0
	for _, k := range k {
		switch k := k.(type) {
		case rune:
			r[i] = byte(k)
			i++
		case uint64:
			binary.BigEndian.PutUint64(r[i:], k)
			i += 8
		case string:
			i += copy(r[i:], k)
		case []byte:
			i += copy(r[i:], k)
		default:
			panic(k)
		}
	}
	return r
}

func unbytes(v []byte, k ...interface{}) {
	i := 0
	for _, k := range k {
		switch k := k.(type) {
		case *rune:
			*k = rune(v[i])
			i++
		case *byte:
			*k = byte(v[i])
			i++
		case *uint64:
			*k = binary.BigEndian.Uint64(v[i:])
			i += 8
		case *string:
			*k = string(v[i:])
			i += len(*k)
		case *[]byte:
			*k = v[i:]
			i += len(*k)
		default:
			panic(k)
		}
	}
}
