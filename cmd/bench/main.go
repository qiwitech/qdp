package main

import (
	"context"
	"crypto/elliptic"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/beorn7/perks/quantile"
	"github.com/btcsuite/btcd/btcec"
	"github.com/eapache/go-resiliency/breaker"
	"github.com/qiwitech/graceful"
	"github.com/qiwitech/tcprpc"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"

	"github.com/qiwitech/qdp/client"
	"github.com/qiwitech/qdp/proto/apipb"
	"github.com/qiwitech/qdp/proto/chainpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
)

var codec graceful.Codec = &graceful.JSONCodec{}

var (
	transfersTotal int64
	transfersSucc  int64
)

var (
	stopc chan struct{}
)

var (
	lmu      sync.Mutex
	Latency  = quantile.NewHighBiased(0.1)
	Inflight = quantile.NewHighBiased(0.1)
	Errors   map[string]int

	inflight int32
)

var (
	accmap  map[uint64]struct{}
	acclist []uint64

	gens map[string]func(string) (Worker, error)
)

var (
	acc  func() uint64                = rndacc
	wgen func(string) (Worker, error) = transfer
)

var oneinit sync.Once

var (
	ostr   io.Writer = os.Stdout
	accrep           = "%20v"
)

var curve elliptic.Curve = btcec.S256()

var (
	saddr     = flag.String("http", ":6006", "http service address")
	addr      = flag.String("addr", "http://localhost:9090/v1/", "api url")
	rate      = flag.Int("rate", 100, "rate limit")
	jobs      = flag.Int("jobs", 1, "parallel jobs")
	burst     = flag.Int("burst", 10, "burst limit")
	bs        = flag.Int("batch", 4, "batch size")
	accs      = flag.Int("accounts", 100000, "unique accounts")
	dur       = flag.Duration("d", 5*time.Second, "duration of test")
	to        = flag.Duration("timeout", time.Minute, "timeout")
	serr      = flag.Bool("e", false, "stop on error")
	verbose   = flag.Bool("verbose", false, "verbose")
	gen       = flag.String("gen", "rnd", "request account generator (rnd|seq)")
	req       = flag.String("req", "transfer", "request type (transfer|keygen|storeput|get|hostname)")
	maxreqs   = flag.Int("maxreqs", 0, "max requests to do (0 for unlimited)")
	silent    = flag.Bool("s", false, "no output")
	inflights = flag.Bool("inflight", true, "calc number of requests inflight")
	perrors   = flag.Bool("errors", true, "count errors")
	prefix    = flag.String("prefix", "", "TCPRPC api prefix")
	kgen      = flag.Bool("keygen", false, "generate private key before transfer if not exists")
	hexrep    = flag.Bool("hex", false, "use account hex representation")
	mixedf    = flag.String("mixed", "1:transfer,10:history", "mixed generator")
)

func init() {
	flag.StringVar(&client.KeysDBFlag, "keysdb", ".plutoclientdb", "plutoclient keys db path")
}

type Worker interface {
	Request(ctx context.Context, s uint64) (interface{}, error)
}

