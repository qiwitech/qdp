package sqlapi

import (
	"context"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/juju/errors"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/sqltypes"
	vtrpcpb "vitess.io/vitess/go/vt/proto/vtrpc"
	"vitess.io/vitess/go/vt/sqlparser"
	"vitess.io/vitess/go/vt/vterrors"

	"github.com/qiwitech/qdp/proto/apipb"
)

const (
	TxnsTable = "txns"
)

var defTimeout = 5 * time.Second

type (
	callback  = func(*sqltypes.Result) error
	valParser func(req *apipb.TransferRequest, i int, f sqlparser.Expr) error

	Server struct {
		l *mysql.Listener

		db  string
		api apipb.APIServiceInterface
		col map[string]valParser
	}
)

func New() (*Server, error) {
	s := &Server{}

	s.col = map[string]valParser{
		"sender":    s.fillSender,
		"receiver":  s.fillReceiver,
		"amount":    s.fillAmount,
		"prev_hash": s.fillPrevHash,
	}

	return s, nil
}

func (s *Server) SetAPIService(a apipb.APIServiceInterface) {
	s.api = a
}

func (s *Server) Serve(lis net.Listener) error {
	l, err := mysql.NewFromListener(lis, &mysql.AuthServerNone{}, s, defTimeout, defTimeout)
	if err != nil {
		return errors.Trace(err)
	}

	s.l = l

	l.Accept()

	return nil
}

func (s *Server) Close() {
	s.l.Close()
}

func (s *Server) NewConnection(c *mysql.Conn) {
	log.Printf("new conn:    %v %v", c.RemoteAddr(), c.ID())
}

func (s *Server) ConnectionClosed(c *mysql.Conn) {
	log.Printf("closed conn: %v %v", c.RemoteAddr(), c.ID())
}

func (s *Server) WarningCount(c *mysql.Conn) uint16 {
	return 0
}

func (s *Server) ComQuery(c *mysql.Conn, q string, cb callback) error {
	log.Printf("query [%v]: %+v", c.ID(), q)

	stmt, err := sqlparser.Parse(q)
	if err != nil {
		return errors.Trace(err)
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Show:
		return s.show(c, stmt, cb)
	case *sqlparser.Insert:
		return s.insert(c, stmt, cb)
	case *sqlparser.Select:
		return s.select_(c, stmt, cb)
	}

	//	log.Printf("query [%v]: (%T) %+v", c.ID(), stmt, stmt)
	return cb(&sqltypes.Result{})
}

func (s *Server) show(c *mysql.Conn, st *sqlparser.Show, cb callback) error {
	log.Printf("query [%v] show: %+v", c.ID(), st)
	var err error
	switch st.Type {
	case "databases":
		err = cb(sqltypes.MakeTestResult(sqltypes.MakeTestFields("db", "text"),
			"main"))
	case "tables":
		err = cb(sqltypes.MakeTestResult(sqltypes.MakeTestFields("table", "text"),
			TxnsTable))
	case "variables":
		err = cb(sqltypes.MakeTestResult(sqltypes.MakeTestFields("Variable_name|Value", "text|text")))
	}
	if err != nil {
		return err
	}
	return cb(&sqltypes.Result{})
}

func (s *Server) insert(c *mysql.Conn, st *sqlparser.Insert, cb callback) error {
	log.Printf("query [%v] insert: %+v %T", c.ID(), st, st.Rows)
	if st.Table.Name.String() != TxnsTable {
		return vterrors.Errorf(vtrpcpb.Code_NOT_FOUND, "table %s is not found", st.Table.Name.String())
	}

	rows := st.Rows.(sqlparser.Values)
	if len(rows) == 0 {
		return cb(&sqltypes.Result{})
	}

	req := &apipb.TransferRequest{
		Batch: make([]*apipb.TransferItem, len(rows)),
	}

	for i, r := range rows {
		req.Batch[i] = &apipb.TransferItem{}

		for j, f := range r {
			p, ok := s.col[st.Columns[j].CompliantName()]
			if !ok {
				return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "column %s does not exists", st.Columns[j].CompliantName())
			}
			//	log.Printf("field %d:%d (%s): %T %+v", i, j, st.Columns[j].CompliantName(), f, f)
			if err := p(req, i, f); err != nil {
				return err
			}
		}
	}

	ctx := context.TODO()
	resp, err := s.api.ProcessTransfer(ctx, req)
	log.Printf("req-resp-err: %+v %+v %v", req, resp, err)
	if err != nil {
		return vterrors.Errorf(vtrpcpb.Code_INTERNAL, "processing error: %v", err)
	}

	if resp.Status != nil && resp.Status.Code != 0 {
		return vterrors.Errorf(vtrpcpb.Code_INTERNAL, "processing error: %v", resp.Status.Message)
	}

	res := &sqltypes.Result{
		Fields:       sqltypes.MakeTestFields("txn_id|hash|settings_id", "text|text|text"),
		Rows:         [][]sqltypes.Value{{sqltypes.NewVarChar(resp.TxnId), sqltypes.NewVarChar(resp.Hash), sqltypes.NewUint64(resp.SettingsId)}},
		RowsAffected: uint64(len(rows)),
	}
	err = cb(res)
	if err != nil {
		return err
	}

	return cb(&sqltypes.Result{})
}

