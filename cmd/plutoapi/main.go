package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/eapache/go-resiliency/breaker"
	"github.com/facebookgo/flagenv"
	"github.com/qiwitech/graceful"
	"github.com/qiwitech/tcprpc"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/trace"

	"github.com/qiwitech/qdp/api"
	"github.com/qiwitech/qdp/proto/apipb"
	"github.com/qiwitech/qdp/proto/gatepb"
	"github.com/qiwitech/qdp/proto/metadbpb"
	"github.com/qiwitech/qdp/proto/plutodbpb"
	"github.com/qiwitech/qdp/pt"
	"github.com/qiwitech/qdp/router"
)

var (
	// TODO(outself): validate!
	gate   = flag.String("gate", ":31337", "gate url")
	pdb    = flag.String("plutodb", ":38388", "plutodb url")
	mdb    = flag.String("metadb", "", "metadb url")
	listen = flag.String("listen", ":9090", "http addr")
	simple = flag.Bool("simple-router", false, "use simple router instead of static")
)
var (
	Version = "dev"
)

func ServiceMeta(backend string, version string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Backend", backend)
			w.Header().Set("X-Backend-Version", version)
			h.ServeHTTP(w, req)
		})
	}
}

/*
func FastServiceMeta(backend string, version string) func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(req *fasthttp.RequestCtx) {
			req.Response.Header.Set("X-Backend", backend)
			req.Response.Header.Set("X-Backend-Version", version)
			h(req)
		}
	}
}
*/

func TimingHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func(now time.Time) {
			w.Header().Set("X-Timing", time.Since(now).String())
		}(time.Now())
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

var (
	serverCodec graceful.Codec = &graceful.JSONCodec{}
)

func buildBreaker() *breaker.Breaker {
	// TODO
	//	return breaker.New(1, 1, time.Second*5)
	return nil
}

func init() {
	flagenv.Prefix = "PLUTOAPI_"
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

	fmt.Printf("Start server on %v\n", lis.Addr())

	server := graceful.NewServer()
	//	server.UseShutdownMiddleware(http.StatusServiceUnavailable, []byte("server is going to shutdown\n"))
	//server.Use(middleware.Logger)
	//	server.Use(middleware.Heartbeat("/"))
	//server.Use(TimingHandler)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	server.Use(ServiceMeta(hostname, Version))

	/*
		go func() {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT)
			signal.Notify(c, syscall.SIGTERM)

			<-c
			server.Shutdown(context.Background(), 0)
		}()
	*/

	clientFn := func(baseurl string) gatepb.ProcessorServiceInterface {
		g := tcprpc.NewClient(baseurl)
		return gatepb.NewTCPRPCProcessorServiceClient(g, "v1/")
	}

	var r pt.Router
	if *simple {
		r = router.New("")
	} else {
		r = router.NewStatic("")
	}

	// make routing table from address
	nodes := []string{"0=" + *gate}
	r.SetNodes(nodes)

	a := api.NewService(api.NewRouter(r, clientFn))

	if *pdb != "" {
		g := tcprpc.NewClient(*pdb)
		plutodb := plutodbpb.NewTCPRPCPlutoDBServiceClient(g, "v1/")

		a.SetPlutoDBClient(plutodb)
	}

	if *mdb != "" {
		g := tcprpc.NewClient(*mdb)
		metadb := metadbpb.NewTCPRPCMetaDBServiceClient(g, "v1/")

		a.SetMetaDBClient(metadb)
	}

	handler := apipb.NewAPIServiceHandler(a, serverCodec)

	//	server.Mount("/", fasthttpadaptor.NewHTTPHandler(http.DefaultServeMux))
	http.Handle("/metrics", prometheus.Handler())
	server.Mount("/", http.DefaultServeMux)
	server.Mount("/v1/", handler)
	//	server.Handle("/debug/expvar", expvar.Handler())

	if err := server.Serve(lis); err != nil {
		// skip closing server error
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}