func main() {
	flag.Parse()

	stopc = make(chan struct{})
	Errors = make(map[string]int)

	oneinit.Do(func() {
		if *silent {
			ostr = ioutil.Discard
			*saddr = ""
		}
		if *saddr != "" {
			go func() {
				panic(http.ListenAndServe(*saddr, http.DefaultServeMux))
			}()
		}
		if *hexrep {
			accrep = "%016x"
		}

		gens = map[string]func(string) (Worker, error){
			"get":      getreq,
			"transfer": transfer,
			"keygen":   keygen,
			"history":  history,
			"balance":  balance,
			"dbput":    dbput,
			"hostname": hostname,
			"none":     nonereq,
			"block":    blockreq,
			"mixed":    mixed,
		}
	})

	addrs := strings.Split(*addr, ",")

	fmt.Fprintf(ostr, "[pid %d] making requests with %d workers for %v with %d rps rate limit\n", os.Getpid(), *jobs, *dur, *rate)
	fmt.Fprintf(ostr, "use endpoints: %q\n", addrs)

	genaccaunts()

	switch *gen {
	case "rnd":
		acc = rndacc
		//	case "rndlim":
	//	acc = rndlimacc
	case "seq":
		acc = seqacc()
	default:
		panic("undefined generator")
	}

	var ok bool
	wgen, ok = gens[*req]
	if !ok {
		log.Fatalf("select one of generators: %v", gens)
	}

	var start time.Time

	ctx := context.Background()

	if *to != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, *to)
		defer cancel()
	}

	g, ctx := errgroup.WithContext(context.Background())

	workersc := make([]chan uint64, *jobs)
	for i := 0; i < *jobs; i++ {
		wc := make(chan uint64)
		workersc[i] = wc
		g.Go(func() error {
			return worker(ctx, addrs[rand.Intn(len(addrs))], wc)
		})
	}

	g.Go(func() error { // generator
		defer close(stopc)
		defer func() {
			for _, c := range workersc {
				close(c)
			}
		}()
		start = time.Now()
		return generator(ctx, workersc)
	})

	g.Go(func() error { // signal handler
		shc := make(chan os.Signal, 1)
		signal.Notify(shc, syscall.SIGINT)
		signal.Notify(shc, syscall.SIGTERM)

		defer func() {
			go func() {
				s := <-shc
				if s == syscall.SIGTERM {
					fmt.Fprintf(ostr, "killed with printing all stack traces\n")
					var buf [10000]byte
					n := runtime.Stack(buf[:], true)
					fmt.Fprintf(ostr, "%s\n", buf[:n])
					os.Exit(1)
				} else {
					fmt.Fprintf(ostr, "killed\n")
					os.Exit(1)
				}
			}()
		}()

		select {
		case <-shc:
			return errors.New("interrupted by user")
		case <-stopc:
			return nil
		}
	})

	err := g.Wait()

	tdur := time.Since(start)

	tot := atomic.LoadInt64(&transfersTotal)
	suc := atomic.LoadInt64(&transfersSucc)

	if err != nil {
		fmt.Fprintf(ostr, "finished with error: %v\n", err)
	} else if *serr {
		fmt.Fprintf(ostr, "finished without errors\n")
	}
	tps := float64(suc) / tdur.Seconds()
	prec := 0
	if tps <= 10 {
		prec = 1
	}
	fmt.Fprintf(ostr, "%d successful transfers of %d made (%3.0f%%) on %v (%.*f tps)\n", suc, tot, float64(suc)/float64(tot)*100, tdur, prec, tps)

	fmt.Fprintf(ostr, "latencies (%v):\n", Latency.Count())
	for _, q := range []float64{0, 0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99, 1} {
		v := Latency.Query(q)
		if v <= 1 {
			fmt.Fprintf(ostr, "  %3.0f%%: <= %12.1f ms\n", q*100, v)
		} else {
			fmt.Fprintf(ostr, "  %3.0f%%: <= %10.0f   ms\n", q*100, v)
		}
	}

	if *inflights {
		fmt.Fprintf(ostr, "inflight (%v):\n", Inflight.Count())
		for _, q := range []float64{0, 0.5, 0.75, 0.9, 0.95, 0.99, 1} {
			v := Inflight.Query(q)
			fmt.Fprintf(ostr, "  %3.0f%%: <= %10.0f\n", q*100, v)
		}
	}
	if *perrors {
		type ErrCnt struct {
			e string
			c int
		}
		sum := 0
		list := make([]ErrCnt, 0, len(Errors))
		for e, c := range Errors {
			sum += c
			list = append(list, ErrCnt{e: e, c: c})
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].c > list[j].c
		})
		fmt.Fprintf(ostr, "errors (%d types, %d total):\n", len(Errors), sum)
		for _, ec := range list {
			fmt.Fprintf(ostr, "  %6d (%5.1f%%): %s\n", ec.c, float64(ec.c)*100/float64(sum), ec.e)
		}
	}
}

func generator(ctx context.Context, ws []chan uint64) error {
	var stop *time.Timer
	if *dur != 0 {
		stop = time.NewTimer(*dur)
		defer stop.Stop()
	} else {
		stop = &time.Timer{}
	}

	ratec := make(chan struct{}, *burst)
	if *rate != 0 {
		ticker := time.NewTicker(time.Second / time.Duration(*rate))
		defer ticker.Stop()

		go func() {
			defer close(ratec)
			for {
				select {
				case <-ticker.C:
					ratec <- struct{}{}
				case <-ctx.Done():
					return
				}
			}
		}()
	} else {
		close(ratec)
	}

	var reason string
	put := func() bool {
		var buf [8]byte
	retry:
		a := acc()

		binary.BigEndian.PutUint64(buf[:], uint64(a))
		sum := crc32.ChecksumIEEE(buf[:])
		i := int(sum) % len(ws)

		select {
		case ws[i] <- a:
		case <-ctx.Done():
			return true
		default:
			goto retry
		}

		tot := atomic.AddInt64(&transfersTotal, 1)
		if *maxreqs != 0 && *maxreqs == int(tot) {
			reason = "all planned requests are made"
			return true
		}
		return false
	}

loop:
	for {
		select {
		case <-stop.C:
			reason = "time is over"
			break loop
		case <-ctx.Done():
			return ctx.Err()
		case _, _ = <-ratec:
			if put() {
				break loop
			}
		}
	}
	if *verbose {
		log.Printf("generator stopped: %v", reason)
	}
	return nil
}