func (s *Server) select_(c *mysql.Conn, st *sqlparser.Select, cb callback) error {
	log.Printf("query [%v] select: %+v", c.ID(), st)

	err := s.checkSelect(st)
	if err != nil {
		return err
	}

	req := &apipb.GetHistoryRequest{}

	err = s.parseWhere(req, st.Where)
	if err != nil {
		return err
	}

	err = s.parseLimit(req, st.Limit)
	if err != nil {
		return err
	}

	resp, err := s.api.GetHistory(context.TODO(), req)
	log.Printf("req-resp-err: %+v %+v %v", req, resp, err)
	if err != nil {
		return vterrors.Errorf(vtrpcpb.Code_INTERNAL, "processing error: %v", err)
	}

	if resp.Status != nil && resp.Status.Code != 0 {
		return vterrors.Errorf(vtrpcpb.Code_INTERNAL, "processing error: %v", resp.Status.Message)
	}

	fs := sqltypes.MakeTestFields("sender|  id|receiver|amount|balance|hash|prev_hash|spent_by", "text|text|text|text|text|text|text|text|text")
	err = cb(&sqltypes.Result{Fields: fs})
	if err != nil {
		return err
	}

	// print response
	res := &sqltypes.Result{
		Fields: fs,
		Rows:   [][]sqltypes.Value{},
	}
	for _, t := range resp.Txns {
		//err = cb(sqltypes.MakeTestResult(fs, fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v", t.Sender, t.Id, t.Receiver, t.Amount, t.Balance, t.Hash, t.SpentBy)))
		res.Rows = append(res.Rows, []sqltypes.Value{
			sqltypes.NewVarChar(t.Sender),
			sqltypes.NewVarChar(t.Id),
			sqltypes.NewVarChar(t.Receiver),
			sqltypes.NewVarChar(t.Amount),
			sqltypes.NewVarChar(t.Balance),
			sqltypes.NewVarChar(t.Hash),
			sqltypes.NewVarChar(t.PrevHash),
			sqltypes.NewVarChar(t.SpentBy),
		})
	}

	err = cb(res)
	if err != nil {
		return err
	}

	return cb(&sqltypes.Result{})
}

func (s *Server) fillSender(req *apipb.TransferRequest, i int, f sqlparser.Expr) error {
	acc, err := s.parseAccID(f)
	if err != nil {
		return err
	}

	if i != 0 && req.Sender != acc {
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "only single sender must be used in one batch")
	}

	req.Sender = acc

	return nil
}

func (s *Server) fillReceiver(req *apipb.TransferRequest, i int, f sqlparser.Expr) error {
	acc, err := s.parseAccID(f)
	if err != nil {
		return err
	}

	log.Printf("req %+v", req)
	req.Batch[i].Receiver = acc

	return nil
}

func (s *Server) fillAmount(req *apipb.TransferRequest, i int, f sqlparser.Expr) error {
	a, err := s.parseAmount(f)
	if err != nil {
		return err
	}

	req.Batch[i].Amount = a

	return nil
}

func (s *Server) fillPrevHash(req *apipb.TransferRequest, i int, f sqlparser.Expr) error {
	h, err := s.parseHash(f)
	if err != nil {
		return err
	}

	if i != 0 && h != "" {
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "only first transaction must have non empty prev_hash")
	}

	if i != 0 {
		return nil
	}

	req.PrevHash = h

	return nil
}

