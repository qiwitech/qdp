package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/facebookgo/flagenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qiwitech/tcprpc"
	"golang.org/x/net/trace"

	"github.com/qiwitech/qdp/bigchain"
	"github.com/qiwitech/qdp/chain"
	"github.com/qiwitech/qdp/gate"
	"github.com/qiwitech/qdp/preloader"
	"github.com/qiwitech/qdp/processor"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
	"github.com/qiwitech/qdp/proto/pusherpb"
	"github.com/qiwitech/qdp/pt"
	"github.com/qiwitech/qdp/pusher"
	"github.com/qiwitech/qdp/pusher/remotepusher"
	"github.com/qiwitech/qdp/pusher/seqpusher"
	"github.com/qiwitech/qdp/router"
)

var (
	nodes    = flag.String("nodes", ":31337", "cluster nodes")
	selfAddr = flag.String("self", "", "self address for router")
	listen   = flag.String("listen", ":31337", "http addr")

	discoverSvc = flag.String("discover", "", "discover swarm service:port to update router")

	dbAddr = flag.String("db", "", "DB addr")

	pushTo = flag.String("push", "", "comma separated addresses to push to")

	threads    = flag.Int("threads", 997, "threads for processor")
	routerType = flag.String("router", "static", "type of router (simple|static)")
)

var (
	Version = "HEAD"
)

func getSelfAddr() string {
	_, port, err := net.SplitHostPort(*listen)
	if err != nil {
		panic(err)
	}

	return net.JoinHostPort(getExternalHostname(), port)
}

func getExternalHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return hostname
			}
			return string(ip)
		}
	}
	return hostname
}

func init() {
	flagenv.Prefix = "PLUTOS_"
	flagenv.Parse()
}

func main() {
	flag.Parse()

	trace.AuthRequest = func(req *http.Request) (any, sens bool) {
		return true, true
	}

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Start server on %v. Version: %s\n", lis.Addr(), Version)

	server := tcprpc.NewServer()
	//	server.UseShutdownMiddleware(http.StatusServiceUnavailable, []byte("server is going to shutdown\n"))

	if *selfAddr == "" {
		*selfAddr = *listen
	}

	hostname := *selfAddr
	if os.Getenv("SWARM") != "" {
		hostname = getSelfAddr()
	}

	fmt.Printf("nodes: %q, self: %q, db addr: %q, additional pusher: %q, router type: %q\n", *nodes, hostname, *dbAddr, *pushTo, *routerType)

	var r pt.Router
	switch *routerType {
	case "failover":
		panic("unsupported")
	case "simple":
		r = router.New(hostname)
	case "static":
		r = router.NewStatic(hostname)
	default:
		panic("undefined router")
	}
	if *nodes != "" {
		r.SetNodes(strings.Split(*nodes, ","))
	}

	if ur, ok := r.(router.UpdatableRouter); ok {
		log.Printf("%T is updatable router, set /cfg/router handler", r)
		http.Handle("/cfg/router", router.Handler(ur))
		if *discoverSvc != "" {
			time.AfterFunc(5*time.Second, func() {
				if err := router.UpdateRouter(ur, *discoverSvc); err != nil {
					log.Printf("router update: %v", err)
				}
			})
		}
	} else {
		log.Printf("%T is not updatable router", r)
	}

	c := chain.NewChain()

	var p pt.TransferProcessor
	if *threads > 1 {
		p = processor.NewMultiprocessor(c, *threads)
	} else {
		p = processor.NewProcessor(c)
	}

	sc := chain.NewSettingsChain()
	sp := processor.NewSettingsProcessor(sc)

	var (
		pushers  []pt.Pusher
		spushers []pt.SettingsPusher
	)

	if *dbAddr != "" {
		dburl := *dbAddr
		db := remotepusher.NewDBPusher(dburl)

		prel := preloader.New(c, sc, newBigchain(dburl))

		p.SetPreloader(prel)
		sp.SetPreloader(prel)

		pushers = append(pushers, db)
		spushers = append(spushers, db)
	}

	if *pushTo != "" {
		for _, p := range strings.Split(*pushTo, ",") {
			if p == "" {
				continue
			}

			dburl := p
			db := remotepusher.NewDBPusher(dburl)
			pushers = append(pushers, db)
			spushers = append(spushers, db)
		}
	}

	pushers = append(pushers, remotepusher.NewRoutedPusher(r))

	if len(pushers) == 1 {
		p.SetPusher(pushers[0])
	} else if len(pushers) > 1 {
		p.SetPusher(seqpusher.New(pushers...))
	}

	if len(spushers) == 1 {
		sp.SetPusher(spushers[0])
	} else if len(pushers) > 1 {
		sp.SetPusher(seqpusher.NewSettings(spushers...))
	}

	p.SetSettingsChain(sc)

	g := gate.NewGate(p, sp)
	g.SetRouter(r)

	ps := remotepusher.NewService(pusher.NewChainReceiversPusher(c))

	tcprpc.RegisterHostnameHandler()
	http.Handle("/metrics", promhttp.Handler())
	server.HandleHTTP(http.DefaultServeMux)

	gatepb.RegisterProcessorServiceHandlers(server, "v1/", g)
	pusherpb.RegisterPusherServiceHandlers(server, "v1/", ps)

	if err := server.Serve(lis); err != nil {
		// skip closing server error
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}

func newBigchain(baseurl string) pt.BigChain {
	g := tcprpc.NewClient(baseurl)
	cl := plutodbpb.NewTCPRPCPlutoDBServiceClient(g, "v1/")
	return bigchain.New(cl)
}