func worker(ctx context.Context, addr string, c chan uint64) error {

	if *verbose {
		log.Printf("start worker with api endpoint %v", addr)
	}

	w, err := wgen(addr)
	if err != nil {
		return errors.Wrap(err, "create worker")
	}

	for {
		select {
		case s, ok := <-c:
			if !ok {
				return nil
			}
			fin := reqstart(s)
			resp, err := w.Request(ctx, s)
			fin(resp, err)
			if err != nil {
				if *serr {
					return err
				} else if !*verbose {
					log.Printf("worker (acc %v): %v", s, err)
				}
			} else {
				atomic.AddInt64(&transfersSucc, 1)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func reqstart(s uint64) func(resp interface{}, err error) {
	atomic.AddInt32(&inflight, 1)
	st := time.Now()

	return func(resp interface{}, err error) {
		d := time.Since(st)
		still := atomic.AddInt32(&inflight, -1) + 1
		lmu.Lock()
		Latency.Insert(d.Seconds() * 1000)
		Inflight.Insert(float64(still))
		if err == nil {
			Errors["<nil>"]++
		} else {
			Errors[err.Error()]++
		}
		lmu.Unlock()
		if *verbose {
			log.Printf("[%4d inflight] %10.2fms acc "+accrep+" -> resp %+v err %v", still, d.Seconds()*1000, s, resp, err)
		}
	}
}

func mixed(addr string) (Worker, error) {
	w := Multi{}

	q := strings.Split(*mixedf, ",")
	for _, g := range q {
		var c int
		var n string
		_, err := fmt.Sscanf(g, "%d:%s", &c, &n)
		if err != nil {
			return nil, errors.Wrapf(err, "parse mixed generator %v", g)
		}

		gen, ok := gens[n]
		if !ok {
			return nil, errors.Wrapf(err, "no such generator (select from %v)", gens)
		}

		worker, err := gen(addr)
		if err != nil {
			return nil, errors.Wrapf(err, "create mixed generator %v", n)
		}

		for i := 0; i < c; i++ {
			w.list = append(w.list, worker)
		}
	}

	return w, nil
}

type Multi struct {
	list []Worker
}

func (w Multi) Request(ctx context.Context, s uint64) (interface{}, error) {
	i := rand.Intn(len(w.list))
	return w.list[i].Request(ctx, s)
}

func transfer(addr string) (Worker, error) {
	cl := NewClient(addr)
	return &Transfer{cl: apipb.NewAPIServiceHTTPClient(cl), cache: make(map[uint64]*AccCache)}, nil
}

type Transfer struct {
	cl    apipb.APIServiceInterface
	cache map[uint64]*AccCache
}

type AccCache struct {
	SittingsID uint64
	LastHash   string
	Priv       *btcec.PrivateKey
}

func (w *Transfer) Request(ctx context.Context, s uint64) (interface{}, error) {
	req := &apipb.TransferRequest{
		Sender: s,
		Batch:  batch(),
	}
	c, ok := w.cache[s]
	if !ok {
		var err error
		c, err = w.openAccCache(ctx, s)
		if err != nil {
			return nil, err
		}
		w.cache[s] = c
	}
	req.SettingsId = c.SittingsID
	req.PrevHash = c.LastHash
	if c.Priv != nil {
		err := client.SignTransfer(req)
		if err != nil {
			return nil, err
		}
	}

	resp, err := w.cl.ProcessTransfer(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Status == nil {
		return nil, errors.New("no status")
	}

	c.LastHash = resp.Hash
	c.SittingsID = resp.SettingsId

	if resp.Status != nil && resp.Status.Code != 0 {
		return nil, errors.New(resp.Status.Message)
	}

	return resp.Status, nil
}

var openAccMu sync.Mutex

func (w *Transfer) openAccCache(ctx context.Context, s uint64) (*AccCache, error) {
	c := &AccCache{}

	resp, err := w.cl.GetPrevHash(ctx, &apipb.GetPrevHashRequest{Account: s})
	if err != nil {
		return nil, err
	}
	if resp.Status != nil && resp.Status.Code != apipb.TransferCode_OK {
		return nil, errors.New(resp.Status.Message)
	}
	c.LastHash = resp.Hash

retry:
	err = func() error {
		defer openAccMu.Unlock()
		openAccMu.Lock()

		k, err := client.LoadPrivateKey(s)
		if err != nil {
			return err
		}
		c.Priv = k

		return nil
	}()
	if err != nil {
		return nil, err
	}

	if *kgen && c.Priv == nil {
		start := time.Now()
		w := Keygen{cl: w.cl}
		_, err = w.Request(ctx, s)
		if *verbose {
			d := time.Since(start)
			log.Printf("generated key at %9.2fms acc "+accrep+" -> %v", d.Seconds()*1000, s, err)
		}
		if err != nil {
			return nil, err
		}
		goto retry
	}

	sett, err := w.cl.GetLastSettings(ctx, &apipb.GetLastSettingsRequest{Account: s})
	if err != nil {
		return nil, err
	}
	if resp.Status != nil && resp.Status.Code != apipb.TransferCode_OK {
		return nil, errors.New(resp.Status.Message)
	}
	c.SittingsID = sett.Id

	return c, nil
}

func batch() []*apipb.TransferItem {
	r := make([]*apipb.TransferItem, *bs)
	for i := 0; i < *bs; i++ {
		r[i] = &apipb.TransferItem{Receiver: acc()}
	}
	return r
}

func history(addr string) (Worker, error) {
	cl := NewClient(addr)
	return &History{cl: apipb.NewAPIServiceHTTPClient(cl)}, nil
}

type History struct {
	cl apipb.APIServiceInterface
}

func (w *History) Request(ctx context.Context, u uint64) (interface{}, error) {
	resp, err := w.cl.GetHistory(ctx, &apipb.GetHistoryRequest{
		Account: u,
		Limit:   20,
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == nil {
		return nil, errors.New("no status")
	}
	if resp.Status != nil && resp.Status.Code != 0 {
		return nil, errors.New(resp.Status.Message)
	}

	return resp.Status, nil
}

func balance(addr string) (Worker, error) {
	cl := NewClient(addr)
	return &Balance{cl: apipb.NewAPIServiceHTTPClient(cl)}, nil
}

type Balance struct {
	cl apipb.APIServiceInterface
}

func (w *Balance) Request(ctx context.Context, u uint64) (interface{}, error) {
	resp, err := w.cl.GetBalance(ctx, &apipb.GetBalanceRequest{
		Account: u,
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == nil {
		return nil, errors.New("no status")
	}
	if resp.Status != nil && resp.Status.Code != 0 {
		return nil, errors.New(resp.Status.Message)
	}

	return resp.Status, nil
}

func keygen(addr string) (Worker, error) {
	cl := NewClient(addr)
	return &Keygen{cl: apipb.NewAPIServiceHTTPClient(cl)}, nil
}

type Keygen struct {
	cl apipb.APIServiceInterface
}

func (w *Keygen) Request(ctx context.Context, u uint64) (interface{}, error) {
	priv, err := btcec.NewPrivateKey(curve)
	if err != nil {
		return nil, err
	}

	pub := priv.PubKey()
	pubb := pt.PublicKey(pub.SerializeHybrid()).String()

	s, err := w.cl.GetLastSettings(context.TODO(), &apipb.GetLastSettingsRequest{Account: u})
	if err != nil {
		return nil, err
	}

	sreq := &apipb.SettingsRequest{
		Account:            u,
		PrevHash:           s.Hash,
		DataHash:           s.DataHash,
		VerifyTransferSign: s.VerifyTransferSign,
		PublicKey:          pubb, // changed field
	}

	resp, err := w.updateSettings(sreq)
	if err != nil {
		return nil, err
	}
	if resp.Status != nil && resp.Status.Code != 0 {
		return nil, errors.New(resp.Status.Message)
	}

	err = client.SavePrivateKey(u, priv)
	if err != nil {
		//	log.Printf("Can't save key to db, but it's already written. Remember it!! %x", hex.EncodeToString(priv.Serialize()))
		return nil, err
	}

	return resp.Status, nil
}

func (w *Keygen) updateSettings(sreq *apipb.SettingsRequest) (*apipb.SettingsResponse, error) {
	prv, err := client.LoadPrivateKey(sreq.Account)
	if err != nil {
		return nil, err
	}

	if prv != nil {
		hash := client.SettingsRequestHash(sreq)
		sign, err := pt.SignTransfer(hash, prv)
		if err != nil {
			return nil, err
		}
		sreq.Sign = sign.String()
	}

	resp, err := w.cl.UpdateSettings(context.TODO(), sreq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func getreq(addr string) (Worker, error) {
	return GetReq{addr: addr}, nil
}

type GetReq struct {
	addr string
}

func (w GetReq) Request(ctx context.Context, s uint64) (interface{}, error) {
	c, _, err := fasthttp.Get(nil, w.addr)
	if err != nil {
		return nil, err
	}
	if c != fasthttp.StatusOK {
		return nil, errors.Errorf("status code: %v", c)
	}
	return nil, nil
}

func dbput(addr string) (Worker, error) {
	cl := tcprpc.NewClient(addr)
	return DBPut{addr: addr, cl: pusherpb.NewTCPRPCPusherServiceClient(cl, *prefix)}, nil
}

type DBPut struct {
	addr string
	cl   pusherpb.TCPRPCPusherServiceClient
}

func (w DBPut) Request(ctx context.Context, s uint64) (interface{}, error) {
	resp, err := w.cl.Push(ctx, &pusherpb.PushRequest{
		Txns: rndBatch(s),
	})
	if err != nil {
		return nil, err
	}
	if resp.Status == nil {
		return nil, errors.New("no status")
	}
	if resp.Status != nil && resp.Status.Code != 0 {
		return nil, errors.New(resp.Status.Message)
	}
	return resp.Status, nil
}

func hostname(addr string) (Worker, error) {
	cl := NewClient(addr)
	return Hostname{Client: cl}, nil
}

type Hostname struct {
	*graceful.Client
}

func (w Hostname) Request(ctx context.Context, s uint64) (interface{}, error) {
	return nil, w.Client.Call(ctx, "hostname", nil, nil)
}

func nonereq(addr string) (Worker, error) {
	return NoneWorker{}, nil
}

type NoneWorker struct{}

func (w NoneWorker) Request(ctx context.Context, s uint64) (interface{}, error) { return nil, nil }

func blockreq(addr string) (Worker, error) {
	return BlockWorker{}, nil
}

type BlockWorker struct{}

func (w BlockWorker) Request(ctx context.Context, s uint64) (interface{}, error) {
	<-stopc
	return nil, nil
}

var pow uint

func genaccaunts() {
	pow = 1
	for (1 << pow) < *accs {
		pow++
	}

	acclist = make([]uint64, *accs)
	accmap = make(map[uint64]struct{}, *accs)

	for len(accmap) < *accs {
		a := genacc()
		if _, ok := accmap[a]; ok {
			continue
		}

		acclist[len(accmap)] = a
		accmap[a] = struct{}{}
	}
}

func genacc() uint64 {
	return uint64(rand.Int() << (64 - pow)) // means 2 ** pow possible accounts
}

func rndacc() uint64 {
	return acclist[rand.Intn(len(acclist))]
}

func seqacc() func() uint64 {
	seqaccc := make(chan uint64)
	go func() {
		defer close(seqaccc)
		for {
			for _, a := range acclist {
				select {
				case seqaccc <- a:
				case <-stopc:
					return
				}
			}
		}
	}()

	return func() uint64 {
		return <-seqaccc
	}
}

func rndlimacc(lim int) func() uint64 {
	sublist := make([]uint64, lim)
	subset := make(map[uint64]struct{}, lim)

	for len(subset) < lim {
		a := rndacc()
		if _, ok := subset[a]; ok {
			continue
		}

		sublist[len(subset)] = a
		subset[a] = struct{}{}
	}

	subset = nil

	return func() uint64 {
		return sublist[rand.Intn(len(sublist))]
	}
}

func buildBreaker() *breaker.Breaker {
	// TODO
	//	return breaker.New(1, 1, time.Second*5)
	return nil
}

func NewClient(addr string) *graceful.Client {
	g, err := graceful.NewClient(addr, codec, buildBreaker())
	if err != nil {
		panic(err)
	}
	return g
}

func rndBatch(s uint64) []*chainpb.Txn {
	b := make([]*chainpb.Txn, *bs)
	for i := range b {
		b[i] = rndTxn(s)
	}
	if *kgen {
		b[0].Sign = make([]byte, 200)
		_, _ = rand.Read(b[0].Sign)
	}
	return b
}

func rndTxn(s uint64) *chainpb.Txn {
	t := &chainpb.Txn{Sender: s, Receiver: acc(), ID: uint64(rand.Int()), Amount: int64(rand.Int63n(1000000000000)), Balance: int64(rand.Int63n(10000000000000))}
	_, _ = rand.Read(t.PrevHash[:])
	_, _ = rand.Read(t.Hash[:])
	return t
}