func (s *Server) parseName(f interface{}) (string, error) {
	switch f := f.(type) {
	case *sqlparser.SQLVal:
		return string(f.Val), nil
	case sqlparser.TableName:
		return f.Name.CompliantName(), nil
	case *sqlparser.ColName:
		return f.Name.CompliantName(), nil
	case *sqlparser.AliasedTableExpr:
		return s.parseName(f.Expr)
	default:
		return "", vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported name expression: %T %v", f, f)
	}
}

func (s *Server) parseAccID(f sqlparser.Expr) (uint64, error) {
	switch f := f.(type) {
	case *sqlparser.SQLVal:
		acc, err := strconv.ParseUint(string(f.Val), 10, 64)
		if err != nil {
			return 0, vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "parse account expression %q: %v", f.Val, err)
		}

		return acc, nil
	default:
		return 0, vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported account expression: %T %v", f, f)
	}
}

func (s *Server) parseAmount(f sqlparser.Expr) (int64, error) {
	switch f := f.(type) {
	case *sqlparser.SQLVal:
		a, err := strconv.ParseInt(string(f.Val), 10, 64)
		if err != nil {
			return 0, vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "parse amount expression %q: %v", f.Val, err)
		}

		return a, nil
	default:
		return 0, vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported amount expression: %T %v", f, f)
	}
}

func (s *Server) parseHash(f sqlparser.Expr) (string, error) {
	switch f := f.(type) {
	case *sqlparser.SQLVal:
		return string(f.Val), nil
	case *sqlparser.FuncExpr:
		return "", vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "functions are not supported for hash expression")
	default:
		return "", vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported hash expression: %T %v", f, f)
	}
}

func (s *Server) parseToken(f sqlparser.Expr) (string, error) {
	switch f := f.(type) {
	case *sqlparser.SQLVal:
		return string(f.Val), nil
	case *sqlparser.FuncExpr:
		return "", vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "functions are not supported for token expression")
	default:
		return "", vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported token expression: %T %v", f, f)
	}
}

func (s *Server) checkSelect(st *sqlparser.Select) error {
	if len(st.From) > 1 {
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "there is only one supported table: %v", TxnsTable)
	}
	if len(st.From) == 0 { // hack
		return nil
	}
	name, err := s.parseName(st.From[0])
	if err != nil {
		return err
	}
	if name != TxnsTable {
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "there is only one supported table: %v", TxnsTable)
	}
	return nil
}

func (s *Server) parseWhere(req *apipb.GetHistoryRequest, w *sqlparser.Where) error {
	if w.Type != "where" {
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "only WHERE expression is supported")
	}

	//	log.Printf("where: %T %+v", w.Expr, w.Expr)
	switch e := w.Expr.(type) {
	case *sqlparser.ComparisonExpr:
		if e.Operator != "=" {
			return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "the only supported WHERE form is account = <acc>")
		}
		name, err := s.parseName(e.Left)
		if err != nil {
			return err
		}
		if name != "account" {
			return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "the only supported WHERE form is account = <acc>")
		}

		acc, err := s.parseAccID(e.Right)
		if err != nil {
			return err
		}

		req.Account = acc
	default:
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported WHERE expression: %T %v", e, e)
	}

	return nil
}

func (s *Server) parseLimit(req *apipb.GetHistoryRequest, l *sqlparser.Limit) error {
	if l == nil {
		req.Limit = 20
		return nil
	}

	//	log.Printf("limit: off %T %+v count %T %+v", l.Offset, l.Offset, l.Rowcount, l.Rowcount)
	if l.Offset != nil {
		t, err := s.parseToken(l.Offset)
		if err != nil {
			return err
		}

		req.Token = t
	}

	switch f := l.Rowcount.(type) {
	case *sqlparser.SQLVal:
		n, err := strconv.ParseInt(string(f.Val), 10, 64)
		if err != nil {
			return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "parse limit expression %q: %v", f.Val, err)
		}

		req.Limit = uint32(n)

	default:
		return vterrors.Errorf(vtrpcpb.Code_INVALID_ARGUMENT, "unsupported amount expression: %T %v", f, f)
	}

	return nil
}
